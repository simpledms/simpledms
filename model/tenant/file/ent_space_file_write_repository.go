package file

import (
	"net/http"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/util/e"
)

type EntSpaceFileWriteRepository struct {
	spaceID int64
}

var _ FileWriteRepository = (*EntSpaceFileWriteRepository)(nil)

func NewEntSpaceFileWriteRepository(spaceID int64) *EntSpaceFileWriteRepository {
	return &EntSpaceFileWriteRepository{
		spaceID: spaceID,
	}
}

func (qq *EntSpaceFileWriteRepository) CreateRootDirectory(ctx ctxx.Context, name string) (*FileDTO, error) {
	filex, err := ctx.TenantCtx().TTx.File.Create().
		SetName(name).
		SetIsDirectory(true).
		SetIndexedAt(time.Now()).
		SetModifiedAt(time.Now()).
		SetSpaceID(qq.spaceID).
		SetIsRootDir(true).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return entFileToDTO(filex), nil
}

func (qq *EntSpaceFileWriteRepository) CreateDirectory(
	ctx ctxx.Context,
	parentID int64,
	name string,
) (*FileDTO, error) {
	filex, err := ctx.TenantCtx().TTx.File.Create().
		SetName(name).
		SetIsDirectory(true).
		SetIndexedAt(time.Now()).
		SetModifiedAt(time.Now()).
		SetParentID(parentID).
		SetSpaceID(qq.spaceID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return entFileToDTO(filex), nil
}

func (qq *EntSpaceFileWriteRepository) CreateFile(
	ctx ctxx.Context,
	parentID int64,
	name string,
	isInInbox bool,
) (*FileDTO, error) {
	filex, err := ctx.TenantCtx().TTx.File.Create().
		SetName(name).
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		SetParentID(parentID).
		SetSpaceID(qq.spaceID).
		SetIsInInbox(isInInbox).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return entFileToDTO(filex), nil
}

func (qq *EntSpaceFileWriteRepository) MoveFileByIDX(
	ctx ctxx.Context,
	fileID int64,
	parentID int64,
	nilableName *string,
) (*FileDTO, error) {
	filex, err := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.ID(fileID),
			file.SpaceID(qq.spaceID),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	update := filex.Update().SetParentID(parentID)
	if nilableName != nil {
		update.SetName(*nilableName)
	}

	filex, err = update.Save(ctx)
	if err != nil {
		return nil, err
	}

	return entFileToDTO(filex), nil
}

func (qq *EntSpaceFileWriteRepository) RenameFileByIDX(ctx ctxx.Context, fileID int64, name string) error {
	filex, err := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.ID(fileID),
			file.SpaceID(qq.spaceID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	return filex.Update().SetName(name).Exec(ctx)
}

func (qq *EntSpaceFileWriteRepository) ResetFileOCRStateByIDX(ctx ctxx.Context, fileID int64) error {
	filex, err := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.ID(fileID),
			file.SpaceID(qq.spaceID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	return filex.Update().
		SetOcrContent("").
		ClearOcrSuccessAt().
		SetOcrRetryCount(0).
		SetOcrLastTriedAt(time.Time{}).
		Exec(ctx)
}

func (qq *EntSpaceFileWriteRepository) SetFileInInboxByIDX(
	ctx ctxx.Context,
	fileID int64,
	isInInbox bool,
) error {
	filex, err := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.ID(fileID),
			file.SpaceID(qq.spaceID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	return filex.Update().
		SetIsInInbox(isInInbox).
		Exec(ctx)
}

func (qq *EntSpaceFileWriteRepository) SetFileDocumentTypeByIDX(
	ctx ctxx.Context,
	fileID int64,
	nilableDocumentTypeID *int64,
) error {
	filex, err := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.ID(fileID),
			file.SpaceID(qq.spaceID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	update := filex.Update()
	if nilableDocumentTypeID == nil {
		return update.ClearDocumentTypeID().Exec(ctx)
	}

	return update.SetDocumentTypeID(*nilableDocumentTypeID).Exec(ctx)
}

func (qq *EntSpaceFileWriteRepository) SoftDeleteFileByIDX(
	ctx ctxx.Context,
	fileID int64,
	deletedBy int64,
) error {
	filex, err := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.ID(fileID),
			file.SpaceID(qq.spaceID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	return filex.Update().
		SetDeletedAt(time.Now()).
		SetDeletedBy(deletedBy).
		Exec(ctx)
}

func (qq *EntSpaceFileWriteRepository) SoftDeleteFilesByIDs(
	ctx ctxx.Context,
	fileIDs []int64,
	deletedBy int64,
) error {
	if len(fileIDs) == 0 {
		return nil
	}

	err := ctx.TenantCtx().TTx.File.Update().
		Where(
			file.IDIn(fileIDs...),
			file.SpaceID(qq.spaceID),
		).
		SetDeletedAt(time.Now()).
		SetDeletedBy(deletedBy).
		Exec(ctx)

	return err
}

func (qq *EntSpaceFileWriteRepository) RestoreDeletedFile(
	ctx ctxx.Context,
	filePublicID string,
) (*RestoreFileResultDTO, error) {
	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	filex, err := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.PublicIDEQ(entx.NewCIText(filePublicID)),
			file.SpaceID(qq.spaceID),
		).
		Only(ctxWithDeleted)
	if err != nil {
		return nil, err
	}

	if filex.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Folders cannot be restored.")
	}
	if filex.DeletedAt.IsZero() {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is not deleted.")
	}

	parentExists := false
	if filex.ParentID != 0 {
		parentExists = ctx.TenantCtx().TTx.File.Query().
			Where(
				file.ID(filex.ParentID),
				file.SpaceID(qq.spaceID),
			).
			ExistX(ctx)
	}

	update := filex.Update().
		ClearDeletedAt().
		ClearDeletedBy()

	if !parentExists {
		update = update.
			SetIsInInbox(true).
			SetParentID(ctx.SpaceCtx().SpaceRootDir().ID)
	}

	filex = update.SaveX(ctx)

	return &RestoreFileResultDTO{
		File:         entFileToDTO(filex),
		ParentExists: parentExists,
	}, nil
}

func (qq *EntSpaceFileWriteRepository) DeleteFileWithVersionsByIDX(ctx ctxx.Context, fileID int64) error {
	ctxWithDeleted := schema.SkipSoftDelete(ctx)

	fileExists, err := ctx.TenantCtx().TTx.File.Query().
		Where(file.ID(fileID), file.SpaceID(qq.spaceID)).
		Exist(ctxWithDeleted)
	if err != nil {
		return err
	}
	if !fileExists {
		return nil
	}

	_, err = ctx.TenantCtx().TTx.FileVersion.Delete().
		Where(fileversion.FileID(fileID)).
		Exec(ctxWithDeleted)
	if err != nil {
		return err
	}

	_, err = ctx.TenantCtx().TTx.File.Delete().
		Where(file.ID(fileID), file.SpaceID(qq.spaceID)).
		Exec(ctxWithDeleted)

	return err
}

func (qq *EntSpaceFileWriteRepository) MergeInboxFileVersion(
	ctx ctxx.Context,
	sourceFile *FileDTO,
	targetFile *FileDTO,
) error {
	if sourceFile == nil || targetFile == nil {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Source and target files are required.")
	}

	sourceVersion, err := ctx.TenantCtx().TTx.FileVersion.Query().
		Where(fileversion.FileID(sourceFile.ID)).
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		WithStoredFile().
		First(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusBadRequest, "Source file has no versions.")
		}
		return err
	}

	if sourceVersion.Edges.StoredFile == nil {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Source file has no stored file.")
	}

	latestVersion, err := ctx.TenantCtx().TTx.FileVersion.Query().
		Where(fileversion.FileID(targetFile.ID)).
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		First(ctx)
	if err != nil && !enttenant.IsNotFound(err) {
		return err
	}

	versionNumber := 1
	if err == nil {
		versionNumber = latestVersion.VersionNumber + 1
	}

	ctx.TenantCtx().TTx.FileVersion.Create().
		SetFileID(targetFile.ID).
		SetStoredFileID(sourceVersion.Edges.StoredFile.ID).
		SetVersionNumber(versionNumber).
		SaveX(ctx)

	update := ctx.TenantCtx().TTx.File.UpdateOneID(targetFile.ID).
		SetName(sourceFile.Name).
		SetOcrRetryCount(0).
		SetOcrLastTriedAt(time.Time{})
	if sourceFile.OcrSuccessAt != nil {
		update.SetOcrContent(sourceFile.OcrContent)
		update.SetOcrSuccessAt(*sourceFile.OcrSuccessAt)
	} else {
		update.SetOcrContent("")
		update.ClearOcrSuccessAt()
	}
	if err = update.Exec(ctx); err != nil {
		return err
	}

	return qq.DeleteFileWithVersionsByIDX(ctx, sourceFile.ID)
}
