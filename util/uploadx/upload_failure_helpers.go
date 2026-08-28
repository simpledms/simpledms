package uploadx

import (
	"log"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	entmainschema "github.com/simpledms/simpledms/db/entmain/schema"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/util/txx"
)

func MarkStoredFileUploadFailed(ctx *ctxx.SpaceContext, storedFileID int64) {
	_, err := txx.WithTenantWriteSpaceTx(ctx, func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
		ctxWithIncomplete := tenantprivacy.DecisionContext(
			enttenantschema.WithUnfinishedUploads(writeCtx),
			tenantprivacy.Allow,
		)
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
