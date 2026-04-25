package documenttype

import (
	"net/http"

	"github.com/simpledms/simpledms/core/util/e"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/attribute"
	documenttypequery "github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tenant/common/attributetype"
	"github.com/simpledms/simpledms/model/tenant/library"
)

type DocumentType struct {
	Data *enttenant.DocumentType
}

func NewDocumentType(data *enttenant.DocumentType) *DocumentType {
	return &DocumentType{
		Data: data,
	}
}

func Create(
	ctx ctxx.Context,
	spaceID int64,
	name string,
) (*DocumentType, error) {
	documentTypex, err := ctx.SpaceCtx().TTx.DocumentType.
		Create().
		SetName(name).
		SetSpaceID(spaceID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return NewDocumentType(documentTypex), nil
}

func QueryByID(
	ctx ctxx.Context,
	spaceID int64,
	documentTypeID int64,
) (*DocumentType, error) {
	documentTypex, err := ctx.TenantCtx().TTx.DocumentType.Query().
		Where(
			documenttypequery.ID(documentTypeID),
			documenttypequery.SpaceID(spaceID),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return NewDocumentType(documentTypex), nil
}

func (qq *DocumentType) Rename(ctx ctxx.Context, newName string) error {
	documentTypex, err := ctx.TenantCtx().TTx.DocumentType.
		UpdateOneID(qq.Data.ID).
		Where(documenttypequery.SpaceID(qq.Data.SpaceID)).
		SetName(newName).
		Save(ctx)
	if err != nil {
		return err
	}

	qq.Data = documentTypex

	return nil
}

func (qq *DocumentType) Delete(ctx ctxx.Context) error {
	return ctx.TenantCtx().TTx.DocumentType.
		DeleteOneID(qq.Data.ID).
		Where(documenttypequery.SpaceID(qq.Data.SpaceID)).
		Exec(ctx)
}

func ImportFromLibrary(ctx ctxx.Context, templateKeys []string) error {
	service := library.NewService()
	if service.SpaceHasMetadata(ctx) {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Import is only available for empty spaces.")
	}

	if len(templateKeys) == 0 {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Please select at least one document type.")
	}

	return service.ImportBuiltinDocumentTypes(ctx, templateKeys, true)
}

func (qq *DocumentType) CreateTagAttribute(
	ctx ctxx.Context,
	name string,
	tagID int64,
	isNameGiving bool,
) (*enttenant.Attribute, error) {
	exists, err := ctx.SpaceCtx().TTx.Attribute.Query().
		Where(
			attribute.DocumentTypeID(qq.Data.ID),
			attribute.TagID(tagID),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}

	if exists {
		tagx, err := qq.Data.QuerySpace().QueryTags().Where(tag.ID(tagID)).Only(ctx)
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
		SetDocumentTypeID(qq.Data.ID).
		SetSpaceID(qq.Data.SpaceID).
		Save(ctx)
}

func (qq *DocumentType) CreatePropertyAttribute(
	ctx ctxx.Context,
	propertyID int64,
	isNameGiving bool,
) (*enttenant.Attribute, error) {
	exists, err := ctx.SpaceCtx().TTx.Attribute.Query().
		Where(
			attribute.DocumentTypeID(qq.Data.ID),
			attribute.PropertyID(propertyID),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}

	if exists {
		propertyx, err := qq.Data.QuerySpace().QueryProperties().Where(property.ID(propertyID)).Only(ctx)
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
		SetDocumentTypeID(qq.Data.ID).
		SetPropertyID(propertyID).
		SetIsNameGiving(isNameGiving).
		SetSpaceID(qq.Data.SpaceID).
		Save(ctx)
}
