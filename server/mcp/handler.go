package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"entgo.io/ent/dialect/sql"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	credentialmodel "github.com/simpledms/simpledms/model/main/mcpcredential"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
)

type requestContextKey int

type requestContext struct {
	source context.Context
	origin string
}

const maxUploadBytes int64 = 10 * 1024 * 1024

type Handler struct {
	config      Config
	credentials *credentialmodel.CredentialService
	transport   *sdk.StreamableHTTPHandler
}

func NewHandler(config Config) *Handler {
	handler := &Handler{
		config:      config,
		credentials: credentialmodel.NewCredentialService(),
	}
	server := sdk.NewServer(&sdk.Implementation{
		Name:    "SimpleDMS",
		Version: "1",
	}, nil)
	registerRead(server, handler, "get_space", "Identify the authorized Space.", handler.getSpace)
	registerRead(server, handler, "list_inbox", "List or search Inbox documents.", handler.listInbox)
	registerRead(server, handler, "get_file", "Read a live file's metadata.", handler.getFile)
	registerRead(server, handler, "read_file_text", "Read bounded existing OCR text.", handler.readText)
	registerWrite(server, handler, "upload_file", "Upload one document into Inbox.", handler.uploadFile)
	handler.transport = sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server {
		return server
	}, &sdk.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		MaxRequestBodyBytes:          16 * 1024 * 1024,
		PropagateRequestCancellation: true,
	})
	return handler
}

func (qq *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Cache-Control", "no-store")
	scheme := "http"
	if req.TLS != nil || qq.trustedHTTPS(req) {
		scheme = "https"
	}
	if !qq.config.DevMode && scheme != "https" {
		rw.WriteHeader(http.StatusForbidden)
		return
	}
	if origin := req.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme != scheme || !strings.EqualFold(parsed.Host, req.Host) ||
			parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			rw.WriteHeader(http.StatusForbidden)
			return
		}
	}
	authorize := func(*ctxx.SpaceContext, *entmain.MCPCredential) error {
		return nil
	}
	_, err := qq.execute(req.Context(), bearer(req.Header), authorize)
	if err != nil {
		status := http.StatusInternalServerError
		var httpErr *e.HTTPError
		if errors.As(err, &httpErr) {
			status = httpErr.StatusCode()
		}
		if status == http.StatusUnauthorized {
			rw.Header().Set("WWW-Authenticate", `Bearer realm="SimpleDMS MCP"`)
		}
		rw.WriteHeader(status)
		return
	}
	origin := qq.config.Infra.SystemConfig().AbsoluteURL("/")
	if origin == "/" {
		origin = scheme + "://" + req.Host
	}
	ctx := context.WithValue(req.Context(), requestContextKey(0), requestContext{
		source: req.Context(),
		origin: strings.TrimSuffix(origin, "/"),
	})
	qq.transport.ServeHTTP(rw, req.WithContext(ctx))
}

func (qq *Handler) trustedHTTPS(req *http.Request) bool {
	if !strings.EqualFold(req.Header.Get("X-Forwarded-Proto"), "https") {
		return false
	}
	address, err := netip.ParseAddrPort(req.RemoteAddr)
	if err != nil {
		return false
	}
	for _, prefix := range qq.config.TrustedProxies {
		if prefix.Contains(address.Addr()) {
			return true
		}
	}
	return false
}

func (qq *Handler) execute(
	ctx context.Context, token string, fn func(*ctxx.SpaceContext, *entmain.MCPCredential) error,
) (bool, error) {
	return qq.credentials.Execute(
		ctx, qq.config.MainDB, qq.config.TenantDBs, qq.config.I18n,
		qq.config.Infra.SystemConfig().CommercialLicenseEnabled(), token, fn,
	)
}

