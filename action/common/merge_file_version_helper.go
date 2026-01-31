package common

import (
	"log"
	"net/http"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/util/e"
)

type MergeFileVersionHelper struct {
}

func NewMergeFileVersionHelper() *MergeFileVersionHelper {
	return &MergeFileVersionHelper{}
}

func (qq *MergeFileVersionHelper) SuggestInboxSources(
	ctx ctxx.Context,
	targetFile *enttenant.File,
	searchQuery string,
	limit int,
) []*enttenant.File {
	if targetFile == nil {
		return nil
	}

	query := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.SpaceID(ctx.SpaceCtx().Space.ID),
			file.IsInInbox(true),
			file.IsDirectory(false),
			file.DeletedAtIsNil(),
		)

	if searchQuery != "" {
		query = query.Where(file.NameContains(searchQuery))
	}

	query = query.Order(file.ByName(sql.OrderAsc()))
	if limit > 0 {
		query = query.Limit(limit)
	}

	return query.AllX(ctx)
}

// TODO logic should be moved to model
func (qq *MergeFileVersionHelper) Merge(
	ctx ctxx.Context,
	sourceFileID int64,
	targetFileID int64,
) (*enttenant.File, error) {
	if sourceFileID == targetFileID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source and target must be different files.")
	}

	fileToMerge, err := ctx.TenantCtx().TTx.File.Get(ctx, sourceFileID)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusNotFound, "Source file not found.")
	}

	targetFile, err := ctx.TenantCtx().TTx.File.Get(ctx, targetFileID)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusNotFound, "Target file not found.")
	}

	if fileToMerge.SpaceID != ctx.SpaceCtx().Space.ID || targetFile.SpaceID != ctx.SpaceCtx().Space.ID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File does not belong to current space.")
	}

	if fileToMerge.IsDirectory || targetFile.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot merge directories.")
	}

	if !fileToMerge.DeletedAt.IsZero() {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file is deleted.")
	}

	sourceVersion, err := fileToMerge.QueryFileVersions().
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

	// TODO should be from model
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

	targetFile, err = targetFile.Update().
		SetName(fileToMerge.Name).
		SetOcrContent("").
		ClearOcrSuccessAt().
		SetOcrRetryCount(0).
		SetOcrLastTriedAt(time.Time{}).
		Save(ctx)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not update target file.")
	}

	// hard delete the file in the inbox
	if !fileToMerge.IsInInbox {
		// safety check, we never want to chance to delete a file that is not in the inbox
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file is not in inbox.")
	}
	_, err = ctx.TenantCtx().TTx.FileVersion.Delete().Where(fileversion.FileID(fileToMerge.ID)).Exec(ctx)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not remove source versions.")
	}
	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	err = ctx.TenantCtx().TTx.File.DeleteOneID(fileToMerge.ID).Exec(ctxWithDeleted)
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not delete source file.")
	}

	return targetFile, nil
}
