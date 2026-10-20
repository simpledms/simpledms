package mcp

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	documenttypequery "github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	credentialmodel "github.com/simpledms/simpledms/model/main/mcpcredential"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	filingmodel "github.com/simpledms/simpledms/model/tenant/filing"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
)

// documentTools exposes reading, uploading, filing, and organizing documents.
type documentTools struct {
	infra       *common.Infra
	credentials *credentialmodel.CredentialService
}

func newDocumentTools(
	infra *common.Infra,
	credentials *credentialmodel.CredentialService,
) *documentTools {
	return &documentTools{
		infra:       infra,
		credentials: credentials,
	}
}

func (qq *documentTools) getSpace(
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

func (qq *documentTools) listInbox(
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
		file.FieldOcrMarkdownSuccessAt,
	).Offset(input.Offset).Limit(limit + 1).All(ctx)
	if err != nil {
		return result, err
	}
	files, result.HasMore, result.NextOffset = trimPage(files, input.Offset, limit)
	for _, filex := range files {
		result.Files = append(result.Files, qq.summary(requestCtx, ctx, filex, filex.Edges.Parent))
	}
	return result, nil
}

func (qq *documentTools) summary(
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
		OCRAvailable: filemodel.NewFile(filex).HasLLMText(ctx),
	}
}

func (qq *documentTools) documentURL(
	requestCtx context.Context,
	ctx *ctxx.SpaceContext,
	parentID, fileID string,
	isDirectory bool,
) string {
	path := route.Browse(ctx.TenantID, ctx.SpaceID, fileID)
	if !isDirectory && parentID != "" {
		path = route.BrowseFile(ctx.TenantID, ctx.SpaceID, parentID, fileID)
	}
	url := qq.infra.SystemConfig().AbsoluteURL(path)
	if url == path {
		if request, ok := requestCtx.Value(requestContextKey(0)).(requestContext); ok {
			url = request.origin + path
		}
	}
	return url
}

func (qq *documentTools) fileData(
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
	if err := addClassification(ctx, filex, &data); err != nil {
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

func (qq *documentTools) getFile(
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

func (qq *documentTools) readText(
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
		return result, e.NewHTTPErrorf(http.StatusBadRequest, fileIsFolderMessage)
	}
	data, err := qq.fileData(requestCtx, ctx, filex)
	if err != nil {
		return result, err
	}
	result.VersionNumber = data.VersionNumber
	result.Available = data.OCRAvailable
	if result.Available {
		text, isMarkdown := filemodel.NewFile(filex).LLMText(ctx)
		result.Format = TextFormatText
		if isMarkdown {
			result.Format = TextFormatMarkdown
		}
		result.Text, result.HasMore, err = reader.TextWindow(text, input.Offset, length)
		if result.HasMore {
			next := input.Offset + length
			result.NextOffset = &next
		}
	}
	return result, err
}

func (qq *documentTools) downloadFile(
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
	result, err := filesystem.NewFileDownloadService(qq.infra.FileSystem()).Read(
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

func (qq *documentTools) listDirectory(
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
		qq.infra.FileSystem(),
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

func (qq *documentTools) createDirectory(
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
	directory, err := filingmodel.NewFilingService(qq.infra.FileSystem()).CreateDirectory(
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

func (qq *documentTools) fileInboxDocument(
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
	filex, err := filingmodel.NewFilingService(qq.infra.FileSystem()).FileInboxDocument(
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

func (qq *documentTools) markInboxFileDone(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input FileInput,
) (FilingData, error) {
	if input.FileID == "" || len(input.FileID) > 100 {
		return FilingData{}, e.NewHTTPErrorf(http.StatusBadRequest, "File ID is required.")
	}
	filex, err := filingmodel.NewFilingService(qq.infra.FileSystem()).CompleteInboxFile(
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

func (qq *documentTools) searchFiles(
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
		file.FieldOcrMarkdownSuccessAt,
	).Offset(input.Offset).Limit(limit + 1).All(ctx)
	if err != nil {
		return result, err
	}
	files, result.HasMore, result.NextOffset = trimPage(files, input.Offset, limit)
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

func (qq *documentTools) uploadFile(
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
	configuredLimit, err := qq.infra.FileSystem().NilableEffectiveUploadSizeLimitBytes(ctx)
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
	ingested, err := filesystem.NewFileIngestionService(qq.infra.FileSystem()).Ingest(
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

func (qq *documentTools) renameFile(
	_ context.Context,
	ctx *ctxx.SpaceContext,
	_ *entmain.MCPCredential,
	input RenameFileInput,
) (FilingData, error) {
	if err := nilableMetadataReferenceError(input.FileID); err != nil {
		return FilingData{}, err
	}
	filex, err := filesystem.NewFileOrganizationService(qq.infra.FileSystem()).Rename(
		ctx, input.FileID, input.NewFilename,
	)
	if err != nil {
		return FilingData{}, organizationError(err)
	}
	return filingProjection(ctx, filex)
}

func (qq *documentTools) moveFile(
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
	filex, err := filesystem.NewFileOrganizationService(qq.infra.FileSystem()).Move(
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

// trimPage drops the lookahead item that list queries fetch with Limit(limit+1) to
// detect whether another page exists.
func trimPage[T any](items []T, offset int, limit int) ([]T, bool, *int) {
	if len(items) <= limit {
		return items, false, nil
	}
	next := offset + limit
	return items[:limit], true, &next
}