func bearer(header http.Header) string {
	parts := strings.Fields(header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func registerRead[I, O any](
	server *sdk.Server,
	handler *Handler,
	name, description string,
	fn func(context.Context, *ctxx.SpaceContext, *entmain.MCPCredential, I) (O, error),
) {
	registerTool(server, handler, name, description, true, fn)
}

func registerWrite[I, O any](
	server *sdk.Server,
	handler *Handler,
	name, description string,
	fn func(context.Context, *ctxx.SpaceContext, *entmain.MCPCredential, I) (O, error),
) {
	registerTool(server, handler, name, description, false, fn)
}

func registerTool[I, O any](
	server *sdk.Server,
	handler *Handler,
	name, description string,
	isReadOnly bool,
	fn func(context.Context, *ctxx.SpaceContext, *entmain.MCPCredential, I) (O, error),
) {
	closedWorld := false
	sdk.AddTool(server, &sdk.Tool{
		Name:        name,
		Description: description,
		Annotations: &sdk.ToolAnnotations{
			ReadOnlyHint:  isReadOnly,
			OpenWorldHint: &closedWorld,
		},
	}, func(ctx context.Context, req *sdk.CallToolRequest, input I) (*sdk.CallToolResult, O, error) {
		var output O
		if req.Extra == nil {
			return nil, output, safeError(e.NewHTTPErrorf(http.StatusUnauthorized, "Invalid MCP credential."))
		}
		// Older SDK protocol paths detach cancellation; preserve the HTTP request's lifetime.
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		if request, ok := ctx.Value(requestContextKey(0)).(requestContext); ok {
			stop := context.AfterFunc(request.source, cancel)
			defer stop()
		}
		execute := func(sc *ctxx.SpaceContext, cred *entmain.MCPCredential) error {
			if !isReadOnly && cred.IsReadOnly {
				return e.NewHTTPErrorf(http.StatusForbidden, "MCP credential cannot write.")
			}
			var err error
			output, err = fn(ctx, sc, cred, input)
			return err
		}
		_, err := handler.execute(ctx, bearer(req.Extra.Header), execute)
		if err != nil {
			var zero O
			return nil, zero, safeError(err)
		}
		return nil, output, nil
	})
}

func safeError(err error) error {
	code, message := "internal_error", "Could not complete the operation."
	var httpErr *e.HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode() {
		case http.StatusBadRequest, http.StatusRequestEntityTooLarge:
			code = "invalid_input"
		case http.StatusUnauthorized, http.StatusForbidden:
			code = "forbidden"
		case http.StatusNotFound:
			code = "not_found"
		case http.StatusServiceUnavailable:
			code = "unavailable"
		}
		if code != "internal_error" {
			message = httpErr.Message()
		}
	}
	log.Printf("MCP tool error: %s (%T)", code, err)
	data, marshalErr := json.Marshal(map[string]string{"code": code, "message": message})
	if marshalErr != nil {
		log.Println(marshalErr)
		return errors.New("internal_error")
	}
	return errors.New(string(data))
}

func (qq *Handler) getSpace(
	_ context.Context, ctx *ctxx.SpaceContext, credential *entmain.MCPCredential, _ struct{},
) (SpaceData, error) {
	return SpaceData{
		TenantID:        ctx.TenantID,
		TenantName:      ctx.Tenant.Name,
		SpaceID:         ctx.SpaceID,
		Name:            ctx.Space.Name,
		Description:     ctx.Space.Description,
		FolderMode:      ctx.Space.IsFolderMode,
		RootDirectoryID: ctx.SpaceRootDir().PublicID.String(),
		ReadOnly:        credential.IsReadOnly,
	}, nil
}

func (qq *Handler) listInbox(
	requestCtx context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input InboxInput,
) (InboxData, error) {
	result := InboxData{Files: []FileSummary{}}
	limit := 50
	if input.Limit != nil {
		limit = *input.Limit
	}
	if limit < 1 || limit > 100 || input.Offset < 0 || input.Offset > 1000000 ||
		utf8.RuneCountInString(input.Query) > 300 || len(input.Sources) > 16 {
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid page or search range.")
	}
	query, err := filemodel.NewInboxQuery().Query(ctx, input.Query, input.Sort, input.Sources)
	if err != nil {
		return result, err
	}
	files, err := query.WithParent(func(query *enttenant.FileQuery) {
		query.Select(file.FieldPublicID)
	}).Select(
		file.FieldID, file.FieldPublicID, file.FieldName, file.FieldIsDirectory,
		file.FieldIsInInbox, file.FieldSource, file.FieldOcrSuccessAt,
	).Offset(input.Offset).Limit(limit + 1).All(ctx)
	if err != nil {
		return result, err
	}
	result.HasMore = len(files) > limit
	if result.HasMore {
		files = files[:limit]
		next := input.Offset + limit
		result.NextOffset = &next
	}
	for _, filex := range files {
		result.Files = append(result.Files, qq.summary(requestCtx, ctx, filex, filex.Edges.Parent))
	}
	return result, nil
}

func (qq *Handler) summary(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	filex *enttenant.File,
	parent *enttenant.File,
) FileSummary {
	parentID := ""
	if parent != nil {
		parentID = parent.PublicID.String()
	}
	url := qq.documentURL(requestCtx, ctx, parentID, filex.PublicID.String(), filex.IsDirectory)
	return FileSummary{
		FileID:       filex.PublicID.String(),
		URL:          url,
		Name:         filex.Name,
		IsDirectory:  filex.IsDirectory,
		IsInInbox:    filex.IsInInbox,
		Source:       filex.Source.String(),
		OCRAvailable: filex.OcrSuccessAt != nil && !filex.OcrSuccessAt.IsZero(),
	}
}

func (qq *Handler) documentURL(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	parentID, fileID string,
	isDirectory bool,
) string {
	path := route.Browse(ctx.TenantID, ctx.SpaceID, fileID)
	if !isDirectory && parentID != "" {
		path = route.BrowseFile(ctx.TenantID, ctx.SpaceID, parentID, fileID)
	}
	url := qq.config.Infra.SystemConfig().AbsoluteURL(path)
	if url == path {
		if request, ok := requestCtx.Value(requestContextKey(0)).(requestContext); ok {
			url = request.origin + path
		}
	}
	return url
}

func (qq *Handler) fileData(
	requestCtx context.Context, ctx *ctxx.SpaceContext, filex *enttenant.File,
) (FileData, error) {
	var parent *enttenant.File
	if filex.ParentID != 0 {
		var err error
		parent, err = filex.QueryParent().Only(ctx)
		if err != nil && !enttenant.IsNotFound(err) {
			return FileData{}, err
		}
	}
	data := FileData{FileSummary: qq.summary(requestCtx, ctx, filex, parent)}
	if parent != nil {
		data.ParentID = parent.PublicID.String()
	}
	if filex.IsDirectory {
		return data, nil
	}
	version, err := filex.QueryFileVersions().Order(
		fileversion.ByVersionNumber(sql.OrderDesc()),
	).WithStoredFile().First(ctx)
	if enttenant.IsNotFound(err) {
		return data, nil
	}
	if err != nil {
		return data, err
	}
	data.VersionNumber = version.VersionNumber
	if stored := version.Edges.StoredFile; stored != nil {
		data.Size = stored.Size
		data.MIMEType = stored.MimeType
		data.ContentSHA256 = stored.ContentSha256
	}
	return data, nil
}

func (qq *Handler) getFile(
	requestCtx context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input FileInput,
) (FileData, error) {
	if input.FileID == "" || len(input.FileID) > 100 {
		return FileData{}, e.NewHTTPErrorf(http.StatusBadRequest, "File ID is required.")
	}
	filex, err := filemodel.NewFileReader().Get(ctx, input.FileID)
	if err != nil {
		return FileData{}, err
	}
	return qq.fileData(requestCtx, ctx, filex)
}

func (qq *Handler) readText(
	requestCtx context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input TextInput,
) (TextData, error) {
	result := TextData{FileID: input.FileID}
	length := 12000
	if input.Length != nil {
		length = *input.Length
	}
	if input.FileID == "" || len(input.FileID) > 100 || input.Offset < 0 ||
		length < 1 || length > 50000 {
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid text range.")
	}
	reader := filemodel.NewFileReader()
	filex, err := reader.Get(ctx, input.FileID)
	if err != nil {
		return result, err
	}
	if filex.IsDirectory {
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "File is a directory.")
	}
	data, err := qq.fileData(requestCtx, ctx, filex)
	if err != nil {
		return result, err
	}
	result.VersionNumber = data.VersionNumber
	result.Available = data.OCRAvailable
	if result.Available {
		result.Text, result.HasMore, err = reader.TextWindow(filex.OcrContent, input.Offset, length)
		if result.HasMore {
			next := input.Offset + length
			result.NextOffset = &next
		}
	}
	return result, err
}

