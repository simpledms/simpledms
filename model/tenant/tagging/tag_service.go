package tagging

import (
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/util/e"
)

type TagService struct {
	repository TagRepository
}

func NewTagService() *TagService {
	return NewTagServiceWithRepository(NewEntTagRepository())
}

func NewTagServiceWithRepository(repository TagRepository) *TagService {
	return &TagService{
		repository: repository,
	}
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
	if groupTagID != 0 {
		if _, err := qq.group(ctx, groupTagID); err != nil {
			return nil, err
		}
	}

	return qq.repository.CreateTag(ctx, spaceID, groupTagID, name, typex)
}

func (qq *TagService) Edit(ctx ctxx.Context, tagID int64, name string) (*enttenant.Tag, error) {
	return qq.repository.UpdateTagName(ctx, tagID, name)
}

// CreateAndAssignToFile creates an assignable Tag and assigns it in the caller's transaction.
func (qq *TagService) CreateAndAssignToFile(
	ctx ctxx.Context,
	fileID int64,
	spaceID int64,
	groupTagID int64,
	name string,
	typex tagtype.TagType,
) (*enttenant.Tag, error) {
	if typex == tagtype.Group {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Tag groups cannot be assigned.")
	}
	filex, err := qq.repository.FileByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if filex.SpaceID != spaceID {
		return nil, e.NewHTTPErrorf(http.StatusNotFound, "File not found.")
	}
	tagx, err := qq.Create(ctx, spaceID, groupTagID, name, typex)
	if err != nil {
		return nil, err
	}
	return qq.AssignToFile(ctx, fileID, tagx.ID, spaceID)
}

func (qq *TagService) Delete(ctx ctxx.Context, tagID int64) (string, error) {
	tagx, err := qq.repository.TagByID(ctx, tagID)
	if err != nil {
		return "", err
	}

	err = qq.repository.DeleteTag(ctx, tagID)
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
	filex, err := qq.repository.FileByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	tagx, err := qq.repository.TagByID(ctx, tagID)
	if err != nil {
		return nil, err
	}
	if filex.SpaceID != spaceID || tagx.SpaceID != spaceID {
		return nil, e.NewHTTPErrorf(http.StatusNotFound, "File or Tag not found.")
	}
	if tagx.Type == tagtype.Group {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Tag groups cannot be assigned.")
	}
	assigned, err := qq.repository.FileHasTagAssignment(ctx, fileID, tagID)
	if err != nil || assigned {
		return tagx, err
	}
	if err := qq.repository.AssignTagToFile(ctx, fileID, tagID, spaceID); err != nil {
		return nil, err
	}
	return tagx, nil
}

func (qq *TagService) UnassignFromFile(
	ctx ctxx.Context,
	fileID int64,
	tagID int64,
) (*enttenant.Tag, error) {
	filex, err := qq.repository.FileByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	tagx, err := qq.repository.TagByID(ctx, tagID)
	if err != nil {
		return nil, err
	}
	if filex.SpaceID != ctx.SpaceCtx().Space.ID || tagx.SpaceID != ctx.SpaceCtx().Space.ID {
		return nil, e.NewHTTPErrorf(http.StatusNotFound, "File or Tag not found.")
	}
	if err := qq.repository.UnassignTagFromFileInSpace(
		ctx, fileID, tagID, ctx.SpaceCtx().Space.ID,
	); err != nil {
		return nil, err
	}
	return tagx, nil
}

func (qq *TagService) ToggleFileTag(
	ctx ctxx.Context,
	fileID int64,
	tagID int64,
	spaceID int64,
) (bool, *enttenant.Tag, error) {
	_, err := qq.repository.FileByID(ctx, fileID)
	if err != nil {
		return false, nil, err
	}

	tagx, err := qq.repository.TagByID(ctx, tagID)
	if err != nil {
		return false, nil, err
	}

	isSelected, err := qq.repository.FileHasTagAssignment(ctx, fileID, tagID)
	if err != nil {
		return false, nil, err
	}

	if isSelected {
		err = qq.repository.UnassignTagFromFileInSpace(ctx, fileID, tagID, spaceID)
		if err != nil {
			return false, nil, err
		}

		return false, tagx, nil
	}

	err = qq.repository.AssignTagToFile(ctx, fileID, tagID, spaceID)
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
	tagx, err := qq.repository.TagByID(ctx, tagID)
	if err != nil {
		return false, nil, err
	}
	if tagx.Type == tagtype.Group {
		return false, nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot add a tag group as child.")
	}
	if groupTagID == 0 {
		err := qq.repository.ClearTagGroup(ctx, tagID)
		if err != nil {
			return false, nil, err
		}

		return true, nil, nil
	}

	groupTag, err := qq.group(ctx, groupTagID)
	if err != nil {
		return false, nil, err
	}
	err = qq.repository.SetTagGroup(ctx, tagID, groupTagID)
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
	if err := qq.nilableCompositionError(ctx, superTagID, subTagID); err != nil {
		return nil, nil, err
	}
	superTag, err := qq.repository.AddSubTag(ctx, superTagID, subTagID)
	if err != nil {
		return nil, nil, err
	}

	subTag, err := qq.repository.SubTagWithChildren(ctx, subTagID)
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
	if err := qq.nilableCompositionError(ctx, superTagID, subTagID); err != nil {
		return nil, nil, err
	}
	superTag, err := qq.repository.RemoveSubTag(ctx, superTagID, subTagID)
	if err != nil {
		return nil, nil, err
	}

	subTag, err := qq.repository.SubTagWithChildren(ctx, subTagID)
	if err != nil {
		return nil, nil, err
	}

	return superTag, subTag, nil
}

func (qq *TagService) Get(ctx ctxx.Context, tagID int64) (*enttenant.Tag, error) {
	return qq.repository.TagByID(ctx, tagID)
}

func (qq *TagService) GroupTags(ctx ctxx.Context) ([]*enttenant.Tag, error) {
	return qq.repository.GroupTags(ctx)
}

func (qq *TagService) group(ctx ctxx.Context, tagID int64) (*enttenant.Tag, error) {
	tagx, err := qq.repository.TagByID(ctx, tagID)
	if err != nil {
		return nil, err
	}
	if tagx.Type != tagtype.Group {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Destination must be a Tag group.")
	}
	return tagx, nil
}

func (qq *TagService) nilableCompositionError(ctx ctxx.Context, superTagID, subTagID int64) error {
	superTag, err := qq.repository.TagByID(ctx, superTagID)
	if err != nil {
		return err
	}
	subTag, err := qq.repository.TagByID(ctx, subTagID)
	if err != nil {
		return err
	}
	if superTag.Type != tagtype.Super || subTag.Type != tagtype.Simple {
		return e.NewHTTPErrorf(
			http.StatusBadRequest, "Composition requires a composed Tag and a simple sub-Tag.",
		)
	}
	return nil
}
