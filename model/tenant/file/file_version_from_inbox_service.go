package file

import (
	"log"
	"net/http"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/core/util/e"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/schema"
)

type FileVersionFromInboxService struct{}

func NewFileVersionFromInboxService() *FileVersionFromInboxService {
	return &FileVersionFromInboxService{}
}

func (qq *FileVersionFromInboxService) MergeFromInbox(
	ctx ctxx.Context,
	sourceFile *enttenant.File,
	targetFile *enttenant.File,
) (*enttenant.File, error) {
	if sourceFile == nil || targetFile == nil {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source and target files are required.")
	}

	if sourceFile.ID == targetFile.ID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source and target must be different files.")
	}

	if sourceFile.SpaceID != ctx.SpaceCtx().Space.ID || targetFile.SpaceID != ctx.SpaceCtx().Space.ID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File does not belong to current space.")
	}

	if sourceFile.IsDirectory || targetFile.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot merge directories.")
	}

	if !sourceFile.DeletedAt.IsZero() {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file is deleted.")
	}

	sourceVersion, err := sourceFile.QueryFileVersions().
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		WithStoredFile().
		First(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file has no versions.")
		}
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not read source version.")
	}

	if sourceVersion.Edges.StoredFile == nil {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file has no stored file.")
	}

	latestVersion, err := targetFile.QueryFileVersions().
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		First(ctx)
	if err != nil && !enttenant.IsNotFound(err) {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not read target versions.")
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

	update := targetFile.Update().
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
	targetFile, err = update.Save(ctx)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not update target file.")
	}

	if !sourceFile.IsInInbox {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file is not in inbox.")
	}
	_, err = ctx.TenantCtx().TTx.FileVersion.Delete().Where(fileversion.FileID(sourceFile.ID)).Exec(ctx)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not remove source versions.")
	}

	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	err = ctx.TenantCtx().TTx.File.DeleteOneID(sourceFile.ID).Exec(ctxWithDeleted)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not delete source file.")
	}

	return targetFile, nil
}
