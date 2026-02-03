package library

import (
	"net/http"
	"slices"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/attribute"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/common/attributetype"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/e"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (qq *Service) SpaceHasMetadata(ctx ctxx.Context) bool {
	documentTypeCount := ctx.SpaceCtx().Space.QueryDocumentTypes().CountX(ctx)
	if documentTypeCount > 0 {
		return true
	}
	tagCount := ctx.SpaceCtx().Space.QueryTags().CountX(ctx)
	if tagCount > 0 {
		return true
	}
	fieldCount := ctx.SpaceCtx().Space.QueryProperties().CountX(ctx)
	if fieldCount > 0 {
		return true
	}
	return false
}

func (qq *Service) ImportDocumentTypes(ctx ctxx.Context, templateKeys []string, requireEmpty bool) error {
	return qq.ImportBuiltinDocumentTypes(ctx, templateKeys, requireEmpty)
}

func (qq *Service) ImportBuiltinDocumentTypes(ctx ctxx.Context, templateKeys []string, requireEmpty bool) error {
	if len(templateKeys) == 0 {
		return nil
	}

	if requireEmpty && qq.SpaceHasMetadata(ctx) {
		return e.NewHTTPErrorf(http.StatusBadRequest, wx.T("Import is only available for empty spaces.").String(ctx))
	}

	keys := qq.SortTemplateKeys(templateKeys)
	byKey := map[string]BuiltinTemplate{}
	for _, template := range BuiltinTemplates() {
		byKey[template.Key] = template
	}

	var selected []BuiltinTemplate
	for _, key := range keys {
		if template, ok := byKey[key]; ok {
			selected = append(selected, template)
		}
	}

	if len(selected) == 0 {
		return nil
	}

	return qq.importBuiltinTemplates(ctx, selected)
}

