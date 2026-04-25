package uploadx

import (
	"log"
	"time"

	"github.com/simpledms/simpledms/core/util/txx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	entmainschema "github.com/simpledms/simpledms/db/entmain/schema"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
)

func MarkStoredFileUploadFailed(ctx *ctxx.SpaceContext, storedFileID int64) {
	_, err := txx.WithTenantWriteSpaceTx(ctx, func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
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

func DeleteFailedUploadFile(ctx *ctxx.SpaceContext, fileID int64) {
	if fileID == 0 {
		return
	}
	_, err := txx.WithTenantWriteSpaceTx(ctx, func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
		ctxWithDeleted := enttenantschema.SkipSoftDelete(writeCtx)
		_, err := writeCtx.TTx.FileVersion.
			Delete().
			Where(fileversion.FileID(fileID)).
			Exec(ctxWithDeleted)
		if err != nil {
			return nil, err
		}
		_, err = writeCtx.TTx.File.
			Delete().
			Where(file.ID(fileID)).
			Exec(ctxWithDeleted)
		return nil, err
	})
	if err != nil {
		log.Println(err)
	}
}

func MarkTemporaryFileUploadFailed(ctx ctxx.Context, temporaryFileID int64) {
	_, err := txx.WithMainWriteTx(ctx, func(writeTx *entmain.Tx) (*struct{}, error) {
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
	DeleteFailedUploadFile(ctx, prepared.FileID)
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
