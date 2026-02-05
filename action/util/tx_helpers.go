package util

import (
	"log"
	"net/http"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	entmainschema "github.com/simpledms/simpledms/db/entmain/schema"
	"github.com/simpledms/simpledms/db/enttenant/file"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/model/filesystem"
	"github.com/simpledms/simpledms/util/e"
)

func WithTenantWriteSpaceTx[T any](ctx *ctxx.SpaceContext, fn func(*ctxx.SpaceContext) (T, error)) (T, error) {
	var zero T
	if ctx != nil && !ctx.TenantCtx().IsReadOnlyTx() {
		return fn(ctx)
	}

	tenantDB, ok := ctx.UnsafeTenantDB()
	if !ok {
		log.Println("tenant db not found", ctx.TenantCtx().Tenant.ID)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Tenant database not found.")
	}

	writeTx, err := tenantDB.Tx(ctx, false)
	if err != nil {
		log.Println(err)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not start transaction.")
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if err := writeTx.Rollback(); err != nil {
			log.Println(err)
		}
	}()

	writeTenantCtx := ctxx.NewTenantContext(ctx.MainCtx(), writeTx, ctx.TenantCtx().Tenant, false)
	writeSpace := writeTx.Space.GetX(writeTenantCtx, ctx.SpaceCtx().Space.ID)
	writeSpaceCtx := ctxx.NewSpaceContext(writeTenantCtx, writeSpace)

	result, err := fn(writeSpaceCtx)
	if err != nil {
		return zero, err
	}

	if err := writeTx.Commit(); err != nil {
		log.Println(err)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not save file.")
	}
	committed = true

	return result, nil
}

func EnsureFileDoesNotExist(ctx ctxx.Context, filename string, parentDirID int64, isInInbox bool) error {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil
	}

	fileExists := ctx.SpaceCtx().Space.QueryFiles().
		Where(file.Name(filename), file.ParentID(parentDirID), file.IsInInbox(isInInbox)).
		ExistX(ctx)
	if fileExists {
		return e.NewHTTPErrorf(http.StatusBadRequest, "File already exists.")
	}

	return nil
}

func WithMainWriteTx[T any](ctx ctxx.Context, fn func(*entmain.Tx) (T, error)) (T, error) {
	var zero T
	if ctx != nil && ctx.MainCtx() != nil && !ctx.MainCtx().IsReadOnlyTx() {
		return fn(ctx.MainCtx().MainTx)
	}

	mainDB := ctx.MainCtx().UnsafeMainDB()
	if mainDB == nil {
		log.Println("main db not found")
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not start transaction.")
	}

	writeTx, err := mainDB.Tx(ctx, false)
	if err != nil {
		log.Println(err)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not start transaction.")
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if err := writeTx.Rollback(); err != nil {
			log.Println(err)
		}
	}()

	result, err := fn(writeTx)
	if err != nil {
		return zero, err
	}

	if err := writeTx.Commit(); err != nil {
		log.Println(err)
		return zero, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not save file.")
	}
	committed = true

	return result, nil
}

func MarkStoredFileUploadFailed(ctx *ctxx.SpaceContext, storedFileID int64) {
	if storedFileID == 0 {
		return
	}

	_, err := WithTenantWriteSpaceTx(ctx, func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
		ctxWithIncomplete := enttenantschema.WithUnfinishedUploads(writeCtx)
		err := writeCtx.TTx.StoredFile.
			UpdateOneID(storedFileID).
			SetUploadFailedAt(time.Now()).
			Exec(ctxWithIncomplete)
		return nil, err
	})
	if err != nil {
		log.Println(err)
	}
}

func MarkTemporaryFileUploadFailed(ctx ctxx.Context, temporaryFileID int64) {
	if temporaryFileID == 0 {
		return
	}

	_, err := WithMainWriteTx(ctx, func(writeTx *entmain.Tx) (*struct{}, error) {
		ctxWithIncomplete := entmainschema.WithUnfinishedUploads(ctx)
		err := writeTx.TemporaryFile.
			UpdateOneID(temporaryFileID).
			SetUploadFailedAt(time.Now()).
			Exec(ctxWithIncomplete)
		return nil, err
	})
	if err != nil {
		log.Println(err)
	}
}

func HandleStoredFileUploadFailure(
	ctx *ctxx.SpaceContext,
	fs *filesystem.S3FileSystem,
	prepared *filesystem.PreparedUpload,
	cause error,
	cleanup bool,
) {
	if cause != nil {
		log.Println(cause)
	}
	if prepared == nil {
		return
	}
	if cleanup {
		if err := fs.RemoveTemporaryObject(ctx, prepared.TemporaryStoragePath, prepared.TemporaryStorageFilename); err != nil {
			log.Println(err)
		}
	}
	MarkStoredFileUploadFailed(ctx, prepared.StoredFileID)
}

func HandleTemporaryFileUploadFailure(
	ctx ctxx.Context,
	fs *filesystem.S3FileSystem,
	prepared *filesystem.PreparedAccountUpload,
	cause error,
	cleanup bool,
) {
	if cause != nil {
		log.Println(cause)
	}
	if prepared == nil {
		return
	}
	if cleanup {
		if err := fs.RemoveTemporaryObject(ctx, prepared.StoragePath, prepared.StorageFilename); err != nil {
			log.Println(err)
		}
	}
	MarkTemporaryFileUploadFailed(ctx, prepared.TemporaryFileID)
}
