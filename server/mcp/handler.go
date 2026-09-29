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
	"github.com/simpledms/simpledms/db/enttenant/attribute"
	documenttypequery "github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/resolvedtagassignment"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/attributetype"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	credentialmodel "github.com/simpledms/simpledms/model/main/mcpcredential"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	filingmodel "github.com/simpledms/simpledms/model/tenant/filing"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/model/tenant/tenantdatamigration"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
	"github.com/simpledms/simpledms/util/timex"
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
	registerRead(
		server, handler, "read_file_text", "Read bounded existing OCR text.", handler.readText,
	)
	registerStorageWrite(
		server, handler, "upload_file", "Upload one document into Inbox.", handler.uploadFile,
	)
	registerRead(server, handler, "list_tags", "List existing Tags.", handler.listTags)
	registerRead(server, handler, "list_properties", "List existing fields.", handler.listProperties)
	registerRead(
		server, handler, "list_document_types", "List document types.", handler.listDocumentTypes,
	)
	registerRead(
		server, handler, "get_document_type", "Read a document type.", handler.getDocumentType,
	)
	registerWrite(server, handler, "assign_tag", "Assign a Tag to a document.", handler.assignTag)
	registerWrite(server, handler, "unassign_tag", "Unassign a direct Tag.", handler.unassignTag)
	registerWrite(
		server, handler, "set_file_property", "Set a typed field value.", handler.setFileProperty,
	)
	registerWrite(
		server, handler, "remove_file_property", "Remove a field value.", handler.removeFileProperty,
	)
	registerWrite(
		server, handler, "set_document_type", "Set a document type.", handler.setDocumentType,
	)
	registerWrite(
		server, handler, "clear_document_type", "Clear a document type.", handler.clearDocumentType,
	)
	registerRead(
		server, handler, "list_directory", "List one filing directory.", handler.listDirectory,
	)
	registerWrite(
		server, handler, "create_directory", "Create a filing directory.", handler.createDirectory,
	)
	registerWrite(
		server, handler, "file_inbox_document", "File one Inbox document.",
		handler.fileInboxDocument,
	)
	registerWrite(
		server, handler, "mark_inbox_file_done", "Complete an Inbox document without moving it.",
		handler.markInboxFileDone,
	)
	registerRead(
		server, handler, "search_files", "Search or list filed documents.", handler.searchFiles,
	)
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
	_, err := qq.execute(req.Context(), bearer(req.Header), true, authorize)
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
	ctx context.Context,
	token string,
	isTenantTxReadOnly bool,
	fn func(*ctxx.SpaceContext, *entmain.MCPCredential) error,
) (bool, error) {
	commercial := qq.config.Infra.SystemConfig().CommercialLicenseEnabled()
	if isTenantTxReadOnly {
		return qq.credentials.Execute(
			ctx, qq.config.MainDB, qq.config.TenantDBs, qq.config.I18n, commercial, token, fn,
		)
	}
	return qq.credentials.ExecuteWrite(
		ctx, qq.config.MainDB, qq.config.TenantDBs, qq.config.I18n, commercial, token, fn,
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
	registerTool(server, handler, name, description, true, true, fn)
}

func registerWrite[I, O any](
	server *sdk.Server,
	handler *Handler,
	name, description string,
	fn func(context.Context, *ctxx.SpaceContext, *entmain.MCPCredential, I) (O, error),
) {
	registerTool(server, handler, name, description, false, false, fn)
}

func registerStorageWrite[I, O any](
	server *sdk.Server,
	handler *Handler,
	name, description string,
	fn func(context.Context, *ctxx.SpaceContext, *entmain.MCPCredential, I) (O, error),
) {
	registerTool(server, handler, name, description, false, true, fn)
}

func registerTool[I, O any](
	server *sdk.Server,
	handler *Handler,
	name, description string,
	isReadOnly bool,
	isTenantTxReadOnly bool,
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
			return nil, output, safeError(e.NewHTTPErrorf(
				http.StatusUnauthorized, "Invalid MCP credential.",
			))
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
		_, err := handler.execute(ctx, bearer(req.Extra.Header), isTenantTxReadOnly, execute)
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
	if err := qq.addClassification(ctx, filex, &data); err != nil {
		return FileData{}, err
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

func (qq *Handler) listDirectory(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input DirectoryInput,
) (DirectoryData, error) {
	result := DirectoryData{Children: []FileSummary{}}
	limit, err := pageLimit(input.Offset, input.Limit)
	if err != nil {
		return result, err
	}
	if len(input.DirectoryID) > 100 {
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid directory ID.")
	}
	directory, children, hasMore, err := filingmodel.NewFilingService(
		qq.config.Infra.FileSystem(),
	).ListDirectory(ctx, input.DirectoryID, input.Offset, limit)
	if err != nil {
		return result, err
	}
	result.DirectoryID = directory.PublicID.String()
	result.Name = directory.Name
	if directory.ParentID != 0 {
		parent, err := directory.QueryParent().Only(ctx)
		if err != nil {
			return DirectoryData{}, err
		}
		result.ParentID = parent.PublicID.String()
	}
	result.HasMore = hasMore
	if hasMore {
		next := input.Offset + limit
		result.NextOffset = &next
	}
	for _, child := range children {
		result.Children = append(result.Children, qq.summary(requestCtx, ctx, child, directory))
	}
	return result, nil
}

func (qq *Handler) createDirectory(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input CreateDirectoryInput,
) (DirectoryMutationData, error) {
	if input.ParentDirectoryID == "" || len(input.ParentDirectoryID) > 100 ||
		input.Name == "" || utf8.RuneCountInString(input.Name) > 255 {
		return DirectoryMutationData{}, e.NewHTTPErrorf(
			http.StatusBadRequest,
			"Parent directory ID and name are required.",
		)
	}
	directory, err := filingmodel.NewFilingService(qq.config.Infra.FileSystem()).CreateDirectory(
		ctx, input.ParentDirectoryID, input.Name,
	)
	if err != nil {
		return DirectoryMutationData{}, err
	}
	return DirectoryMutationData{
		DirectoryID: directory.PublicID.String(),
		Name:        directory.Name,
		ParentID:    input.ParentDirectoryID,
	}, nil
}

func (qq *Handler) fileInboxDocument(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input FileInboxDocumentInput,
) (FilingData, error) {
	if input.FileID == "" || len(input.FileID) > 100 ||
		input.DestinationDirectoryID == "" || len(input.DestinationDirectoryID) > 100 ||
		utf8.RuneCountInString(input.Filename) > 255 ||
		utf8.RuneCountInString(input.NewDirectoryName) > 255 {
		return FilingData{}, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid filing input.")
	}
	filex, err := filingmodel.NewFilingService(qq.config.Infra.FileSystem()).FileInboxDocument(
		ctx,
		input.FileID,
		input.DestinationDirectoryID,
		input.Filename,
		input.NewDirectoryName,
	)
	if err != nil {
		return FilingData{}, err
	}
	return filingProjection(ctx, filex)
}

func (qq *Handler) markInboxFileDone(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input FileInput,
) (FilingData, error) {
	if input.FileID == "" || len(input.FileID) > 100 {
		return FilingData{}, e.NewHTTPErrorf(http.StatusBadRequest, "File ID is required.")
	}
	filex, err := filingmodel.NewFilingService(qq.config.Infra.FileSystem()).CompleteInboxFile(
		ctx, input.FileID,
	)
	if err != nil {
		return FilingData{}, err
	}
	return filingProjection(ctx, filex)
}

func filingProjection(ctx *ctxx.SpaceContext, filex *enttenant.File) (FilingData, error) {
	result := FilingData{
		FileID:    filex.PublicID.String(),
		Name:      filex.Name,
		IsInInbox: filex.IsInInbox,
	}
	if filex.ParentID == 0 {
		return result, nil
	}
	parent, err := filex.QueryParent().Only(ctx)
	if err != nil {
		return FilingData{}, err
	}
	result.ParentID = parent.PublicID.String()
	return result, nil
}

func (qq *Handler) searchFiles(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input SearchFilesInput,
) (FileListData, error) {
	result := FileListData{Files: []FileSummary{}}
	limit, err := pageLimit(input.Offset, input.Limit)
	if err != nil {
		return result, err
	}
	if utf8.RuneCountInString(input.Query) > 300 || len(input.TagIDs) > 32 ||
		len(input.DocumentTypeID) > 100 {
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid search input.")
	}
	tagIDs, documentTypeID, err := resolveFiledFilters(ctx, input)
	if err != nil {
		return result, err
	}
	query, err := filemodel.NewFiledQuery().Query(
		ctx, input.Query, input.Sort, tagIDs, documentTypeID,
	)
	if err != nil {
		return result, err
	}
	files, err := query.WithParent(func(query *enttenant.FileQuery) {
		query.Select(file.FieldPublicID)
	}).Select(
		file.FieldID,
		file.FieldPublicID,
		file.FieldName,
		file.FieldIsDirectory,
		file.FieldIsInInbox,
		file.FieldSource,
		file.FieldOcrSuccessAt,
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

func pageLimit(offset int, requested *int) (int, error) {
	limit := 50
	if requested != nil {
		limit = *requested
	}
	if offset < 0 || offset > 1000000 || limit < 1 || limit > 100 {
		return 0, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid page range.")
	}
	return limit, nil
}

func resolveFiledFilters(
	ctx *ctxx.SpaceContext,
	input SearchFilesInput,
) ([]int64, int64, error) {
	seen := make(map[string]struct{}, len(input.TagIDs))
	publicIDs := make([]entx.CIText, 0, len(input.TagIDs))
	for _, publicID := range input.TagIDs {
		if publicID == "" || len(publicID) > 100 {
			return nil, 0, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid Tag filter.")
		}
		if _, exists := seen[publicID]; exists {
			continue
		}
		seen[publicID] = struct{}{}
		publicIDs = append(publicIDs, entx.NewCIText(publicID))
	}
	var tagIDs []int64
	if len(publicIDs) > 0 {
		tags, err := ctx.Space.QueryTags().Where(tag.PublicIDIn(publicIDs...)).All(ctx)
		if err != nil {
			return nil, 0, err
		}
		if len(tags) != len(publicIDs) {
			return nil, 0, e.NewHTTPErrorf(http.StatusNotFound, "Tag not found.")
		}
		for _, tagx := range tags {
			tagIDs = append(tagIDs, tagx.ID)
		}
	}
	if input.DocumentTypeID == "" {
		return tagIDs, 0, nil
	}
	documentTypex, err := ctx.Space.QueryDocumentTypes().Where(
		documenttypequery.PublicID(entx.NewCIText(input.DocumentTypeID)),
	).Only(ctx)
	if err != nil {
		return nil, 0, metadataNotFound(err, "Document type not found.")
	}
	return tagIDs, documentTypex.ID, nil
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

func metadataPage(input MetadataListInput) (int, error) {
	limit := 50
	if input.Limit != nil {
		limit = *input.Limit
	}
	if limit < 1 || limit > 100 || input.Offset < 0 || input.Offset > 1000000 {
		return 0, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid page range.")
	}
	return limit, nil
}

func requireMetadataPublicIDs(ctx *ctxx.SpaceContext) error {
	ready, err := tenantdatamigration.MetadataPublicIDsReady(ctx, ctx.TTx.Client())
	if err != nil {
		return err
	}
	if !ready {
		return e.NewHTTPErrorf(http.StatusServiceUnavailable, "Metadata upgrade is still in progress.")
	}
	return nil
}

func (qq *Handler) listTags(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input TagListInput,
) (TagListData, error) {
	result := TagListData{Tags: []TagData{}}
	if err := requireMetadataPublicIDs(ctx); err != nil {
		return result, err
	}
	limit, err := metadataPage(input.MetadataListInput)
	if err != nil {
		return result, err
	}
	query := ctx.Space.QueryTags().WithGroup().WithSubTags()
	if input.GroupID != "" {
		group, err := ctx.Space.QueryTags().Where(
			tag.PublicID(entx.NewCIText(input.GroupID)),
			tag.TypeEQ(tagtype.Group),
		).Only(ctx)
		if err != nil {
			return result, metadataNotFound(err, "Tag group not found.")
		}
		query.Where(tag.GroupID(group.ID))
	}
	tags, err := query.Order(tag.ByName()).Offset(input.Offset).Limit(limit + 1).All(ctx)
	if err != nil {
		return result, err
	}
	result.HasMore = len(tags) > limit
	if result.HasMore {
		tags = tags[:limit]
		next := input.Offset + limit
		result.NextOffset = &next
	}
	for _, tagx := range tags {
		data, err := tagProjection(tagx)
		if err != nil {
			return result, err
		}
		result.Tags = append(result.Tags, data)
	}
	return result, nil
}

func (qq *Handler) listProperties(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input MetadataListInput,
) (PropertyListData, error) {
	result := PropertyListData{Properties: []PropertyData{}}
	if err := requireMetadataPublicIDs(ctx); err != nil {
		return result, err
	}
	limit, err := metadataPage(input)
	if err != nil {
		return result, err
	}
	properties, err := ctx.Space.QueryProperties().Order(property.ByName()).
		Offset(input.Offset).Limit(limit + 1).All(ctx)
	if err != nil {
		return result, err
	}
	result.HasMore = len(properties) > limit
	if result.HasMore {
		properties = properties[:limit]
		next := input.Offset + limit
		result.NextOffset = &next
	}
	for _, propertyx := range properties {
		data, err := propertyProjection(propertyx)
		if err != nil {
			return result, err
		}
		result.Properties = append(result.Properties, data)
	}
	return result, nil
}

func (qq *Handler) listDocumentTypes(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input MetadataListInput,
) (DocumentTypeListData, error) {
	result := DocumentTypeListData{DocumentTypes: []DocumentTypeSummary{}}
	if err := requireMetadataPublicIDs(ctx); err != nil {
		return result, err
	}
	limit, err := metadataPage(input)
	if err != nil {
		return result, err
	}
	documentTypes, err := ctx.Space.QueryDocumentTypes().Order(documenttypequery.ByName()).
		Offset(input.Offset).Limit(limit + 1).All(ctx)
	if err != nil {
		return result, err
	}
	result.HasMore = len(documentTypes) > limit
	if result.HasMore {
		documentTypes = documentTypes[:limit]
		next := input.Offset + limit
		result.NextOffset = &next
	}
	for _, documentTypex := range documentTypes {
		data, err := documentTypeProjection(documentTypex)
		if err != nil {
			return result, err
		}
		result.DocumentTypes = append(result.DocumentTypes, data)
	}
	return result, nil
}

func (qq *Handler) getDocumentType(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input DocumentTypeInput,
) (DocumentTypeData, error) {
	if err := requireMetadataPublicIDs(ctx); err != nil {
		return DocumentTypeData{}, err
	}
	if input.DocumentTypeID == "" || len(input.DocumentTypeID) > 100 {
		return DocumentTypeData{}, e.NewHTTPErrorf(http.StatusBadRequest, "Document type ID is required.")
	}
	documentTypex, err := ctx.Space.QueryDocumentTypes().Where(
		documenttypequery.PublicID(entx.NewCIText(input.DocumentTypeID)),
	).WithAttributes(func(query *enttenant.AttributeQuery) {
		query.WithTag().WithProperty().Order(attribute.ByID())
	}).Only(ctx)
	if err != nil {
		return DocumentTypeData{}, metadataNotFound(err, "Document type not found.")
	}
	summary, err := documentTypeProjection(documentTypex)
	if err != nil {
		return DocumentTypeData{}, err
	}
	result := DocumentTypeData{
		DocumentTypeSummary: summary,
		Attributes:          []DocumentTypeAttributeData{},
	}
	for _, attributex := range documentTypex.Edges.Attributes {
		data := DocumentTypeAttributeData{
			Type:         attributex.Type.String(),
			Name:         attributex.Name,
			IsNameGiving: attributex.IsNameGiving,
			IsProtected:  attributex.IsProtected,
			IsDisabled:   attributex.IsDisabled,
			IsRequired:   attributex.IsRequired,
		}
		if attributex.Type == attributetype.Tag && attributex.Edges.Tag != nil {
			data.TagID, err = metadataPublicID(attributex.Edges.Tag.PublicID)
			if err != nil {
				return DocumentTypeData{}, err
			}
		}
		if attributex.Type == attributetype.Field && attributex.Edges.Property != nil {
			data.PropertyID, err = metadataPublicID(attributex.Edges.Property.PublicID)
			if err != nil {
				return DocumentTypeData{}, err
			}
		}
		result.Attributes = append(result.Attributes, data)
	}
	return result, nil
}

func tagProjection(tagx *enttenant.Tag) (TagData, error) {
	tagID, err := metadataPublicID(tagx.PublicID)
	if err != nil {
		return TagData{}, err
	}
	data := TagData{
		TagID:     tagID,
		Name:      tagx.Name,
		Type:      tagx.Type.String(),
		SubTagIDs: []string{},
	}
	if tagx.Edges.Group != nil {
		data.GroupID, err = metadataPublicID(tagx.Edges.Group.PublicID)
		if err != nil {
			return TagData{}, err
		}
	}
	for _, subTag := range tagx.Edges.SubTags {
		subTagID, err := metadataPublicID(subTag.PublicID)
		if err != nil {
			return TagData{}, err
		}
		data.SubTagIDs = append(data.SubTagIDs, subTagID)
	}
	return data, nil
}

func propertyProjection(propertyx *enttenant.Property) (PropertyData, error) {
	propertyID, err := metadataPublicID(propertyx.PublicID)
	if err != nil {
		return PropertyData{}, err
	}
	return PropertyData{
		PropertyID: propertyID,
		Name:       propertyx.Name,
		Type:       propertyx.Type.String(),
		Unit:       propertyx.Unit,
	}, nil
}

func documentTypeProjection(documentTypex *enttenant.DocumentType) (DocumentTypeSummary, error) {
	documentTypeID, err := metadataPublicID(documentTypex.PublicID)
	if err != nil {
		return DocumentTypeSummary{}, err
	}
	return DocumentTypeSummary{
		DocumentTypeID: documentTypeID,
		Name:           documentTypex.Name,
		IsProtected:    documentTypex.IsProtected,
		IsDisabled:     documentTypex.IsDisabled,
	}, nil
}

func metadataPublicID(publicID entx.CIText) (string, error) {
	value := publicID.String()
	if value == "" {
		return "", e.NewHTTPErrorf(
			http.StatusServiceUnavailable, "Metadata identifiers are unavailable.",
		)
	}
	return value, nil
}

func metadataNotFound(err error, message string) error {
	if enttenant.IsNotFound(err) {
		return e.NewHTTPErrorf(http.StatusNotFound, message)
	}
	return err
}

func (qq *Handler) assignTag(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input FileTagInput,
) (TagAssignmentData, error) {
	filex, tagx, err := resolveFileAndTag(ctx, input)
	if err != nil {
		return TagAssignmentData{}, err
	}
	if _, err := taggingmodel.NewTagService().AssignToFile(
		ctx, filex.ID, tagx.ID, ctx.Space.ID,
	); err != nil {
		return TagAssignmentData{}, err
	}
	return tagAssignmentProjection(ctx, filex, tagx)
}

func (qq *Handler) unassignTag(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input FileTagInput,
) (TagAssignmentData, error) {
	filex, tagx, err := resolveFileAndTag(ctx, input)
	if err != nil {
		return TagAssignmentData{}, err
	}
	if _, err := taggingmodel.NewTagService().UnassignFromFile(ctx, filex.ID, tagx.ID); err != nil {
		return TagAssignmentData{}, err
	}
	return tagAssignmentProjection(ctx, filex, tagx)
}

func resolveFileAndTag(
	ctx *ctxx.SpaceContext,
	input FileTagInput,
) (*enttenant.File, *enttenant.Tag, error) {
	if err := requireMetadataPublicIDs(ctx); err != nil {
		return nil, nil, err
	}
	if input.FileID == "" || len(input.FileID) > 100 || input.TagID == "" || len(input.TagID) > 100 {
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "File ID and Tag ID are required.")
	}
	filex, err := filemodel.NewFileReader().Get(ctx, input.FileID)
	if err != nil {
		return nil, nil, err
	}
	if filex.IsDirectory {
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is a directory.")
	}
	tagx, err := ctx.Space.QueryTags().Where(tag.PublicID(entx.NewCIText(input.TagID))).Only(ctx)
	if err != nil {
		return nil, nil, metadataNotFound(err, "Tag not found.")
	}
	return filex, tagx, nil
}

func tagAssignmentProjection(
	ctx *ctxx.SpaceContext,
	filex *enttenant.File,
	tagx *enttenant.Tag,
) (TagAssignmentData, error) {
	tagID, err := metadataPublicID(tagx.PublicID)
	if err != nil {
		return TagAssignmentData{}, err
	}
	direct, err := ctx.TTx.TagAssignment.Query().Where(
		tagassignment.FileID(filex.ID),
		tagassignment.TagID(tagx.ID),
		tagassignment.SpaceID(ctx.Space.ID),
	).Exist(ctx)
	if err != nil {
		return TagAssignmentData{}, err
	}
	resolved, err := ctx.TTx.ResolvedTagAssignment.Query().Where(
		resolvedtagassignment.FileID(filex.ID),
		resolvedtagassignment.TagID(tagx.ID),
		resolvedtagassignment.SpaceID(ctx.Space.ID),
	).Exist(ctx)
	return TagAssignmentData{
		FileID:           filex.PublicID.String(),
		TagID:            tagID,
		DirectlyAssigned: direct,
		Resolved:         resolved,
	}, err
}

func (qq *Handler) setFileProperty(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input SetFilePropertyInput,
) (FilePropertyMutationData, error) {
	filex, propertyx, err := resolveFileAndProperty(
		ctx, input.FileID, input.PropertyID,
	)
	if err != nil {
		return FilePropertyMutationData{}, err
	}
	value, err := propertyValueFromInput(propertyx.Type, input)
	if err != nil {
		return FilePropertyMutationData{}, err
	}
	_, assignment, err := propertymodel.NewFilePropertyAssignmentService().Set(
		ctx, filex.ID, propertyx.ID, value,
	)
	if err != nil {
		return FilePropertyMutationData{}, err
	}
	projection, err := filePropertyProjection(propertyx, assignment)
	if err != nil {
		return FilePropertyMutationData{}, err
	}
	return FilePropertyMutationData{
		FileID:   filex.PublicID.String(),
		Property: projection,
		Assigned: true,
	}, nil
}

func (qq *Handler) removeFileProperty(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input FilePropertyInput,
) (FilePropertyMutationData, error) {
	filex, propertyx, err := resolveFileAndProperty(ctx, input.FileID, input.PropertyID)
	if err != nil {
		return FilePropertyMutationData{}, err
	}
	_, _, err = propertymodel.NewFilePropertyAssignmentService().Remove(
		ctx, filex.ID, propertyx.ID,
	)
	if err != nil {
		return FilePropertyMutationData{}, err
	}
	projection, err := filePropertyProjection(propertyx, nil)
	if err != nil {
		return FilePropertyMutationData{}, err
	}
	return FilePropertyMutationData{
		FileID:   filex.PublicID.String(),
		Property: projection,
		Assigned: false,
	}, nil
}

func resolveFileAndProperty(
	ctx *ctxx.SpaceContext,
	fileID string,
	propertyID string,
) (*enttenant.File, *enttenant.Property, error) {
	if err := requireMetadataPublicIDs(ctx); err != nil {
		return nil, nil, err
	}
	if fileID == "" || len(fileID) > 100 || propertyID == "" || len(propertyID) > 100 {
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "File ID and field ID are required.")
	}
	filex, err := filemodel.NewFileReader().Get(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if filex.IsDirectory {
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is a directory.")
	}
	propertyx, err := ctx.Space.QueryProperties().Where(
		property.PublicID(entx.NewCIText(propertyID)),
	).Only(ctx)
	if err != nil {
		return nil, nil, metadataNotFound(err, "Field not found.")
	}
	return filex, propertyx, nil
}

func propertyValueFromInput(
	propertyType fieldtype.FieldType,
	input SetFilePropertyInput,
) (propertymodel.FilePropertyValue, error) {
	provided := 0
	for _, exists := range []bool{
		input.TextValue != nil,
		input.NumberValue != nil,
		input.MoneyMinorUnits != nil,
		input.DateValue != nil,
		input.CheckboxValue != nil,
	} {
		if exists {
			provided++
		}
	}
	if provided != 1 {
		return propertymodel.FilePropertyValue{}, e.NewHTTPErrorf(
			http.StatusBadRequest, "Exactly one typed value is required.",
		)
	}
	switch propertyType {
	case fieldtype.Text:
		if input.TextValue != nil {
			return propertymodel.NewTextFilePropertyValue(*input.TextValue), nil
		}
	case fieldtype.Number:
		if input.NumberValue != nil {
			return propertymodel.NewNumberFilePropertyValue(*input.NumberValue)
		}
	case fieldtype.Money:
		if input.MoneyMinorUnits != nil {
			return propertymodel.NewMoneyFilePropertyValue(*input.MoneyMinorUnits)
		}
	case fieldtype.Date:
		if input.DateValue != nil {
			date, err := timex.ParseDate(*input.DateValue)
			if err != nil {
				return propertymodel.FilePropertyValue{}, e.NewHTTPErrorf(
					http.StatusBadRequest, "Date must use YYYY-MM-DD.",
				)
			}
			return propertymodel.NewDateFilePropertyValue(date)
		}
	case fieldtype.Checkbox:
		if input.CheckboxValue != nil {
			return propertymodel.NewCheckboxFilePropertyValue(*input.CheckboxValue), nil
		}
	default:
		return propertymodel.FilePropertyValue{}, e.NewHTTPErrorf(
			http.StatusBadRequest, "Unsupported field type.",
		)
	}
	return propertymodel.FilePropertyValue{}, e.NewHTTPErrorf(
		http.StatusBadRequest, "Value does not match the field type.",
	)
}

func (qq *Handler) setDocumentType(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input SetDocumentTypeInput,
) (DocumentTypeAssignmentData, error) {
	filex, documentTypex, err := resolveFileAndDocumentType(
		ctx, input.FileID, input.DocumentTypeID,
	)
	if err != nil {
		return DocumentTypeAssignmentData{}, err
	}
	if _, err := documenttypemodel.NewAssignmentService().Set(
		ctx, filex.ID, documentTypex.ID,
	); err != nil {
		return DocumentTypeAssignmentData{}, err
	}
	projection, err := documentTypeProjection(documentTypex)
	if err != nil {
		return DocumentTypeAssignmentData{}, err
	}
	return DocumentTypeAssignmentData{
		FileID:       filex.PublicID.String(),
		DocumentType: &projection,
	}, nil
}

func (qq *Handler) clearDocumentType(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input FileInput,
) (DocumentTypeAssignmentData, error) {
	if err := requireMetadataPublicIDs(ctx); err != nil {
		return DocumentTypeAssignmentData{}, err
	}
	if input.FileID == "" || len(input.FileID) > 100 {
		return DocumentTypeAssignmentData{}, e.NewHTTPErrorf(
			http.StatusBadRequest, "File ID is required.",
		)
	}
	filex, err := filemodel.NewFileReader().Get(ctx, input.FileID)
	if err != nil {
		return DocumentTypeAssignmentData{}, err
	}
	if filex.IsDirectory {
		return DocumentTypeAssignmentData{}, e.NewHTTPErrorf(
			http.StatusBadRequest, "File is a directory.",
		)
	}
	if _, err := documenttypemodel.NewAssignmentService().Clear(ctx, filex.ID); err != nil {
		return DocumentTypeAssignmentData{}, err
	}
	return DocumentTypeAssignmentData{FileID: filex.PublicID.String()}, nil
}

func resolveFileAndDocumentType(
	ctx *ctxx.SpaceContext,
	fileID string,
	documentTypeID string,
) (*enttenant.File, *enttenant.DocumentType, error) {
	if err := requireMetadataPublicIDs(ctx); err != nil {
		return nil, nil, err
	}
	if fileID == "" || len(fileID) > 100 || documentTypeID == "" || len(documentTypeID) > 100 {
		return nil, nil, e.NewHTTPErrorf(
			http.StatusBadRequest, "File ID and document type ID are required.",
		)
	}
	filex, err := filemodel.NewFileReader().Get(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if filex.IsDirectory {
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is a directory.")
	}
	documentTypex, err := ctx.Space.QueryDocumentTypes().Where(
		documenttypequery.PublicID(entx.NewCIText(documentTypeID)),
	).Only(ctx)
	if err != nil {
		return nil, nil, metadataNotFound(err, "Document type not found.")
	}
	return filex, documentTypex, nil
}

func (qq *Handler) addClassification(
	ctx *ctxx.SpaceContext,
	filex *enttenant.File,
	data *FileData,
) error {
	if err := requireMetadataPublicIDs(ctx); err != nil {
		return err
	}
	data.DirectTags = []TagData{}
	data.ResolvedTags = []TagData{}
	data.Properties = []FilePropertyData{}
	if filex.DocumentTypeID != 0 {
		documentTypex, err := ctx.Space.QueryDocumentTypes().Where(
			documenttypequery.ID(filex.DocumentTypeID),
		).Only(ctx)
		if err != nil {
			return err
		}
		projection, err := documentTypeProjection(documentTypex)
		if err != nil {
			return err
		}
		data.DocumentType = &projection
	}
	directTags, err := filex.QueryTags().WithGroup().WithSubTags().Order(tag.ByName()).All(ctx)
	if err != nil {
		return err
	}
	for _, tagx := range directTags {
		projection, err := tagProjection(tagx)
		if err != nil {
			return err
		}
		data.DirectTags = append(data.DirectTags, projection)
	}
	var resolvedIDs []int64
	err = ctx.TTx.ResolvedTagAssignment.Query().Where(
		resolvedtagassignment.FileID(filex.ID),
		resolvedtagassignment.SpaceID(ctx.Space.ID),
	).Select(resolvedtagassignment.FieldTagID).Scan(ctx, &resolvedIDs)
	if err != nil {
		return err
	}
	if len(resolvedIDs) > 0 {
		resolvedTags, err := ctx.Space.QueryTags().Where(tag.IDIn(resolvedIDs...)).
			WithGroup().WithSubTags().Order(tag.ByName()).All(ctx)
		if err != nil {
			return err
		}
		for _, tagx := range resolvedTags {
			projection, err := tagProjection(tagx)
			if err != nil {
				return err
			}
			data.ResolvedTags = append(data.ResolvedTags, projection)
		}
	}
	assignments, err := ctx.TTx.FilePropertyAssignment.Query().Where(
		filepropertyassignment.FileID(filex.ID),
		filepropertyassignment.SpaceID(ctx.Space.ID),
	).WithProperty().All(ctx)
	if err != nil {
		return err
	}
	for _, assignment := range assignments {
		if assignment.Edges.Property != nil {
			projection, err := filePropertyProjection(assignment.Edges.Property, assignment)
			if err != nil {
				return err
			}
			data.Properties = append(data.Properties, projection)
		}
	}
	return nil
}

func filePropertyProjection(
	propertyx *enttenant.Property,
	assignment *enttenant.FilePropertyAssignment,
) (FilePropertyData, error) {
	propertyID, err := metadataPublicID(propertyx.PublicID)
	if err != nil {
		return FilePropertyData{}, err
	}
	data := FilePropertyData{
		PropertyID: propertyID,
		Name:       propertyx.Name,
		Type:       propertyx.Type.String(),
		Unit:       propertyx.Unit,
	}
	if assignment == nil {
		return data, nil
	}
	switch propertyx.Type {
	case fieldtype.Text:
		value := assignment.TextValue
		data.TextValue = &value
	case fieldtype.Number:
		value := int64(assignment.NumberValue)
		data.NumberValue = &value
	case fieldtype.Money:
		value := int64(assignment.NumberValue)
		data.MoneyMinorUnits = &value
	case fieldtype.Date:
		value := assignment.DateValue.Format("2006-01-02")
		data.DateValue = &value
	case fieldtype.Checkbox:
		value := assignment.BoolValue
		data.CheckboxValue = &value
	}
	return data, nil
}