func (qq *Service) importBuiltinTemplates(ctx ctxx.Context, templates []BuiltinTemplate) error {
	groupTags := map[string]*enttenant.Tag{}
	tags := map[string]*enttenant.Tag{}
	fields := map[string]*enttenant.Property{}

	for _, template := range templates {
		for _, tagx := range template.Tags {
			if tagx.Type != tagtype.Group {
				continue
			}
			if _, exists := groupTags[tagx.Key]; exists {
				continue
			}
			name := wx.T(tagx.Name).String(ctx)
			spaceTag, err := qq.ensureSpaceTag(ctx, name, tagx.Type, 0, tagx.Color, tagx.Icon)
			if err != nil {
				return err
			}
			groupTags[tagx.Key] = spaceTag
			tags[tagx.Key] = spaceTag
		}
	}

	for _, template := range templates {
		for _, tagx := range template.Tags {
			if tagx.Type == tagtype.Group {
				continue
			}
			if _, exists := tags[tagx.Key]; exists {
				continue
			}
			parent := groupTags[tagx.GroupKey]
			if parent == nil {
				continue
			}
			name := wx.T(tagx.Name).String(ctx)
			spaceTag, err := qq.ensureSpaceTag(ctx, name, tagx.Type, parent.ID, tagx.Color, tagx.Icon)
			if err != nil {
				return err
			}
			tags[tagx.Key] = spaceTag
		}
	}

	for _, template := range templates {
		for _, fieldx := range template.Fields {
			if _, exists := fields[fieldx.Key]; exists {
				continue
			}
			name := wx.T(fieldx.Name).String(ctx)
			spaceField, err := qq.ensureSpaceField(ctx, name, fieldx.Type, fieldx.Unit)
			if err != nil {
				return err
			}
			fields[fieldx.Key] = spaceField
		}
	}

	for _, template := range templates {
		docName := wx.T(template.Name).String(ctx)
		spaceDocType, err := qq.ensureSpaceDocumentType(ctx, docName, template.Icon)
		if err != nil {
			return err
		}

		for _, attributex := range template.Attributes {
			if attributex.Type == attributetype.Tag {
				spaceTag := tags[attributex.TagKey]
				if spaceTag == nil {
					continue
				}
				attributeName := attributex.Name
				if attributeName != "" {
					attributeName = wx.T(attributeName).String(ctx)
				}
				if err := qq.ensureSpaceTagAttribute(ctx, spaceDocType.ID, spaceTag.ID, attributeName, attributex.IsRequired, attributex.IsNameGiving); err != nil {
					return err
				}
				continue
			}

			if attributex.Type == attributetype.Field {
				spaceField := fields[attributex.FieldKey]
				if spaceField == nil {
					continue
				}
				if err := qq.ensureSpaceFieldAttribute(ctx, spaceDocType.ID, spaceField.ID, attributex.IsRequired, attributex.IsNameGiving); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (qq *Service) ensureSpaceTag(ctx ctxx.Context, name string, tagType tagtype.TagType, groupID int64, color string, icon string) (*enttenant.Tag, error) {
	query := ctx.SpaceCtx().Space.QueryTags().Where(
		tag.NameEQ(name),
		tag.TypeEQ(tagType),
	)
	if groupID == 0 {
		query = query.Where(tag.GroupIDIsNil())
	} else {
		query = query.Where(tag.GroupID(groupID))
	}
	existing, err := query.Only(ctx)
	if err == nil {
		return existing, nil
	}
	if !enttenant.IsNotFound(err) {
		return nil, err
	}

	create := ctx.TenantCtx().TTx.Tag.Create().
		SetName(name).
		SetType(tagType).
		SetSpaceID(ctx.SpaceCtx().Space.ID)
	if groupID != 0 {
		create.SetGroupID(groupID)
	}
	if color != "" {
		create.SetColor(color)
	}
	if icon != "" {
		create.SetIcon(icon)
	}
	return create.Save(ctx)
}

func (qq *Service) ensureSpaceField(ctx ctxx.Context, name string, fieldType fieldtype.FieldType, unit string) (*enttenant.Property, error) {
	query := ctx.SpaceCtx().Space.QueryProperties().Where(
		property.NameEQ(name),
		property.TypeEQ(fieldType),
	)
	if unit != "" {
		query = query.Where(property.UnitEQ(unit))
	}
	existing, err := query.Only(ctx)
	if err == nil {
		return existing, nil
	}
	if !enttenant.IsNotFound(err) {
		return nil, err
	}
	return ctx.TenantCtx().TTx.Property.Create().
		SetName(name).
		SetType(fieldType).
		SetUnit(unit).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		Save(ctx)
}

func (qq *Service) ensureSpaceDocumentType(ctx ctxx.Context, name string, icon string) (*enttenant.DocumentType, error) {
	existing, err := ctx.SpaceCtx().Space.QueryDocumentTypes().
		Where(documenttype.NameEQ(name)).
		Only(ctx)
	if err == nil {
		return existing, nil
	}
	if !enttenant.IsNotFound(err) {
		return nil, err
	}
	create := ctx.TenantCtx().TTx.DocumentType.Create().
		SetName(name).
		SetSpaceID(ctx.SpaceCtx().Space.ID)
	if icon != "" {
		create.SetIcon(icon)
	}
	return create.Save(ctx)
}

func (qq *Service) ensureSpaceTagAttribute(ctx ctxx.Context, documentTypeID int64, tagID int64, name string, isRequired bool, isNameGiving bool) error {
	exists := ctx.SpaceCtx().TTx.Attribute.Query().
		Where(
			attribute.DocumentTypeID(documentTypeID),
			attribute.TagID(tagID),
		).
		ExistX(ctx)
	if exists {
		return nil
	}
	create := ctx.TenantCtx().TTx.Attribute.Create().
		SetDocumentTypeID(documentTypeID).
		SetTagID(tagID).
		SetType(attributetype.Tag).
		SetIsRequired(isRequired).
		SetIsNameGiving(isNameGiving).
		SetSpaceID(ctx.SpaceCtx().Space.ID)
	if name != "" {
		create.SetName(name)
	}
	_, err := create.Save(ctx)
	return err
}

func (qq *Service) ensureSpaceFieldAttribute(ctx ctxx.Context, documentTypeID int64, fieldID int64, isRequired bool, isNameGiving bool) error {
	exists := ctx.SpaceCtx().TTx.Attribute.Query().
		Where(
			attribute.DocumentTypeID(documentTypeID),
			attribute.PropertyID(fieldID),
		).
		ExistX(ctx)
	if exists {
		return nil
	}
	_, err := ctx.TenantCtx().TTx.Attribute.Create().
		SetDocumentTypeID(documentTypeID).
		SetPropertyID(fieldID).
		SetType(attributetype.Field).
		SetIsRequired(isRequired).
		SetIsNameGiving(isNameGiving).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		Save(ctx)
	return err
}

func (qq *Service) SortTemplateKeys(keys []string) []string {
	slices.Sort(keys)
	keys = slices.Compact(keys)
	return keys
}
