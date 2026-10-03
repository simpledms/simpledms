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

	wx "github.com/simpledms/simpledms/core/ui/widget"
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
	"github.com/simpledms/simpledms/model/tenant/library"
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
	registerRead(
		server, handler, "download_file", "Read original bytes as a bounded base64 chunk.",
		handler.downloadFile,
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
	registerMetadataWrite(server, handler, "create_tag", "Create a Tag.", handler.createTag)
	registerMetadataWrite(server, handler, "edit_tag", "Rename a Tag.", handler.editTag)
	registerMetadataWrite(server, handler, "delete_tag", "Delete an unused Tag.", handler.deleteTag)
	registerMetadataWrite(
		server, handler, "create_property", "Create a field definition.", handler.createProperty,
	)
	registerMetadataWrite(
		server, handler, "edit_property", "Edit a field's name and unit.", handler.editProperty,
	)
	registerMetadataWrite(
		server, handler, "delete_property", "Delete an unused field.", handler.deleteProperty,
	)
	registerMetadataWrite(
		server, handler, "create_document_type", "Create a document type.", handler.createDocumentType,
	)
	registerMetadataWrite(
		server, handler, "rename_document_type", "Rename a document type.", handler.renameDocumentType,
	)
	registerMetadataWrite(
		server, handler, "delete_document_type", "Delete an unused document type.",
		handler.deleteDocumentType,
	)
	registerMetadataWrite(
		server, handler, "create_and_assign_tag", "Create a Tag and assign it to a document.",
		handler.createAndAssignTag,
	)
	registerMetadataWrite(
		server, handler, "move_tag_to_group", "Move a Tag into or out of a group.",
		handler.moveTagToGroup,
	)
	registerMetadataWrite(
		server, handler, "assign_sub_tag", "Add a simple Tag to a composed Tag.", handler.assignSubTag,
	)
	registerMetadataWrite(
		server, handler, "unassign_sub_tag", "Remove a simple Tag from a composed Tag.",
		handler.unassignSubTag,
	)
	registerMetadataWrite(
		server, handler, "create_document_type_tag_attribute", "Add a Tag group attribute.",
		handler.createDocumentTypeTagAttribute,
	)
	registerMetadataWrite(
		server, handler, "edit_document_type_tag_attribute", "Edit a Tag group attribute.",
		handler.editDocumentTypeTagAttribute,
	)
	registerMetadataWrite(
		server, handler, "create_document_type_property_attribute", "Add a field attribute.",
		handler.createDocumentTypePropertyAttribute,
	)
	registerMetadataWrite(
		server, handler, "edit_document_type_property_attribute", "Edit a field attribute.",
		handler.editDocumentTypePropertyAttribute,
	)
	registerMetadataWrite(
		server, handler, "delete_document_type_attribute", "Remove an attribute from a document type.",
		handler.deleteDocumentTypeAttribute,
	)
	registerRead(
		server, handler, "list_document_type_templates", "List available library templates.",
		handler.listDocumentTypeTemplates,
	)
	registerMetadataWrite(
		server, handler, "import_document_types", "Import library templates into an empty Space.",
		handler.importDocumentTypes,
	)
	registerRead(
		server, handler, "list_document_notes", "List bounded note previews and optional history.",
		handler.listDocumentNotes,
	)
	registerRead(
		server, handler, "get_document_note", "Read a bounded note body and its metadata.",
		handler.getDocumentNote,
	)
	registerWrite(
		server, handler, "create_document_note", "Create an authored document note.",
		handler.createDocumentNote,
	)
	registerWrite(
		server, handler, "edit_document_note", "Edit a current note with existing author permissions.",
		handler.editDocumentNote,
	)
	registerWrite(
		server, handler, "replace_document_note", "Replace a current note and retain its history.",
		handler.replaceDocumentNote,
	)
	registerWrite(
		server, handler, "delete_document_note", "Delete a current note into read-only history.",
		handler.deleteDocumentNote,
	)
	registerWrite(
		server, handler, "rename_file", "Rename a filed document or directory.", handler.renameFile,
	)
	registerWrite(
		server, handler, "move_file", "Move a filed document or directory.", handler.moveFile,
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

func registerMetadataWrite[I, O any](
	server *sdk.Server,
	handler *Handler,
	name, description string,
	fn func(context.Context, *ctxx.SpaceContext, *entmain.MCPCredential, I) (O, error),
) {
	registerWrite(server, handler, name, description, func(
		requestCtx context.Context, ctx *ctxx.SpaceContext, cred *entmain.MCPCredential, input I,
	) (O, error) {
		var zero O
		if err := requireMetadataPublicIDs(ctx); err != nil {
			return zero, err
		}
		output, err := fn(requestCtx, ctx, cred, input)
		if enttenant.IsConstraintError(err) {
			return zero, e.NewHTTPErrorf(
				http.StatusBadRequest, "Metadata already exists or is still in use.",
			)
		}
		return output, err
	})
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
		case http.StatusConflict:
			code = "conflict"
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
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "File is a folder.")
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

func (qq *Handler) downloadFile(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input DownloadFileInput,
) (DownloadFileData, error) {
	versionNumber := 0
	if input.VersionNumber != nil {
		versionNumber = *input.VersionNumber
		if versionNumber < 1 {
			return DownloadFileData{}, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid version number.")
		}
	}
	length := filesystem.MaxDownloadChunkBytes
	if input.Length != nil {
		length = *input.Length
	}
	result, err := filesystem.NewFileDownloadService(qq.config.Infra.FileSystem()).Read(
		ctx, input.FileID, versionNumber, input.Offset, length,
	)
	if err != nil {
		return DownloadFileData{}, err
	}
	data := DownloadFileData{
		FileID:        result.FilePublicID,
		Filename:      result.Filename,
		VersionNumber: result.VersionNumber,
		MIMEType:      result.MIMEType,
		Size:          result.Size,
		ContentSHA256: result.ContentSHA256,
		Offset:        result.Offset,
		ContentBase64: base64.StdEncoding.EncodeToString(result.Content),
		HasMore:       result.HasMore,
	}
	if result.HasMore {
		next := result.Offset + int64(len(result.Content))
		data.NextOffset = &next
	}
	return data, nil
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
		return result, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid folder ID.")
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
		len(input.DocumentTypeID) > 100 || len(input.PropertyFilters) > 32 {
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
	for _, inputFilter := range input.PropertyFilters {
		filter, err := filedPropertyFilter(ctx, inputFilter)
		if err != nil {
			return result, err
		}
		query = filter.Apply(query)
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

func filedPropertyFilter(
	ctx *ctxx.SpaceContext, input PropertyFilterInput,
) (propertymodel.FilePropertyFilter, error) {
	if err := requireMetadataPublicIDs(ctx); err != nil {
		return propertymodel.FilePropertyFilter{}, err
	}
	propertyx, err := scopedProperty(ctx, input.PropertyID)
	if err != nil {
		return propertymodel.FilePropertyFilter{}, err
	}
	if input.TextValue != nil && utf8.RuneCountInString(*input.TextValue) > 1000 {
		return propertymodel.FilePropertyFilter{}, e.NewHTTPErrorf(
			http.StatusBadRequest, "Field filter text is too long.",
		)
	}
	value, err := propertyValueFromInput(propertyx.Type, SetFilePropertyInput{
		TextValue:       input.TextValue,
		NumberValue:     input.NumberValue,
		MoneyMinorUnits: input.MoneyMinorUnits,
		DateValue:       input.DateValue,
		CheckboxValue:   input.CheckboxValue,
	})
	if err != nil {
		return propertymodel.FilePropertyFilter{}, err
	}
	var end *propertymodel.FilePropertyValue
	if input.EndNumberValue != nil || input.EndMoneyMinorUnits != nil || input.EndDateValue != nil {
		endValue, err := propertyValueFromInput(propertyx.Type, SetFilePropertyInput{
			NumberValue:     input.EndNumberValue,
			MoneyMinorUnits: input.EndMoneyMinorUnits,
			DateValue:       input.EndDateValue,
		})
		if err != nil {
			return propertymodel.FilePropertyFilter{}, err
		}
		end = &endValue
	}
	return propertymodel.NewFilePropertyFilter(propertyx.ID, input.Operator, value, end)
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

func nilableMetadataNameError(name string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 300 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Name must contain 1 to 300 characters.")
	}
	return nil
}

func nilableMetadataReferenceError(publicID string) error {
	if publicID == "" || len(publicID) > 100 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "A public metadata ID is required.")
	}
	return nil
}

func scopedTag(ctx *ctxx.SpaceContext, publicID string) (*enttenant.Tag, error) {
	if err := nilableMetadataReferenceError(publicID); err != nil {
		return nil, err
	}
	tagx, err := ctx.Space.QueryTags().Where(tag.PublicID(entx.NewCIText(publicID))).
		WithGroup().WithSubTags().Only(ctx)
	return tagx, metadataNotFound(err, "Tag not found.")
}

func scopedProperty(ctx *ctxx.SpaceContext, publicID string) (*enttenant.Property, error) {
	if err := nilableMetadataReferenceError(publicID); err != nil {
		return nil, err
	}
	propertyx, err := ctx.Space.QueryProperties().Where(
		property.PublicID(entx.NewCIText(publicID)),
	).Only(ctx)
	return propertyx, metadataNotFound(err, "Field not found.")
}

func scopedDocumentType(
	ctx *ctxx.SpaceContext, publicID string,
) (*documenttypemodel.DocumentType, error) {
	if err := nilableMetadataReferenceError(publicID); err != nil {
		return nil, err
	}
	documentTypex, err := ctx.Space.QueryDocumentTypes().Where(
		documenttypequery.PublicID(entx.NewCIText(publicID)),
	).Only(ctx)
	if err != nil {
		return nil, metadataNotFound(err, "Document type not found.")
	}
	return documenttypemodel.NewDocumentType(documentTypex), nil
}

func (qq *Handler) createTag(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input CreateTagInput,
) (TagData, error) {
	tagx, err := createScopedTag(ctx, input)
	if err != nil {
		return TagData{}, err
	}
	tagx, err = scopedTag(ctx, tagx.PublicID.String())
	if err != nil {
		return TagData{}, err
	}
	return tagProjection(tagx)
}

func createScopedTag(ctx *ctxx.SpaceContext, input CreateTagInput) (*enttenant.Tag, error) {
	typex, groupID, err := tagCreationValues(ctx, input)
	if err != nil {
		return nil, err
	}
	return taggingmodel.NewTagService().Create(ctx, ctx.Space.ID, groupID, input.Name, typex)
}

func tagCreationValues(
	ctx *ctxx.SpaceContext, input CreateTagInput,
) (tagtype.TagType, int64, error) {
	if err := nilableMetadataNameError(input.Name); err != nil {
		return 0, 0, err
	}
	typex, err := tagtype.TagTypeString(input.Type)
	if err != nil || (typex != tagtype.Simple && typex != tagtype.Group && typex != tagtype.Super) {
		return 0, 0, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid Tag type.")
	}
	var groupID int64
	if input.GroupID != "" {
		group, err := scopedTag(ctx, input.GroupID)
		if err != nil {
			return 0, 0, err
		}
		groupID = group.ID
	}
	return typex, groupID, nil
}

func (qq *Handler) editTag(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input EditTagInput,
) (TagData, error) {
	if err := nilableMetadataNameError(input.Name); err != nil {
		return TagData{}, err
	}
	tagx, err := scopedTag(ctx, input.TagID)
	if err != nil {
		return TagData{}, err
	}
	if _, err := taggingmodel.NewTagService().Edit(ctx, tagx.ID, input.Name); err != nil {
		return TagData{}, err
	}
	tagx, err = scopedTag(ctx, input.TagID)
	if err != nil {
		return TagData{}, err
	}
	return tagProjection(tagx)
}

func (qq *Handler) deleteTag(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input TagInput,
) (MetadataDeletionData, error) {
	tagx, err := scopedTag(ctx, input.TagID)
	if err != nil {
		return MetadataDeletionData{}, err
	}
	_, err = taggingmodel.NewTagService().Delete(ctx, tagx.ID)
	return MetadataDeletionData{Deleted: err == nil}, err
}

func (qq *Handler) createProperty(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input CreatePropertyInput,
) (PropertyData, error) {
	if err := nilableMetadataNameError(input.Name); err != nil {
		return PropertyData{}, err
	}
	if utf8.RuneCountInString(input.Unit) > 300 {
		return PropertyData{}, e.NewHTTPErrorf(http.StatusBadRequest, "Field unit is too long.")
	}
	typex, err := fieldtype.FieldTypeString(input.Type)
	if err != nil || (typex != fieldtype.Text && typex != fieldtype.Number &&
		typex != fieldtype.Money && typex != fieldtype.Date && typex != fieldtype.Checkbox) {
		return PropertyData{}, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid field type.")
	}
	propertyx, err := propertymodel.NewPropertyService().Create(
		ctx, ctx.Space.ID, input.Name, typex, input.Unit,
	)
	if err != nil {
		return PropertyData{}, err
	}
	return propertyProjection(propertyx)
}

func (qq *Handler) editProperty(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input EditPropertyInput,
) (PropertyData, error) {
	if err := nilableMetadataNameError(input.Name); err != nil {
		return PropertyData{}, err
	}
	propertyx, err := scopedProperty(ctx, input.PropertyID)
	if err != nil {
		return PropertyData{}, err
	}
	unit := propertyx.Unit
	if input.Unit != nil {
		unit = *input.Unit
	}
	if utf8.RuneCountInString(unit) > 300 {
		return PropertyData{}, e.NewHTTPErrorf(http.StatusBadRequest, "Field unit is too long.")
	}
	propertyx, err = propertymodel.NewPropertyService().Edit(
		ctx, ctx.Space, propertyx.ID, input.Name, unit,
	)
	if err != nil {
		return PropertyData{}, err
	}
	return propertyProjection(propertyx)
}

func (qq *Handler) deleteProperty(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input PropertyInput,
) (MetadataDeletionData, error) {
	propertyx, err := scopedProperty(ctx, input.PropertyID)
	if err != nil {
		return MetadataDeletionData{}, err
	}
	err = propertymodel.NewPropertyService().Delete(ctx, ctx.Space, propertyx.ID)
	return MetadataDeletionData{Deleted: err == nil}, err
}

func (qq *Handler) createDocumentType(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input CreateDocumentTypeInput,
) (DocumentTypeSummary, error) {
	if err := nilableMetadataNameError(input.Name); err != nil {
		return DocumentTypeSummary{}, err
	}
	documentTypex, err := documenttypemodel.Create(ctx, ctx.Space.ID, input.Name)
	if err != nil {
		return DocumentTypeSummary{}, err
	}
	return documentTypeProjection(documentTypex.Data)
}

func (qq *Handler) renameDocumentType(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input RenameDocumentTypeInput,
) (DocumentTypeSummary, error) {
	if err := nilableMetadataNameError(input.Name); err != nil {
		return DocumentTypeSummary{}, err
	}
	documentTypex, err := scopedDocumentType(ctx, input.DocumentTypeID)
	if err != nil {
		return DocumentTypeSummary{}, err
	}
	if err := documentTypex.Rename(ctx, input.Name); err != nil {
		return DocumentTypeSummary{}, err
	}
	return documentTypeProjection(documentTypex.Data)
}

func (qq *Handler) deleteDocumentType(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input DocumentTypeInput,
) (MetadataDeletionData, error) {
	documentTypex, err := scopedDocumentType(ctx, input.DocumentTypeID)
	if err != nil {
		return MetadataDeletionData{}, err
	}
	err = documentTypex.Delete(ctx)
	return MetadataDeletionData{Deleted: err == nil}, err
}

func (qq *Handler) createAndAssignTag(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input CreateAndAssignTagInput,
) (TagAssignmentData, error) {
	if err := nilableMetadataReferenceError(input.FileID); err != nil {
		return TagAssignmentData{}, err
	}
	filex, err := filemodel.NewFileReader().Get(ctx, input.FileID)
	if err != nil {
		return TagAssignmentData{}, err
	}
	if filex.IsDirectory {
		return TagAssignmentData{}, e.NewHTTPErrorf(http.StatusBadRequest, "File is a folder.")
	}
	typex, groupID, err := tagCreationValues(ctx, input.CreateTagInput)
	if err != nil {
		return TagAssignmentData{}, err
	}
	tagx, err := taggingmodel.NewTagService().CreateAndAssignToFile(
		ctx, filex.ID, ctx.Space.ID, groupID, input.Name, typex,
	)
	if err != nil {
		return TagAssignmentData{}, err
	}
	return tagAssignmentProjection(ctx, filex, tagx)
}

func (qq *Handler) moveTagToGroup(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input MoveTagToGroupInput,
) (TagData, error) {
	tagx, err := scopedTag(ctx, input.TagID)
	if err != nil {
		return TagData{}, err
	}
	var groupID int64
	if input.GroupID != "" {
		group, err := scopedTag(ctx, input.GroupID)
		if err != nil {
			return TagData{}, err
		}
		groupID = group.ID
	}
	if _, _, err := taggingmodel.NewTagService().MoveToGroup(ctx, tagx.ID, groupID); err != nil {
		return TagData{}, err
	}
	tagx, err = scopedTag(ctx, input.TagID)
	if err != nil {
		return TagData{}, err
	}
	return tagProjection(tagx)
}

func (qq *Handler) assignSubTag(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input TagCompositionInput,
) (TagData, error) {
	return qq.setTagComposition(ctx, input, true)
}

func (qq *Handler) unassignSubTag(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input TagCompositionInput,
) (TagData, error) {
	return qq.setTagComposition(ctx, input, false)
}

func (qq *Handler) setTagComposition(
	ctx *ctxx.SpaceContext, input TagCompositionInput, isAssigned bool,
) (TagData, error) {
	superTag, err := scopedTag(ctx, input.SuperTagID)
	if err != nil {
		return TagData{}, err
	}
	subTag, err := scopedTag(ctx, input.SubTagID)
	if err != nil {
		return TagData{}, err
	}
	service := taggingmodel.NewTagService()
	if isAssigned {
		_, _, err = service.AssignSubTag(ctx, superTag.ID, subTag.ID)
	} else {
		_, _, err = service.UnassignSubTag(ctx, superTag.ID, subTag.ID)
	}
	if err != nil {
		return TagData{}, err
	}
	superTag, err = scopedTag(ctx, input.SuperTagID)
	if err != nil {
		return TagData{}, err
	}
	return tagProjection(superTag)
}

func (qq *Handler) createDocumentTypeTagAttribute(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input CreateDocumentTypeTagAttributeInput,
) (DocumentTypeData, error) {
	if err := nilableMetadataNameError(input.Name); err != nil {
		return DocumentTypeData{}, err
	}
	documentTypex, err := scopedDocumentType(ctx, input.DocumentTypeID)
	if err != nil {
		return DocumentTypeData{}, err
	}
	tagx, err := scopedTag(ctx, input.TagID)
	if err != nil {
		return DocumentTypeData{}, err
	}
	if _, err := documentTypex.CreateTagAttribute(
		ctx, input.Name, tagx.ID, input.IsNameGiving,
	); err != nil {
		return DocumentTypeData{}, err
	}
	return qq.getDocumentType(requestCtx, ctx, cred, DocumentTypeInput{
		DocumentTypeID: input.DocumentTypeID,
	})
}

func (qq *Handler) createDocumentTypePropertyAttribute(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input CreateDocumentTypePropertyAttributeInput,
) (DocumentTypeData, error) {
	documentTypex, err := scopedDocumentType(ctx, input.DocumentTypeID)
	if err != nil {
		return DocumentTypeData{}, err
	}
	propertyx, err := scopedProperty(ctx, input.PropertyID)
	if err != nil {
		return DocumentTypeData{}, err
	}
	if _, err := documentTypex.CreatePropertyAttribute(
		ctx, propertyx.ID, input.IsNameGiving,
	); err != nil {
		return DocumentTypeData{}, err
	}
	return qq.getDocumentType(requestCtx, ctx, cred, DocumentTypeInput{
		DocumentTypeID: input.DocumentTypeID,
	})
}

func scopedDocumentTypeAttribute(
	ctx *ctxx.SpaceContext, input DocumentTypeAttributeInput,
) (*documenttypemodel.Attribute, error) {
	if (input.TagID == "") == (input.PropertyID == "") {
		return nil, e.NewHTTPErrorf(
			http.StatusBadRequest, "Exactly one Tag ID or field ID is required.",
		)
	}
	documentTypex, err := scopedDocumentType(ctx, input.DocumentTypeID)
	if err != nil {
		return nil, err
	}
	query := documentTypex.Data.QueryAttributes()
	if input.TagID != "" {
		tagx, err := scopedTag(ctx, input.TagID)
		if err != nil {
			return nil, err
		}
		query.Where(attribute.TagID(tagx.ID), attribute.TypeEQ(attributetype.Tag))
	} else {
		propertyx, err := scopedProperty(ctx, input.PropertyID)
		if err != nil {
			return nil, err
		}
		query.Where(attribute.PropertyID(propertyx.ID), attribute.TypeEQ(attributetype.Field))
	}
	attributex, err := query.Only(ctx)
	if err != nil {
		return nil, metadataNotFound(err, "Document type attribute not found.")
	}
	return documenttypemodel.NewAttribute(attributex), nil
}

func (qq *Handler) editDocumentTypeTagAttribute(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input EditDocumentTypeTagAttributeInput,
) (DocumentTypeData, error) {
	if err := nilableMetadataNameError(input.Name); err != nil {
		return DocumentTypeData{}, err
	}
	attributex, err := scopedDocumentTypeAttribute(ctx, DocumentTypeAttributeInput{
		DocumentTypeID: input.DocumentTypeID,
		TagID:          input.TagID,
	})
	if err != nil {
		return DocumentTypeData{}, err
	}
	if err := attributex.RenameAndSetIsNameGiving(ctx, input.Name, input.IsNameGiving); err != nil {
		return DocumentTypeData{}, err
	}
	return qq.getDocumentType(requestCtx, ctx, cred, DocumentTypeInput{
		DocumentTypeID: input.DocumentTypeID,
	})
}

func (qq *Handler) editDocumentTypePropertyAttribute(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input EditDocumentTypePropertyAttributeInput,
) (DocumentTypeData, error) {
	attributex, err := scopedDocumentTypeAttribute(ctx, DocumentTypeAttributeInput{
		DocumentTypeID: input.DocumentTypeID,
		PropertyID:     input.PropertyID,
	})
	if err != nil {
		return DocumentTypeData{}, err
	}
	if err := attributex.SetIsNameGiving(ctx, input.IsNameGiving); err != nil {
		return DocumentTypeData{}, err
	}
	return qq.getDocumentType(requestCtx, ctx, cred, DocumentTypeInput{
		DocumentTypeID: input.DocumentTypeID,
	})
}

func (qq *Handler) deleteDocumentTypeAttribute(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input DocumentTypeAttributeInput,
) (DocumentTypeData, error) {
	attributex, err := scopedDocumentTypeAttribute(ctx, input)
	if err != nil {
		return DocumentTypeData{}, err
	}
	if err := attributex.Delete(ctx); err != nil {
		return DocumentTypeData{}, err
	}
	return qq.getDocumentType(requestCtx, ctx, cred, DocumentTypeInput{
		DocumentTypeID: input.DocumentTypeID,
	})
}

func (qq *Handler) listDocumentTypeTemplates(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, _ struct{},
) (DocumentTypeTemplateListData, error) {
	result := DocumentTypeTemplateListData{Templates: []DocumentTypeTemplateData{}}
	for _, template := range library.BuiltinTemplates() {
		result.Templates = append(result.Templates, DocumentTypeTemplateData{
			Key:  template.Key,
			Name: wx.T(template.Name).String(ctx),
			Icon: template.Icon,
		})
	}
	return result, nil
}

func (qq *Handler) importDocumentTypes(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input ImportDocumentTypesInput,
) (DocumentTypeListData, error) {
	if len(input.TemplateKeys) == 0 || len(input.TemplateKeys) > 64 {
		return DocumentTypeListData{}, e.NewHTTPErrorf(
			http.StatusBadRequest, "Select between 1 and 64 document type templates.",
		)
	}
	if err := documenttypemodel.ImportFromLibrary(ctx, input.TemplateKeys); err != nil {
		return DocumentTypeListData{}, err
	}
	limit := 100
	return qq.listDocumentTypes(requestCtx, ctx, cred, MetadataListInput{Limit: &limit})
}

func (qq *Handler) listDocumentNotes(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input DocumentNoteListInput,
) (DocumentNoteListData, error) {
	result := DocumentNoteListData{FileID: input.FileID, Notes: []DocumentNoteData{}}
	if err := nilableMetadataReferenceError(input.FileID); err != nil {
		return result, err
	}
	limit, err := pageLimit(input.Offset, input.Limit)
	if err != nil {
		return result, err
	}
	doc, query, err := filemodel.NewDocumentNotes().Query(ctx, input.FileID, input.ShowHistory)
	if err != nil {
		return result, err
	}
	notes, err := query.Offset(input.Offset).Limit(limit + 1).All(ctx)
	if err != nil {
		return result, err
	}
	result.HasMore = len(notes) > limit
	if result.HasMore {
		notes = notes[:limit]
		next := input.Offset + limit
		result.NextOffset = &next
	}
	for _, note := range notes {
		data, err := documentNoteProjection(ctx, cred, doc, note, 0, 1000)
		if err != nil {
			return result, err
		}
		result.Notes = append(result.Notes, data)
	}
	if input.Offset == 0 && doc.Notes != "" {
		legacy, err := documentNoteProjection(ctx, cred, doc, nil, 0, 1000)
		if err != nil {
			return result, err
		}
		result.LegacyNote = &legacy
	}
	return result, nil
}

func (qq *Handler) getDocumentNote(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input GetDocumentNoteInput,
) (DocumentNoteData, error) {
	if err := nilableDocumentNoteReferenceError(input.DocumentNoteInput); err != nil {
		return DocumentNoteData{}, err
	}
	length := 12000
	if input.Length != nil {
		length = *input.Length
	}
	if input.Offset < 0 || input.Offset > 1000000 || length < 1 || length > 50000 {
		return DocumentNoteData{}, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid note text range.")
	}
	doc, note, err := filemodel.NewDocumentNotes().Get(ctx, input.FileID, input.NoteID)
	if err != nil {
		return DocumentNoteData{}, err
	}
	return documentNoteProjection(ctx, cred, doc, note, input.Offset, length)
}

func documentNoteProjection(
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	doc *enttenant.File,
	note *enttenant.DocumentNote,
	offset, length int,
) (DocumentNoteData, error) {
	data := DocumentNoteData{
		FileID:     doc.PublicID.String(),
		NoteID:     "legacy",
		IsLegacy:   note == nil,
		BodyOffset: offset,
	}
	body, authorID, isCurrent := doc.Notes, int64(0), true
	if note != nil {
		data.NoteID = note.PublicID.String()
		data.Title = note.Title
		data.Author = nilableNoteActorProjection(note.Edges.Author)
		data.AuthoredAt = note.AuthoredAt
		data.Editor = nilableNoteActorProjection(note.Edges.Editor)
		data.EditedAt = note.EditedAt
		data.DeletedAt = note.DeletedAt
		if replacement := note.Edges.Replacement; replacement != nil {
			data.ReplacedByNoteID = replacement.PublicID.String()
		}
		body, authorID = note.Body, note.AuthorID
		isCurrent = note.DeletedAt == nil && note.ReplacedByID == 0
	}
	var err error
	data.Body, data.HasMoreBody, err = filemodel.NewFileReader().TextWindow(body, offset, length)
	if err != nil {
		return DocumentNoteData{}, err
	}
	if data.HasMoreBody {
		next := offset + length
		data.NextBodyOffset = &next
	}
	data.CanChange = !cred.IsReadOnly && isCurrent &&
		filemodel.NewDocumentNotes().CanChange(ctx, doc, authorID)
	return data, nil
}

func nilableNoteActorProjection(actor *enttenant.User) *NoteActorData {
	if actor == nil {
		return nil
	}
	return &NoteActorData{
		UserID: actor.PublicID.String(),
		Name:   strings.TrimSpace(actor.FirstName + " " + actor.LastName),
	}
}

func nilableDocumentNoteReferenceError(input DocumentNoteInput) error {
	if input.FileID == "" || len(input.FileID) > 100 || input.NoteID == "" || len(input.NoteID) > 100 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Document and note IDs are required.")
	}
	return nil
}

func nilableDocumentNoteTextError(title, body string) error {
	if utf8.RuneCountInString(title) > 300 || utf8.RuneCountInString(body) > 50000 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Note title or body is too long.")
	}
	return nil
}

func (qq *Handler) createDocumentNote(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input CreateDocumentNoteInput,
) (DocumentNoteData, error) {
	if err := nilableMetadataReferenceError(input.FileID); err != nil {
		return DocumentNoteData{}, err
	}
	if err := nilableDocumentNoteTextError(input.Title, input.Body); err != nil {
		return DocumentNoteData{}, err
	}
	note, err := filemodel.NewDocumentNotes().Create(ctx, input.FileID, input.Title, input.Body)
	if err != nil {
		return DocumentNoteData{}, err
	}
	length := 50000
	return qq.getDocumentNote(requestCtx, ctx, cred, GetDocumentNoteInput{
		DocumentNoteInput: DocumentNoteInput{
			FileID: input.FileID,
			NoteID: note.PublicID.String(),
		},
		Length: &length,
	})
}

func (qq *Handler) editDocumentNote(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input EditDocumentNoteInput,
) (DocumentNoteData, error) {
	return qq.changeDocumentNote(requestCtx, ctx, cred, input, false)
}

func (qq *Handler) replaceDocumentNote(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input EditDocumentNoteInput,
) (DocumentNoteData, error) {
	return qq.changeDocumentNote(requestCtx, ctx, cred, input, true)
}

func (qq *Handler) changeDocumentNote(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	cred *entmain.MCPCredential,
	input EditDocumentNoteInput,
	isReplacement bool,
) (DocumentNoteData, error) {
	if err := nilableDocumentNoteReferenceError(input.DocumentNoteInput); err != nil {
		return DocumentNoteData{}, err
	}
	if err := nilableDocumentNoteTextError(input.Title, input.Body); err != nil {
		return DocumentNoteData{}, err
	}
	var note *enttenant.DocumentNote
	var err error
	service := filemodel.NewDocumentNotes()
	if isReplacement {
		note, err = service.Replace(ctx, input.FileID, input.NoteID, input.Title, input.Body)
	} else {
		note, err = service.Edit(ctx, input.FileID, input.NoteID, input.Title, input.Body)
	}
	if err != nil {
		return DocumentNoteData{}, err
	}
	length := 50000
	return qq.getDocumentNote(requestCtx, ctx, cred, GetDocumentNoteInput{
		DocumentNoteInput: DocumentNoteInput{
			FileID: input.FileID,
			NoteID: note.PublicID.String(),
		},
		Length: &length,
	})
}

func (qq *Handler) deleteDocumentNote(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input DocumentNoteInput,
) (DocumentNoteDeletionData, error) {
	if err := nilableDocumentNoteReferenceError(input); err != nil {
		return DocumentNoteDeletionData{}, err
	}
	deleted, err := filemodel.NewDocumentNotes().Delete(ctx, input.FileID, input.NoteID)
	return DocumentNoteDeletionData{
		FileID:  input.FileID,
		NoteID:  input.NoteID,
		Deleted: deleted,
	}, err
}

func (qq *Handler) renameFile(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input RenameFileInput,
) (FilingData, error) {
	if err := nilableMetadataReferenceError(input.FileID); err != nil {
		return FilingData{}, err
	}
	filex, err := filesystem.NewFileOrganizationService(qq.config.Infra.FileSystem()).Rename(
		ctx, input.FileID, input.NewFilename,
	)
	if err != nil {
		return FilingData{}, organizationError(err)
	}
	return filingProjection(ctx, filex)
}

func (qq *Handler) moveFile(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input MoveFileInput,
) (FilingData, error) {
	if err := nilableMetadataReferenceError(input.FileID); err != nil {
		return FilingData{}, err
	}
	if err := nilableMetadataReferenceError(input.DestinationDirectoryID); err != nil {
		return FilingData{}, err
	}
	filex, err := filesystem.NewFileOrganizationService(qq.config.Infra.FileSystem()).Move(
		ctx, input.FileID, input.DestinationDirectoryID, input.Filename, input.NewDirectoryName,
	)
	if err != nil {
		return FilingData{}, organizationError(err)
	}
	return filingProjection(ctx, filex)
}

func organizationError(err error) error {
	if enttenant.IsConstraintError(err) {
		return e.NewHTTPErrorf(http.StatusConflict, "A file with this name already exists.")
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
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is a folder.")
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
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is a folder.")
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
			http.StatusBadRequest, "File is a folder.",
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
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is a folder.")
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
