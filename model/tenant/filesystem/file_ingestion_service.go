package filesystem

import (
	"context"
	"io"
	"log"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/util/txx"
)

// FileIngestionService coordinates prepared storage without spanning I/O with a transaction.
type FileIngestionService struct {
	fileSystem *S3FileSystem
}

// NewFileIngestionService creates shared browser and protocol ingestion coordination.
func NewFileIngestionService(fileSystem *S3FileSystem) *FileIngestionService {
	return &FileIngestionService{fileSystem: fileSystem}
}

// Ingest prepares hidden storage, streams and verifies bytes, then finalizes under fresh access.
func (qq *FileIngestionService) Ingest(
	ctx *ctxx.SpaceContext,
	reader io.Reader,
	filename string,
	parentDirID int64,
	isInInbox bool,
	source filesource.FileSource,
	expectedBytes *int64,
	mainCheck func(context.Context, *entmain.Tx) error,
) (*FileIngestionResult, error) {
	prepared, err := txx.WithTenantWriteSpaceTx(
		ctx,
		func(writeCtx *ctxx.SpaceContext) (*PreparedUpload, error) {
			return qq.fileSystem.PrepareFileUploadWithSource(
				writeCtx, filename, parentDirID, isInInbox, source,
			)
		},
	)
	if err != nil {
		return nil, err
	}
	return qq.IngestPrepared(ctx, reader, prepared, expectedBytes, mainCheck)
}

// IngestPrepared streams and finalizes an upload intent prepared by a specialized workflow.
func (qq *FileIngestionService) IngestPrepared(
	ctx *ctxx.SpaceContext,
	reader io.Reader,
	prepared *PreparedUpload,
	expectedBytes *int64,
	mainCheck func(context.Context, *entmain.Tx) error,
) (*FileIngestionResult, error) {
	var uploadResult *PreparedUploadResult
	var err error
	if expectedBytes == nil {
		uploadResult, err = qq.fileSystem.UploadPreparedFile(ctx, reader, prepared)
	} else {
		uploadResult, err = qq.fileSystem.UploadPreparedFileWithExpectedSize(
			ctx, reader, prepared, *expectedBytes,
		)
	}
	if err != nil {
		qq.fileSystem.HandlePreparedUploadFailure(ctx, prepared, err, true)
		return nil, err
	}

	_, err = txx.WithFreshAuthorizedTenantWriteSpaceTxAndMainCheck(
		ctx,
		mainCheck,
		func(writeCtx *ctxx.SpaceContext) (*struct{}, error) {
			return nil, qq.fileSystem.FinalizePreparedUploadWithoutMime(
				writeCtx, prepared, uploadResult,
			)
		},
	)
	if err != nil {
		// A failed commit can be ambiguous. Leave verified temporary bytes and the
		// unfinished row for reconciliation instead of risking canonical-byte loss.
		log.Println(err)
		return nil, err
	}
	if _, err := qq.fileSystem.UpdateMimeTypeAfterFinalization(
		ctx, true, prepared.StoredFileID,
	); err != nil {
		log.Println(err)
	}

	return &FileIngestionResult{
		FilePublicID: prepared.FilePublicID,
		Filename:     prepared.OriginalFilename,
		Size:         uploadResult.FileSize,
		IsInInbox:    prepared.IsInInbox,
	}, nil
}
