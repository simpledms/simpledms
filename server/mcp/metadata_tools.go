package mcp

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/attribute"
	documenttypequery "github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/attributetype"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/library"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/model/tenant/tenantdatamigration"
	"github.com/simpledms/simpledms/util/e"
)

// metadataTools exposes the Space's Tags, fields, and document types.
type metadataTools struct{}

func newMetadataTools() *metadataTools {
	return &metadataTools{}
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

func (qq *metadataTools) listTags(
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
	tags, result.HasMore, result.NextOffset = trimPage(tags, input.Offset, limit)
	for _, tagx := range tags {
		data, err := tagProjection(tagx)
		if err != nil {
			return result, err
		}
		result.Tags = append(result.Tags, data)
	}
	return result, nil
}

func (qq *metadataTools) listProperties(
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
	properties, result.HasMore, result.NextOffset = trimPage(properties, input.Offset, limit)
	for _, propertyx := range properties {
		data, err := propertyProjection(propertyx)
		if err != nil {
			return result, err
		}
		result.Properties = append(result.Properties, data)
	}
	return result, nil
}

func (qq *metadataTools) listDocumentTypes(
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
	documentTypes, result.HasMore, result.NextOffset = trimPage(documentTypes, input.Offset, limit)
	for _, documentTypex := range documentTypes {
		data, err := documentTypeProjection(documentTypex)
		if err != nil {
			return result, err
		}
		result.DocumentTypes = append(result.DocumentTypes, data)
	}
	return result, nil
}

func (qq *metadataTools) getDocumentType(
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
		data, err := documentTypeAttributeProjection(attributex)
		if err != nil {
			return DocumentTypeData{}, err
		}
		result.Attributes = append(result.Attributes, data)
	}
	return result, nil
}

func documentTypeAttributeProjection(
	attributex *enttenant.Attribute,
) (DocumentTypeAttributeData, error) {
	data := DocumentTypeAttributeData{
		Type:         attributex.Type.String(),
		Name:         attributex.Name,
		IsNameGiving: attributex.IsNameGiving,
		IsProtected:  attributex.IsProtected,
		IsDisabled:   attributex.IsDisabled,
		IsRequired:   attributex.IsRequired,
	}
	var err error
	if attributex.Type == attributetype.Tag && attributex.Edges.Tag != nil {
		data.TagID, err = metadataPublicID(attributex.Edges.Tag.PublicID)
		if err != nil {
			return DocumentTypeAttributeData{}, err
		}
	}
	if attributex.Type == attributetype.Field && attributex.Edges.Property != nil {
		data.PropertyID, err = metadataPublicID(attributex.Edges.Property.PublicID)
		if err != nil {
			return DocumentTypeAttributeData{}, err
		}
	}
	return data, nil
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

func (qq *metadataTools) createTag(
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

func (qq *metadataTools) editTag(
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

func (qq *metadataTools) deleteTag(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input TagInput,
) (MetadataDeletionData, error) {
	tagx, err := scopedTag(ctx, input.TagID)
	if err != nil {
		return MetadataDeletionData{}, err
	}
	// The MCP spec forbids clearing assignments to force a deletion; the UI's Delete does that.
	_, err = taggingmodel.NewTagService().DeleteUnused(ctx, tagx.ID)
	return MetadataDeletionData{Deleted: err == nil}, err
}

func (qq *metadataTools) createProperty(
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

func (qq *metadataTools) editProperty(
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

func (qq *metadataTools) deleteProperty(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input PropertyInput,
) (MetadataDeletionData, error) {
	propertyx, err := scopedProperty(ctx, input.PropertyID)
	if err != nil {
		return MetadataDeletionData{}, err
	}
	err = propertymodel.NewPropertyService().Delete(ctx, ctx.Space, propertyx.ID)
	return MetadataDeletionData{Deleted: err == nil}, err
}

func (qq *metadataTools) createDocumentType(
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

func (qq *metadataTools) renameDocumentType(
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

func (qq *metadataTools) deleteDocumentType(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input DocumentTypeInput,
) (MetadataDeletionData, error) {
	documentTypex, err := scopedDocumentType(ctx, input.DocumentTypeID)
	if err != nil {
		return MetadataDeletionData{}, err
	}
	err = documentTypex.Delete(ctx)
	return MetadataDeletionData{Deleted: err == nil}, err
}

func (qq *metadataTools) createAndAssignTag(
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
		return TagAssignmentData{}, e.NewHTTPErrorf(http.StatusBadRequest, fileIsFolderMessage)
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

func (qq *metadataTools) moveTagToGroup(
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

func (qq *metadataTools) assignSubTag(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input TagCompositionInput,
) (TagData, error) {
	return qq.setTagComposition(ctx, input, true)
}

func (qq *metadataTools) unassignSubTag(
	_ context.Context, ctx *ctxx.SpaceContext, _ *entmain.MCPCredential, input TagCompositionInput,
) (TagData, error) {
	return qq.setTagComposition(ctx, input, false)
}

func (qq *metadataTools) setTagComposition(
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

func (qq *metadataTools) createDocumentTypeTagAttribute(
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

func (qq *metadataTools) createDocumentTypePropertyAttribute(
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

func (qq *metadataTools) editDocumentTypeTagAttribute(
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

func (qq *metadataTools) editDocumentTypePropertyAttribute(
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

func (qq *metadataTools) deleteDocumentTypeAttribute(
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

func (qq *metadataTools) listDocumentTypeTemplates(
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

func (qq *metadataTools) importDocumentTypes(
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
