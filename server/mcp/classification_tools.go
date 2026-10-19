package mcp

import (
	"context"
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	documenttypequery "github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/resolvedtagassignment"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/timex"
)

// classificationTools exposes Tag, field, and document type assignments of documents.
type classificationTools struct{}

func newClassificationTools() *classificationTools {
	return &classificationTools{}
}

func (qq *classificationTools) assignTag(
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

func (qq *classificationTools) unassignTag(
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
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, fileIsFolderMessage)
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

func (qq *classificationTools) setFileProperty(
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

func (qq *classificationTools) removeFileProperty(
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
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, fileIsFolderMessage)
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

func (qq *classificationTools) setDocumentType(
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

func (qq *classificationTools) clearDocumentType(
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
			http.StatusBadRequest, fileIsFolderMessage,
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
		return nil, nil, e.NewHTTPErrorf(http.StatusBadRequest, fileIsFolderMessage)
	}
	documentTypex, err := ctx.Space.QueryDocumentTypes().Where(
		documenttypequery.PublicID(entx.NewCIText(documentTypeID)),
	).Only(ctx)
	if err != nil {
		return nil, nil, metadataNotFound(err, "Document type not found.")
	}
	return filex, documentTypex, nil
}

func addClassification(
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
