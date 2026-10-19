package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	credentialmodel "github.com/simpledms/simpledms/model/main/mcpcredential"
	"github.com/simpledms/simpledms/util/e"
)

type requestContextKey int

type requestContext struct {
	source context.Context
	origin string
}

const maxUploadBytes int64 = 10 * 1024 * 1024

const fileIsFolderMessage = "File is a folder."

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
	documents := newDocumentTools(config.Infra, handler.credentials)
	metadata := newMetadataTools()
	notes := newNoteTools()
	classification := newClassificationTools()
	server := sdk.NewServer(&sdk.Implementation{
		Name:    "SimpleDMS",
		Version: "1",
	}, nil)
	registerRead(server, handler, "get_space", "Identify the authorized Space.", documents.getSpace)
	registerRead(server, handler, "list_inbox", "List or search Inbox documents.", documents.listInbox)
	registerRead(server, handler, "get_file", "Read a live file's metadata.", documents.getFile)
	registerRead(
		server, handler, "read_file_text",
		"Read bounded existing OCR text, as markdown with tables when available.",
		documents.readText,
	)
	registerRead(
		server, handler, "download_file", "Read original bytes as a bounded base64 chunk.",
		documents.downloadFile,
	)
	registerStorageWrite(
		server, handler, "upload_file", "Upload one document into Inbox.", documents.uploadFile,
	)
	registerRead(server, handler, "list_tags", "List existing Tags.", metadata.listTags)
	registerRead(server, handler, "list_properties", "List existing fields.", metadata.listProperties)
	registerRead(
		server, handler, "list_document_types", "List document types.", metadata.listDocumentTypes,
	)
	registerRead(
		server, handler, "get_document_type", "Read a document type.", metadata.getDocumentType,
	)
	registerWrite(
		server, handler, "assign_tag", "Assign a Tag to a document.", classification.assignTag,
	)
	registerWrite(
		server, handler, "unassign_tag", "Unassign a direct Tag.", classification.unassignTag,
	)
	registerWrite(
		server, handler, "set_file_property", "Set a typed field value.", classification.setFileProperty,
	)
	registerWrite(
		server, handler, "remove_file_property", "Remove a field value.",
		classification.removeFileProperty,
	)
	registerWrite(
		server, handler, "set_document_type", "Set a document type.", classification.setDocumentType,
	)
	registerWrite(
		server, handler, "clear_document_type", "Clear a document type.",
		classification.clearDocumentType,
	)
	registerRead(
		server, handler, "list_directory", "List one filing directory.", documents.listDirectory,
	)
	registerWrite(
		server, handler, "create_directory", "Create a filing directory.", documents.createDirectory,
	)
	registerWrite(
		server, handler, "file_inbox_document", "File one Inbox document.",
		documents.fileInboxDocument,
	)
	registerWrite(
		server, handler, "mark_inbox_file_done", "Complete an Inbox document without moving it.",
		documents.markInboxFileDone,
	)
	registerRead(
		server, handler, "search_files", "Search or list filed documents.", documents.searchFiles,
	)
	registerMetadataWrite(server, handler, "create_tag", "Create a Tag.", metadata.createTag)
	registerMetadataWrite(server, handler, "edit_tag", "Rename a Tag.", metadata.editTag)
	registerMetadataWrite(server, handler, "delete_tag", "Delete an unused Tag.", metadata.deleteTag)
	registerMetadataWrite(
		server, handler, "create_property", "Create a field definition.", metadata.createProperty,
	)
	registerMetadataWrite(
		server, handler, "edit_property", "Edit a field's name and unit.", metadata.editProperty,
	)
	registerMetadataWrite(
		server, handler, "delete_property", "Delete an unused field.", metadata.deleteProperty,
	)
	registerMetadataWrite(
		server, handler, "create_document_type", "Create a document type.", metadata.createDocumentType,
	)
	registerMetadataWrite(
		server, handler, "rename_document_type", "Rename a document type.", metadata.renameDocumentType,
	)
	registerMetadataWrite(
		server, handler, "delete_document_type", "Delete an unused document type.",
		metadata.deleteDocumentType,
	)
	registerMetadataWrite(
		server, handler, "create_and_assign_tag", "Create a Tag and assign it to a document.",
		metadata.createAndAssignTag,
	)
	registerMetadataWrite(
		server, handler, "move_tag_to_group", "Move a Tag into or out of a group.",
		metadata.moveTagToGroup,
	)
	registerMetadataWrite(
		server, handler, "assign_sub_tag", "Add a simple Tag to a composed Tag.", metadata.assignSubTag,
	)
	registerMetadataWrite(
		server, handler, "unassign_sub_tag", "Remove a simple Tag from a composed Tag.",
		metadata.unassignSubTag,
	)
	registerMetadataWrite(
		server, handler, "create_document_type_tag_attribute", "Add a Tag group attribute.",
		metadata.createDocumentTypeTagAttribute,
	)
	registerMetadataWrite(
		server, handler, "edit_document_type_tag_attribute", "Edit a Tag group attribute.",
		metadata.editDocumentTypeTagAttribute,
	)
	registerMetadataWrite(
		server, handler, "create_document_type_property_attribute", "Add a field attribute.",
		metadata.createDocumentTypePropertyAttribute,
	)
	registerMetadataWrite(
		server, handler, "edit_document_type_property_attribute", "Edit a field attribute.",
		metadata.editDocumentTypePropertyAttribute,
	)
	registerMetadataWrite(
		server, handler, "delete_document_type_attribute", "Remove an attribute from a document type.",
		metadata.deleteDocumentTypeAttribute,
	)
	registerRead(
		server, handler, "list_document_type_templates", "List available library templates.",
		metadata.listDocumentTypeTemplates,
	)
	registerMetadataWrite(
		server, handler, "import_document_types", "Import library templates into an empty Space.",
		metadata.importDocumentTypes,
	)
	registerRead(
		server, handler, "list_document_notes", "List bounded note previews and optional history.",
		notes.listDocumentNotes,
	)
	registerRead(
		server, handler, "get_document_note", "Read a bounded note body and its metadata.",
		notes.getDocumentNote,
	)
	registerWrite(
		server, handler, "create_document_note", "Create an authored document note.",
		notes.createDocumentNote,
	)
	registerWrite(
		server, handler, "edit_document_note", "Edit a current note with existing author permissions.",
		notes.editDocumentNote,
	)
	registerWrite(
		server, handler, "replace_document_note", "Replace a current note and retain its history.",
		notes.replaceDocumentNote,
	)
	registerWrite(
		server, handler, "delete_document_note", "Delete a current note into read-only history.",
		notes.deleteDocumentNote,
	)
	registerWrite(
		server, handler, "rename_file", "Rename a filed document or directory.", documents.renameFile,
	)
	registerWrite(
		server, handler, "move_file", "Move a filed document or directory.", documents.moveFile,
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
