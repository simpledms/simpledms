package tagging

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

type EntTagRepository struct{}

var _ TagRepository = (*EntTagRepository)(nil)

func NewEntTagRepository() *EntTagRepository {
	return &EntTagRepository{}
}

func (qq *EntTagRepository) AssignTagToFile(ctx ctxx.Context, fileID int64, tagID int64, spaceID int64) error {
	_, err := ctx.TenantCtx().TTx.TagAssignment.Create().
		SetFileID(fileID).
		SetTagID(tagID).
		SetSpaceID(spaceID).
		Save(ctx)

	return err
}

func (qq *EntTagRepository) UnassignTagFromFile(ctx ctxx.Context, fileID int64, tagID int64) error {
	_, err := ctx.TenantCtx().TTx.TagAssignment.Delete().
		Where(tagassignment.FileID(fileID), tagassignment.TagID(tagID)).
		Exec(ctx)

	return err
}

func (qq *EntTagRepository) UnassignTagFromFileInSpace(
	ctx ctxx.Context,
	fileID int64,
	tagID int64,
	spaceID int64,
) error {
	_, err := ctx.TenantCtx().TTx.TagAssignment.Delete().
		Where(
			tagassignment.FileID(fileID),
			tagassignment.TagID(tagID),
			tagassignment.SpaceID(spaceID),
		).
		Exec(ctx)

	return err
}

func (qq *EntTagRepository) SubTagWithChildren(ctx ctxx.Context, subTagID int64) (*enttenant.Tag, error) {
	return ctx.TenantCtx().TTx.Tag.Query().
		WithChildren(func(query *enttenant.TagQuery) {
			query.Order(tag.ByName())
			query.Where(tag.TypeNEQ(tagtype.Super))
		}).
		Where(tag.ID(subTagID)).
		Only(ctx)
}

func (qq *EntTagRepository) CreateTag(
	ctx ctxx.Context,
	spaceID int64,
	groupTagID int64,
	name string,
	typex tagtype.TagType,
) (*enttenant.Tag, error) {
	tagCreate := ctx.TenantCtx().TTx.Tag.Create().
		SetName(name).
		SetType(typex).
		SetSpaceID(spaceID)

	if groupTagID != 0 {
		tagCreate.SetGroupID(groupTagID)
	}

	return tagCreate.Save(ctx)
}

func (qq *EntTagRepository) UpdateTagName(ctx ctxx.Context, tagID int64, name string) (*enttenant.Tag, error) {
	return ctx.TenantCtx().TTx.Tag.UpdateOneID(tagID).SetName(name).Save(ctx)
}

func (qq *EntTagRepository) DeleteTag(ctx ctxx.Context, tagID int64) error {
	return ctx.TenantCtx().TTx.Tag.DeleteOneID(tagID).Exec(ctx)
}

func (qq *EntTagRepository) TagByID(ctx ctxx.Context, tagID int64) (*enttenant.Tag, error) {
	return ctx.TenantCtx().TTx.Tag.Get(ctx, tagID)
}

func (qq *EntTagRepository) FileByID(ctx ctxx.Context, fileID int64) (*enttenant.File, error) {
	return ctx.TenantCtx().TTx.File.Get(ctx, fileID)
}

func (qq *EntTagRepository) FileHasTagAssignment(ctx ctxx.Context, fileID int64, tagID int64) (bool, error) {
	return ctx.TenantCtx().TTx.TagAssignment.Query().
		Where(
			tagassignment.FileID(fileID),
			tagassignment.TagID(tagID),
		).
		Exist(ctx)
}

func (qq *EntTagRepository) ClearTagGroup(ctx ctxx.Context, tagID int64) error {
	_, err := ctx.TenantCtx().TTx.Tag.UpdateOneID(tagID).ClearGroupID().Save(ctx)

	return err
}

func (qq *EntTagRepository) SetTagGroup(ctx ctxx.Context, tagID int64, groupTagID int64) error {
	_, err := ctx.TenantCtx().TTx.Tag.UpdateOneID(tagID).SetGroupID(groupTagID).Save(ctx)

	return err
}

func (qq *EntTagRepository) AddSubTag(ctx ctxx.Context, superTagID int64, subTagID int64) (*enttenant.Tag, error) {
	return ctx.TenantCtx().TTx.Tag.
		UpdateOneID(superTagID).
		AddSubTagIDs(subTagID).
		Save(ctx)
}

func (qq *EntTagRepository) RemoveSubTag(ctx ctxx.Context, superTagID int64, subTagID int64) (*enttenant.Tag, error) {
	return ctx.TenantCtx().TTx.Tag.
		UpdateOneID(superTagID).
		RemoveSubTagIDs(subTagID).
		Save(ctx)
}

func (qq *EntTagRepository) GroupTags(ctx ctxx.Context) ([]*enttenant.Tag, error) {
	return ctx.TenantCtx().TTx.Tag.Query().Where(tag.TypeEQ(tagtype.Group)).All(ctx)
}
