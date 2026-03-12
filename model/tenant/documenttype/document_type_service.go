package documenttype

import (
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/attribute"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/main/common/attributetype"
	"github.com/simpledms/simpledms/model/tenant/library"
	"github.com/simpledms/simpledms/util/e"
)

type DocumentTypeService struct{}

func NewDocumentTypeService() *DocumentTypeService {
	return &DocumentTypeService{}
}

func (qq *DocumentTypeService) Create(
	ctx ctxx.Context,
	spaceID int64,
	name string,
) (*enttenant.DocumentType, error) {
	return ctx.SpaceCtx().TTx.DocumentType.
		Create().
		SetName(name).
		SetSpaceID(spaceID).
		Save(ctx)
}

func (qq *DocumentTypeService) Rename(
	ctx ctxx.Context,
	spaceID int64,
	documentTypeID int64,
	newName string,
) error {
	return ctx.TenantCtx().TTx.DocumentType.
		UpdateOneID(documentTypeID).
		Where(documenttype.SpaceID(spaceID)).
		SetName(newName).
		Exec(ctx)
}

func (qq *DocumentTypeService) Delete(ctx ctxx.Context, spaceID int64, documentTypeID int64) error {
	return ctx.TenantCtx().TTx.DocumentType.
		DeleteOneID(documentTypeID).
		Where(documenttype.SpaceID(spaceID)).
		Exec(ctx)
}

func (qq *DocumentTypeService) ImportFromLibrary(ctx ctxx.Context, templateKeys []string) error {
	service := library.NewService()
	if service.SpaceHasMetadata(ctx) {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Import is only available for empty spaces.")
	}

	if len(templateKeys) == 0 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Please select at least one document type.")
	}

	return service.ImportBuiltinDocumentTypes(ctx, templateKeys, true)
}

func (qq *DocumentTypeService) CreateTagAttribute(
	ctx ctxx.Context,
	space *enttenant.Space,
	documentTypeID int64,
	name string,
	tagID int64,
	isNameGiving bool,
) (*enttenant.Attribute, error) {
	exists, err := ctx.SpaceCtx().TTx.Attribute.Query().
		Where(
			attribute.DocumentTypeID(documentTypeID),
			attribute.TagID(tagID),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}

	if exists {
		tagx, err := space.QueryTags().Where(tag.ID(tagID)).Only(ctx)
		if err != nil {
			return nil, err
		}

		return nil, e.NewHTTPErrorf(
			http.StatusBadRequest,
			"Tag group «%s» is already added to this document type.",
			tagx.Name,
		)
	}

	return ctx.TenantCtx().TTx.Attribute.Create().
		SetName(name).
		SetTagID(tagID).
		SetType(attributetype.Tag).
		SetIsNameGiving(isNameGiving).
		SetDocumentTypeID(documentTypeID).
		SetSpaceID(space.ID).
		Save(ctx)
}

func (qq *DocumentTypeService) CreatePropertyAttribute(
	ctx ctxx.Context,
	space *enttenant.Space,
	documentTypeID int64,
	propertyID int64,
	isNameGiving bool,
) (*enttenant.Attribute, error) {
	exists, err := ctx.SpaceCtx().TTx.Attribute.Query().
		Where(
			attribute.DocumentTypeID(documentTypeID),
			attribute.PropertyID(propertyID),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}

	if exists {
		propertyx, err := space.QueryProperties().Where(property.ID(propertyID)).Only(ctx)
		if err != nil {
			return nil, err
		}

		return nil, e.NewHTTPErrorf(
			http.StatusBadRequest,
			"Field «%s» is already added to this document type.",
			propertyx.Name,
		)
	}

	return ctx.TenantCtx().TTx.Attribute.Create().
		SetType(attributetype.Field).
		SetDocumentTypeID(documentTypeID).
		SetPropertyID(propertyID).
		SetIsNameGiving(isNameGiving).
		SetSpaceID(space.ID).
		Save(ctx)
}

func (qq *DocumentTypeService) EditPropertyAttribute(
	ctx ctxx.Context,
	attributeID int64,
	isNameGiving bool,
) error {
	return ctx.TenantCtx().TTx.Attribute.UpdateOneID(attributeID).
		SetIsNameGiving(isNameGiving).
		Exec(ctx)
}

func (qq *DocumentTypeService) EditTagAttribute(
	ctx ctxx.Context,
	attributeID int64,
	newName string,
	isNameGiving bool,
) error {
	return ctx.TenantCtx().TTx.Attribute.UpdateOneID(attributeID).
		SetName(newName).
		SetIsNameGiving(isNameGiving).
		Exec(ctx)
}

func (qq *DocumentTypeService) DeleteAttribute(ctx ctxx.Context, attributeID int64) error {
	return ctx.TenantCtx().TTx.Attribute.DeleteOneID(attributeID).Exec(ctx)
}
