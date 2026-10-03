package file

import (
	"log"
	"net/http"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	dbfile "github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/db/enttenant/webdavresource"
	"github.com/simpledms/simpledms/util/e"
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
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File does not belong to the current Space.")
	}

	if sourceFile.IsDirectory || targetFile.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot merge folders.")
	}

	if !sourceFile.DeletedAt.IsZero() {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file is deleted.")
	}
	if !sourceFile.IsInInbox {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file is not in the Inbox.")
	}
	if !targetFile.DeletedAt.IsZero() {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File not found.")
	}

	filename, err := qq.mergedFilename(ctx, sourceFile, targetFile)
	if err != nil {
		return nil, err
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

	_, err = ctx.TenantCtx().TTx.FileVersion.Create().
		SetFileID(targetFile.ID).
		SetStoredFileID(sourceVersion.Edges.StoredFile.ID).
		SetVersionNumber(versionNumber).
		Save(ctx)
	if err != nil {
		log.Printf("add merged file version: %v", err)
		return nil, err
	}

	update := targetFile.Update().
		SetName(filename).
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

	if _, err := NewDocumentNotes().Transfer(ctx, sourceFile, targetFile); err != nil {
		return nil, err
	}
	if _, err := ctx.TenantCtx().TTx.TagAssignment.Delete().
		Where(tagassignment.FileID(sourceFile.ID)).Exec(ctx); err != nil {
		log.Printf("remove merged source tags: %v", err)
		return nil, err
	}
	if _, err := ctx.TenantCtx().TTx.FilePropertyAssignment.Delete().
		Where(filepropertyassignment.FileID(sourceFile.ID)).Exec(ctx); err != nil {
		log.Printf("remove merged source fields: %v", err)
		return nil, err
	}
	if _, err := ctx.TenantCtx().TTx.WebDAVResource.Delete().
		Where(enttenantwebdavresource.FileID(sourceFile.ID)).Exec(ctx); err != nil {
		log.Printf("remove merged source upload aliases: %v", err)
		return nil, err
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

func (qq *FileVersionFromInboxService) mergedFilename(
	ctx ctxx.Context, source, target *enttenant.File,
) (string, error) {
	if target.IsInInbox || target.Name == source.Name {
		return source.Name, nil
	}
	conflict, err := ctx.SpaceCtx().Space.QueryFiles().Where(
		dbfile.ParentID(target.ParentID),
		dbfile.Name(source.Name),
		dbfile.IsInInbox(false),
		dbfile.IDNEQ(target.ID),
	).Exist(ctx)
	if err != nil {
		log.Printf("check merged filename: %v", err)
		return "", err
	}
	if conflict {
		return target.Name, nil
	}
	return source.Name, nil
}