func (qq *Handler) uploadFile(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	credential *entmain.MCPCredential,
	input UploadFileInput,
) (UploadFileData, error) {
	var result UploadFileData
	if input.Filename == "." || filepath.Clean(input.Filename) != input.Filename ||
		!filenamex.IsAllowed(input.Filename) {
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid filename.")
	}
	if input.ContentBase64 == "" {
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "Upload is empty.")
	}
	if strings.ContainsAny(input.ContentBase64, "\r\n") {
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid base64 content.")
	}

	limit := maxUploadBytes
	configuredLimit, err := qq.config.Infra.FileSystem().NilableEffectiveUploadSizeLimitBytes(ctx)
	if err != nil {
		return result, err
	}
	if configuredLimit != nil && *configuredLimit < limit {
		limit = *configuredLimit
	}
	if len(input.ContentBase64) > base64.StdEncoding.EncodedLen(int(limit)) {
		return result, e.NewHTTPErrorf(http.StatusRequestEntityTooLarge, "Upload is too large.")
	}

	decoded := base64.NewDecoder(
		base64.StdEncoding.Strict(), strings.NewReader(input.ContentBase64),
	)
	expectedBytes, err := io.Copy(io.Discard, io.LimitReader(decoded, limit+1))
	if err != nil {
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid base64 content.")
	}
	if expectedBytes == 0 {
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "Upload is empty.")
	}
	if expectedBytes > limit {
		return result, e.NewHTTPErrorf(http.StatusRequestEntityTooLarge, "Upload is too large.")
	}

	mainCheck := func(checkCtx context.Context, tx *entmain.Tx) error {
		return qq.credentials.AuthorizeFinalization(checkCtx, tx, credential)
	}
	ingested, err := filesystem.NewFileIngestionService(qq.config.Infra.FileSystem()).Ingest(
		ctx,
		base64.NewDecoder(base64.StdEncoding.Strict(), strings.NewReader(input.ContentBase64)),
		input.Filename,
		ctx.SpaceRootDir().ID,
		true,
		filesource.MCP,
		&expectedBytes,
		mainCheck,
	)
	if err != nil {
		return result, err
	}
	return UploadFileData{
		FileID: ingested.FilePublicID,
		URL: qq.documentURL(
			requestCtx, ctx, ctx.SpaceRootDir().PublicID.String(), ingested.FilePublicID, false,
		),
		Filename:  ingested.Filename,
		Size:      ingested.Size,
		IsInInbox: ingested.IsInInbox,
	}, nil
}
