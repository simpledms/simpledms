package tagging

import (
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/util/e"
)

type TagService struct{}

func NewTagService() *TagService {
	return &TagService{}
}

func (qq *TagService) assignTagToFile(ctx ctxx.Context, fileID int64, tagID int64, spaceID int64) error {
	_, err := ctx.TenantCtx().TTx.TagAssignment.Create().
		SetFileID(fileID).
		SetTagID(tagID).
		SetSpaceID(spaceID).
		Save(ctx)

	return err
}

func (qq *TagService) unassignTagFromFile(ctx ctxx.Context, fileID int64, tagID int64) error {
	_, err := ctx.TenantCtx().TTx.TagAssignment.Delete().
		Where(tagassignment.FileID(fileID), tagassignment.TagID(tagID)).
		Exec(ctx)

	return err
}

func (qq *TagService) unassignTagFromFileInSpace(
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

func (qq *TagService) querySubTagWithChildren(ctx ctxx.Context, subTagID int64) (*enttenant.Tag, error) {
	return ctx.TenantCtx().TTx.Tag.Query().
		WithChildren(func(query *enttenant.TagQuery) {
			query.Order(tag.ByName())
			query.Where(tag.TypeNEQ(tagtype.Super))
		}).
		Where(tag.ID(subTagID)).
		Only(ctx)
}

func (qq *TagService) Create(
	ctx ctxx.Context,
	spaceID int64,
	groupTagID int64,
	name string,
	typex tagtype.TagType,
) (*enttenant.Tag, error) {
	if groupTagID != 0 && typex == tagtype.Group {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot add a tag group as child.")
	}

	tagCreate := ctx.TenantCtx().TTx.Tag.Create().
		SetName(name).
		SetType(typex).
		SetSpaceID(spaceID)

	if groupTagID != 0 {
		tagCreate.SetGroupID(groupTagID)
	}

	return tagCreate.Save(ctx)
}

func (qq *TagService) Edit(ctx ctxx.Context, tagID int64, name string) (*enttenant.Tag, error) {
	return ctx.TenantCtx().TTx.Tag.UpdateOneID(tagID).SetName(name).Save(ctx)
}

func (qq *TagService) Delete(ctx ctxx.Context, tagID int64) (string, error) {
	tagx, err := ctx.TenantCtx().TTx.Tag.Get(ctx, tagID)
	if err != nil {
		return "", err
	}

	err = ctx.TenantCtx().TTx.Tag.DeleteOneID(tagID).Exec(ctx)
	if err != nil {
		return "", err
	}

	return tagx.Name, nil
}

func (qq *TagService) AssignToFile(
	ctx ctxx.Context,
	fileID int64,
	tagID int64,
	spaceID int64,
) (*enttenant.Tag, error) {
	err := qq.assignTagToFile(ctx, fileID, tagID, spaceID)
	if err != nil {
		return nil, err
	}

	return ctx.TenantCtx().TTx.Tag.Get(ctx, tagID)
}

func (qq *TagService) UnassignFromFile(
	ctx ctxx.Context,
	fileID int64,
	tagID int64,
) (*enttenant.Tag, error) {
	err := qq.unassignTagFromFile(ctx, fileID, tagID)
	if err != nil {
		return nil, err
	}

	return ctx.TenantCtx().TTx.Tag.Get(ctx, tagID)
}

func (qq *TagService) ToggleFileTag(
	ctx ctxx.Context,
	fileID int64,
	tagID int64,
	spaceID int64,
) (bool, *enttenant.Tag, error) {
	filex, err := ctx.TenantCtx().TTx.File.Get(ctx, fileID)
	if err != nil {
		return false, nil, err
	}

	tagx, err := ctx.TenantCtx().TTx.Tag.Get(ctx, tagID)
	if err != nil {
		return false, nil, err
	}

	isSelected, err := filex.QueryTagAssignment().Where(tagassignment.TagID(tagID)).Exist(ctx)
	if err != nil {
		return false, nil, err
	}

	if isSelected {
		err = qq.unassignTagFromFileInSpace(ctx, fileID, tagID, spaceID)
		if err != nil {
			return false, nil, err
		}

		return false, tagx, nil
	}

	err = qq.assignTagToFile(ctx, fileID, tagID, spaceID)
	if err != nil {
		return false, nil, err
	}

	return true, tagx, nil
}

func (qq *TagService) MoveToGroup(
	ctx ctxx.Context,
	tagID int64,
	groupTagID int64,
) (bool, *enttenant.Tag, error) {
	if groupTagID == 0 {
		_, err := ctx.TenantCtx().TTx.Tag.UpdateOneID(tagID).ClearGroupID().Save(ctx)
		if err != nil {
			return false, nil, err
		}

		return true, nil, nil
	}

	_, err := ctx.TenantCtx().TTx.Tag.UpdateOneID(tagID).SetGroupID(groupTagID).Save(ctx)
	if err != nil {
		return false, nil, err
	}

	groupTag, err := ctx.TenantCtx().TTx.Tag.Get(ctx, groupTagID)
	if err != nil {
		return false, nil, err
	}

	return false, groupTag, nil
}

func (qq *TagService) AssignSubTag(
	ctx ctxx.Context,
	superTagID int64,
	subTagID int64,
) (*enttenant.Tag, *enttenant.Tag, error) {
	superTag, err := ctx.TenantCtx().TTx.Tag.
		UpdateOneID(superTagID).
		AddSubTagIDs(subTagID).
		Save(ctx)
	if err != nil {
		return nil, nil, err
	}

	subTag, err := qq.querySubTagWithChildren(ctx, subTagID)
	if err != nil {
		return nil, nil, err
	}

	return superTag, subTag, nil
}

func (qq *TagService) UnassignSubTag(
	ctx ctxx.Context,
	superTagID int64,
	subTagID int64,
) (*enttenant.Tag, *enttenant.Tag, error) {
	superTag, err := ctx.TenantCtx().TTx.Tag.
		UpdateOneID(superTagID).
		RemoveSubTagIDs(subTagID).
		Save(ctx)
	if err != nil {
		return nil, nil, err
	}

	subTag, err := qq.querySubTagWithChildren(ctx, subTagID)
	if err != nil {
		return nil, nil, err
	}

	return superTag, subTag, nil
}
