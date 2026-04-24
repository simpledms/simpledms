package file

import (
	"log"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/entx"
)

type EntSpaceFileReadRepository struct {
	spaceID int64
}

var _ FileReadRepository = (*EntSpaceFileReadRepository)(nil)

func NewEntSpaceFileReadRepository(spaceID int64) *EntSpaceFileReadRepository {
	return &EntSpaceFileReadRepository{
		spaceID: spaceID,
	}
}

func (qq *EntSpaceFileReadRepository) FileByPublicID(
	ctx ctxx.Context,
	filePublicID string,
) (*FileDTO, error) {
	filex, err := qq.scopedFileQuery(ctx).
		Where(file.PublicIDEQ(entx.NewCIText(filePublicID))).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return entFileToDTO(filex), nil
}

func (qq *EntSpaceFileReadRepository) FileByPublicIDX(ctx ctxx.Context, filePublicID string) *FileDTO {
	filex := qq.scopedFileQuery(ctx).
		Where(file.PublicIDEQ(entx.NewCIText(filePublicID))).
		OnlyX(ctx)

	return entFileToDTO(filex)
}

func (qq *EntSpaceFileReadRepository) FileByID(ctx ctxx.Context, fileID int64) (*FileDTO, error) {
	filex, err := qq.scopedFileQuery(ctx).
		Where(file.ID(fileID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return entFileToDTO(filex), nil
}

func (qq *EntSpaceFileReadRepository) FileByParentIDAndName(
	ctx ctxx.Context,
	parentID int64,
	name string,
) (*FileDTO, error) {
	filex, err := qq.scopedFileQuery(ctx).
		Where(
			file.ParentID(parentID),
			file.Name(name),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return entFileToDTO(filex), nil
}

func (qq *EntSpaceFileReadRepository) FileExistsByNameAndParentX(
	ctx ctxx.Context,
	name string,
	parentID int64,
	isInInbox bool,
) bool {
	return qq.scopedFileQuery(ctx).
		Where(
			file.Name(name),
			file.ParentID(parentID),
			file.IsInInbox(isInInbox),
		).
		ExistX(ctx)
}

func (qq *EntSpaceFileReadRepository) FilesByIDs(
	ctx ctxx.Context,
	fileIDs []int64,
) ([]*FileDTO, error) {
	if len(fileIDs) == 0 {
		return []*FileDTO{}, nil
	}

	filexs, err := qq.scopedFileQuery(ctx).
		Where(file.IDIn(fileIDs...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	fileDTOs := make([]*FileDTO, 0, len(filexs))
	for _, filex := range filexs {
		fileDTOs = append(fileDTOs, entFileToDTO(filex))
	}

	return fileDTOs, nil
}

func (qq *EntSpaceFileReadRepository) FileByPublicIDWithParentX(
	ctx ctxx.Context,
	filePublicID string,
) *FileWithParentDTO {
	filex := qq.scopedFileQuery(ctx).
		WithParent().
		Where(file.PublicIDEQ(entx.NewCIText(filePublicID))).
		OnlyX(ctx)

	return entFileToWithParentDTO(filex)
}

func (qq *EntSpaceFileReadRepository) FileByPublicIDWithChildrenX(
	ctx ctxx.Context,
	filePublicID string,
) *FileWithChildrenDTO {
	filex := qq.scopedFileQuery(ctx).
		WithChildren().
		Where(file.PublicIDEQ(entx.NewCIText(filePublicID))).
		OnlyX(ctx)

	return entFileToWithChildrenDTO(filex)
}

func (qq *EntSpaceFileReadRepository) FileByIDWithChildrenX(
	ctx ctxx.Context,
	fileID int64,
) *FileWithChildrenDTO {
	filex := qq.scopedFileQuery(ctx).
		WithChildren().
		Where(file.ID(fileID)).
		OnlyX(ctx)

	return entFileToWithChildrenDTO(filex)
}

func (qq *EntSpaceFileReadRepository) FileByIDX(ctx ctxx.Context, fileID int64) *FileDTO {
	filex := qq.scopedFileQuery(ctx).
		Where(file.ID(fileID)).
		OnlyX(ctx)

	return entFileToDTO(filex)
}

func (qq *EntSpaceFileReadRepository) FileByPublicIDWithDeletedX(
	ctx ctxx.Context,
	filePublicID string,
) *FileDTO {
	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	filex := qq.scopedFileQuery(ctx).
		Where(file.PublicIDEQ(entx.NewCIText(filePublicID))).
		OnlyX(ctxWithDeleted)

	return entFileToDTO(filex)
}

func (qq *EntSpaceFileReadRepository) ParentDirectoryNameByIDX(ctx ctxx.Context, parentID int64) string {
	if parentID == 0 {
		return ""
	}

	parent, err := qq.scopedFileQuery(ctx).
		Where(file.ID(parentID), file.IsDirectory(true)).
		Only(ctx)
	if err != nil {
		if !enttenant.IsNotFound(err) {
			log.Println(err)
		}
		return ""
	}

	return parent.Name
}

func (qq *EntSpaceFileReadRepository) scopedFileQuery(ctx ctxx.Context) *enttenant.FileQuery {
	return ctx.TenantCtx().TTx.File.Query().Where(file.SpaceID(qq.spaceID))
}
