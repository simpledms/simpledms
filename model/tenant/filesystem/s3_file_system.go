package filesystem

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"filippo.io/age"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/minio/minio-go/v7"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	entmainschema "github.com/simpledms/simpledms/db/entmain/schema"
	"github.com/simpledms/simpledms/db/entmain/systemconfig"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/encryptor"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/pathx"
	"github.com/simpledms/simpledms/util"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
	"github.com/simpledms/simpledms/util/fileutil"
	"github.com/simpledms/simpledms/util/recoverx"
)

type S3FileSystem struct {
	*FileSystem
	client                *minio.Client
	bucketName            string
	disableFileEncryption bool
	storageQuota          *StorageQuota
}

const bytesPerMiB int64 = 1024 * 1024

const uploadProgressInterval = 30 * time.Second

const accountConversionTakeoverAfter = time.Hour

const temporaryAccountFileExpiry = 15 * time.Minute

const (
	emptyUploadMessage            = "Upload is empty."
	couldNotSaveFileMessage       = "Could not save file."
	tenantDatabaseNotFoundMessage = "Tenant database not found."
)

var errUploadTooLarge = errors.New("upload is too large")

func NewS3FileSystem(
	client *minio.Client,
	bucketName string,
	fileSystem *FileSystem,
	disableFileEncryption bool,
	storageQuota *StorageQuota,
) *S3FileSystem {
	if storageQuota == nil {
		storageQuota = NewStorageQuota(false)
	}

	return &S3FileSystem{
		FileSystem:            fileSystem,
		client:                client,
		bucketName:            bucketName,
		disableFileEncryption: disableFileEncryption,
		storageQuota:          storageQuota,
	}
}

func (qq *S3FileSystem) TenantUsageBytes(ctx ctxx.Context) (int64, int64, error) {
	return qq.storageQuota.TenantUsageBytes(ctx)
}

func (qq *S3FileSystem) StorageQuota() *StorageQuota {
	return qq.storageQuota
}

// caller has to close io.ReadCloser
// TODO OpenFile or CopyFile?
func (qq *S3FileSystem) OpenFile(ctx ctxx.Context, file *storedfilemodel.StoredFile) (io.ReadCloser, error) {
	objectName, err := file.ObjectNameWithPrefix()
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not get object name.")
	}

	x25519Identity, err := qq.x25519Identity(ctx, objectName)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return qq.UnsafeOpenFile(ctx, x25519Identity, file)
}

// caller has to close io.ReadCloser;
//
// Unsafe because it should never be used directly, but is done in Scheduler because otherwise
// ctxx.TenantContext needs to be constructed
func (qq *S3FileSystem) UnsafeOpenFile(ctx context.Context, x25519Identity *age.X25519Identity, file *storedfilemodel.StoredFile) (io.ReadCloser, error) {
	objectName, err := file.ObjectNameWithPrefix()
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not get object name.")
	}

	obj, err := qq.client.GetObject(ctx, qq.bucketName, objectName, minio.GetObjectOptions{
		ServerSideEncryption: nil,
		VersionID:            "",
		PartNumber:           0,
		Checksum:             false,
		Internal:             minio.AdvancedGetOptions{},
	})
	if err != nil {
		// if the file doesn't exist an error might not be returned, just the client starts reading the file...
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not open file.")
	}

	// caller has to close pipeReader
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		defer recoverx.Recover("openFile")
		var err error

		// FIXME when to close obj?
		defer func() {
			err = obj.Close()
			if err != nil {
				log.Println(err)
			}
		}()

		var gzipReaderInput io.Reader
		gzipReaderInput = obj

		if !qq.disableFileEncryption {
			decryptor, err := age.Decrypt(obj, x25519Identity)
			if err != nil {
				// if the file doesn't exist, an error like the following is returned, whereby `the specified key does not exist.` comes
				// from the minio client and is just wrapped:
				// `failed to read header: parsing age header: failed to read intro: The specified key does not exist.`
				// minio errors cannot be checked easily because they are just strings, like `minio.NoSuchKey` and
				// minio.ToErrorResponse(err) probably doesn't work with wrapped errors too...

				log.Println(err)

				erry := pipeWriter.CloseWithError(err)
				if erry != nil {
					log.Println(erry)
				}

				return
			}

			gzipReaderInput = decryptor
		}

		gzipReader, err := gzip.NewReader(gzipReaderInput)
		if err != nil {
			log.Println(err)

			erry := pipeWriter.CloseWithError(err)
			if erry != nil {
				log.Println(erry)
			}
			return
		}
		defer func() {
			// FIXME is order important as for write?
			err = gzipReader.Close()
			if err != nil {
				log.Println(err)
			}
		}()

		if _, err := io.Copy(pipeWriter, gzipReader); err != nil {
			erry := pipeWriter.CloseWithError(err)
			if erry != nil {
				log.Println(erry)
			}
			return
		}

		err = pipeWriter.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	return pipeReader, nil
}

// used for restore of backup
func (qq *S3FileSystem) UnsafeUploadBlobToStorageLocation(
	ctx context.Context,
	x25519Identity *age.X25519Identity,
	fileToSave io.Reader,
	originalFilename string,
	storagePath string,
	storageFilename string,
) error {
	_, _, _, err := qq.saveFile(
		ctx,
		x25519Identity,
		fileToSave,
		originalFilename,
		storagePath,
		storageFilename,
		nil,
	)

	return err
}

func (qq *S3FileSystem) PrepareFileUpload(
	ctx ctxx.Context,
	originalFilename string,
	parentDirFileID int64,
	isInInbox bool,
) (*PreparedUpload, error) {
	return qq.PrepareFileUploadWithSource(
		ctx,
		originalFilename,
		parentDirFileID,
		isInInbox,
		filesource.WebInterface,
	)
}

func (qq *S3FileSystem) PrepareFileUploadWithSource(
	ctx ctxx.Context,
	originalFilename string,
	parentDirFileID int64,
	isInInbox bool,
	source filesource.FileSource,
) (*PreparedUpload, error) {
	return qq.prepareFileUploadWithSource(ctx, originalFilename, parentDirFileID, isInInbox, source, true)
}

func (qq *S3FileSystem) PrepareFileUploadIntentWithSource(
	ctx ctxx.Context,
	originalFilename string,
	parentDirFileID int64,
	isInInbox bool,
	source filesource.FileSource,
) (*PreparedUpload, error) {
	return qq.prepareFileUploadWithSource(
		ctx,
		originalFilename,
		parentDirFileID,
		isInInbox,
		source,
		false,
	)
}

func (qq *S3FileSystem) prepareFileUploadWithSource(
	ctx ctxx.Context,
	originalFilename string,
	parentDirFileID int64,
	isInInbox bool,
	source filesource.FileSource,
	checkExisting bool,
) (*PreparedUpload, error) {
	meta, err := qq.prepareUploadMetadata(ctx, originalFilename)
	if err != nil {
		return nil, err
	}
	if checkExisting {
		err := qq.ensureFileDoesNotExistInFolderMode(ctx, meta.originalFilename, parentDirFileID, isInInbox)
		if err != nil {
			return nil, err
		}
	}
	prepared, err := qq.createStoredFileForPreparedUpload(ctx, meta)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	prepared.IsNewFile = true
	prepared.ParentDirFileID = parentDirFileID
	prepared.IsInInbox = isInInbox
	prepared.Source = source

	return prepared, nil
}

func (qq *S3FileSystem) PrepareFileVersionUpload(
	ctx ctxx.Context,
	originalFilename string,
	fileID int64,
) (*PreparedUpload, error) {
	meta, err := qq.prepareUploadMetadata(ctx, originalFilename)
	if err != nil {
		return nil, err
	}

	filex := ctx.TenantCtx().TTx.File.GetX(ctx, fileID)
	if filex.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot upload versions for directories.")
	}
	if err := qq.ensureFileDoesNotExistInFolderMode(ctx, meta.originalFilename, filex.ParentID, filex.IsInInbox); err != nil {
		return nil, err
	}

	prepared, err := qq.createStoredFileForPreparedUpload(ctx, meta)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	prepared.FileID = filex.ID
	prepared.ParentDirFileID = filex.ParentID
	prepared.IsInInbox = filex.IsInInbox

	return prepared, nil
}

func (qq *S3FileSystem) prepareUploadMetadata(ctx ctxx.Context, originalFilename string) (*uploadMetadata, error) {
	originalFilename = filepath.Clean(originalFilename)
	if !filenamex.IsAllowed(originalFilename) {
		log.Println("invalid filename")
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid filename.")
	}

	tmpStoragePrefix := pathx.S3TemporaryStoragePrefix(ctx.TenantCtx().TenantID)
	if tmpStoragePrefix == "" {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Storage path is empty.")
	}

	storageFilenameWithoutExt := util.NewPublicID()
	storageFilename := qq.storageFilename(originalFilename, storageFilenameWithoutExt)

	return &uploadMetadata{
		originalFilename:          originalFilename,
		temporaryStoragePath:      tmpStoragePrefix,
		storageFilenameWithoutExt: storageFilenameWithoutExt,
		storageFilename:           storageFilename,
	}, nil
}

func (qq *S3FileSystem) createStoredFileForPreparedUpload(
	ctx ctxx.Context,
	meta *uploadMetadata,
) (*PreparedUpload, error) {
	finalStoragePrefix := pathx.S3StoragePrefix(ctx.TenantCtx().TenantID)
	now := time.Now()
	storedFilex := ctx.TenantCtx().TTx.StoredFile.Create().
		SetFilename(meta.originalFilename).
		SetSizeInStorage(0).
		SetStorageType(storagetype.S3).
		SetBucketName(qq.bucketName).
		SetStoragePath(finalStoragePrefix).
		SetStorageFilename(meta.storageFilename).
		// temporary because it gets moved by scheduler to prevent orphan files in object storage
		// if transaction fails
		SetTemporaryStoragePath(meta.temporaryStoragePath).
		SetTemporaryStorageFilename(meta.storageFilename).
		SetUploadStartedAt(now).
		SetUploadLastProgressAt(now).
		SaveX(ctx)

	prepared := &PreparedUpload{
		StoredFileID:              storedFilex.ID,
		OriginalFilename:          meta.originalFilename,
		StorageFilenameWithoutExt: meta.storageFilenameWithoutExt,
		StorageFilename:           meta.storageFilename,
		TemporaryStoragePath:      meta.temporaryStoragePath,
		TemporaryStorageFilename:  meta.storageFilename,
	}

	return prepared, nil
}

func (qq *S3FileSystem) ensureFileDoesNotExistInFolderMode(
	ctx ctxx.Context,
	filename string,
	parentDirFileID int64,
	isInInbox bool,
) error {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil
	}

	fileExists := ctx.SpaceCtx().Space.QueryFiles().
		Where(file.Name(filename), file.ParentID(parentDirFileID), file.IsInInbox(isInInbox)).
		ExistX(ctx)
	if fileExists {
		return e.NewHTTPErrorf(http.StatusBadRequest, "File already exists.")
	}

	return nil
}

// caller has to close fileToSave
func (qq *S3FileSystem) UploadPreparedFileWithExpectedSize(
	ctx ctxx.Context,
	fileToSave io.Reader,
	prepared *PreparedUpload,
	expectedUploadedBytes int64,
) (*PreparedUploadResult, error) {
	if expectedUploadedBytes <= 0 {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, emptyUploadMessage)
	}

	nilableUploadLimitBytes, err := qq.NilableEffectiveUploadSizeLimitBytes(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if nilableUploadLimitBytes != nil && expectedUploadedBytes > *nilableUploadLimitBytes {
		return nil, qq.uploadTooLargeError(*nilableUploadLimitBytes)
	}

	limitedFileToSave := fileToSave
	if nilableUploadLimitBytes != nil {
		limitedFileToSave = &maxBytesReader{r: fileToSave, max: *nilableUploadLimitBytes}
	}
	mimeReader := &mimeSniffingReader{r: limitedFileToSave}

	if expectedUploadedBytes > 0 && ctx.IsTenantCtx() {
		err := qq.EnsureTenantStorageLimit(ctx, expectedUploadedBytes)
		if err != nil {
			return nil, err
		}
	}

	fileInfo, fileSize, contentSHA256, storageCRC32C, err := qq.uploadPreparedFileWithParams(
		ctx,
		mimeReader,
		prepared.OriginalFilename,
		prepared.StorageFilenameWithoutExt,
		prepared.TemporaryStoragePath,
		prepared.StorageFilename,
		qq.storedFileUploadHeartbeat(ctx, prepared.StoredFileID),
	)
	if err != nil {
		if errors.Is(err, errUploadTooLarge) && nilableUploadLimitBytes != nil {
			return nil, qq.uploadTooLargeError(*nilableUploadLimitBytes)
		}
		log.Println(err)
		return nil, err
	}
	if fileSize != expectedUploadedBytes {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Upload size mismatch.")
	}
	if fileSize == 0 {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, emptyUploadMessage)
	}

	if nilableUploadLimitBytes != nil && fileSize > *nilableUploadLimitBytes {
		return nil, qq.uploadTooLargeError(*nilableUploadLimitBytes)
	}

	return &PreparedUploadResult{
		FileInfo:      fileInfo,
		FileSize:      fileSize,
		ContentSHA256: contentSHA256,
		StorageCRC32C: storageCRC32C,
		MimeType:      mimeReader.MimeType(),
	}, nil
}

func (qq *S3FileSystem) UploadPreparedFile(
	ctx ctxx.Context,
	fileToSave io.Reader,
	prepared *PreparedUpload,
) (*PreparedUploadResult, error) {
	nilableUploadLimitBytes, err := qq.NilableEffectiveUploadSizeLimitBytes(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	limitedFileToSave := fileToSave
	if nilableUploadLimitBytes != nil {
		limitedFileToSave = &maxBytesReader{r: fileToSave, max: *nilableUploadLimitBytes}
	}
	mimeReader := &mimeSniffingReader{r: limitedFileToSave}

	fileInfo, fileSize, contentSHA256, storageCRC32C, err := qq.uploadPreparedFileWithParams(
		ctx,
		mimeReader,
		prepared.OriginalFilename,
		prepared.StorageFilenameWithoutExt,
		prepared.TemporaryStoragePath,
		prepared.StorageFilename,
		qq.storedFileUploadHeartbeat(ctx, prepared.StoredFileID),
	)
	if err != nil {
		if errors.Is(err, errUploadTooLarge) && nilableUploadLimitBytes != nil {
			return nil, qq.uploadTooLargeError(*nilableUploadLimitBytes)
		}
		log.Println(err)
		return nil, err
	}
	if fileSize == 0 {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, emptyUploadMessage)
	}

	return &PreparedUploadResult{
		FileInfo:      fileInfo,
		FileSize:      fileSize,
		ContentSHA256: contentSHA256,
		StorageCRC32C: storageCRC32C,
		MimeType:      mimeReader.MimeType(),
	}, nil
}

func (qq *S3FileSystem) FinalizePreparedUpload(
	ctx ctxx.Context,
	prepared *PreparedUpload,
	result *PreparedUploadResult,
) error {
	return qq.finalizePreparedUploadAs(ctx, prepared, result, prepared.OriginalFilename, true)
}

func (qq *S3FileSystem) FinalizePreparedUploadAsWithoutMime(
	ctx ctxx.Context,
	prepared *PreparedUpload,
	result *PreparedUploadResult,
	filename string,
) error {
	return qq.finalizePreparedUploadAs(ctx, prepared, result, filename, false)
}

func (qq *S3FileSystem) FinalizePreparedUploadWithoutMime(
	ctx ctxx.Context,
	prepared *PreparedUpload,
	result *PreparedUploadResult,
) error {
	return qq.finalizePreparedUploadAs(ctx, prepared, result, prepared.OriginalFilename, false)
}

func (qq *S3FileSystem) finalizePreparedUploadAs(
	ctx ctxx.Context,
	prepared *PreparedUpload,
	result *PreparedUploadResult,
	filename string,
	updateMimeType bool,
) error {
	err := qq.EnsureTenantStorageLimit(ctx, result.FileSize)
	if err != nil {
		return err
	}

	ctxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(ctx),
		tenantprivacy.Allow,
	)
	storedFilex, err := ctx.TenantCtx().TTx.StoredFile.Query().
		Where(
			storedfile.ID(prepared.StoredFileID),
			storedfile.UploadFailedAtIsNil(),
			storedfile.UploadSucceededAtIsNil(),
		).
		Only(ctxWithIncomplete)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusConflict, "Upload is no longer active.")
		}
		return err
	}
	storedFilex = storedFilex.Update().
		SetSize(result.FileSize).
		SetSizeInStorage(result.FileInfo.Size).
		SetSha256(result.FileInfo.ChecksumSHA256).
		SetContentSha256(result.ContentSHA256).
		SetStorageCrc32c(result.StorageCRC32C).
		SetNillableMimeType(nilableNonEmptyString(result.MimeType)).
		SetUploadSucceededAt(time.Now()).
		SaveX(ctxWithIncomplete)

	var filex *enttenant.File
	if prepared.FileID == 0 {
		filex = ctx.TenantCtx().TTx.File.Create().
			SetName(filename).
			SetSource(prepared.Source).
			SetIsDirectory(false).
			SetIndexedAt(time.Now()).
			SetParentID(prepared.ParentDirFileID).
			SetSpaceID(ctx.SpaceCtx().Space.ID).
			SetIsInInbox(prepared.IsInInbox).
			SaveX(ctx)
		prepared.FileID = filex.ID
	} else {
		filex = ctx.TenantCtx().TTx.File.GetX(ctx, prepared.FileID)
		if filex.IsDirectory {
			return e.NewHTTPErrorf(http.StatusBadRequest, "Cannot upload versions for directories.")
		}
		if filex.Name != filename {
			filex = filex.Update().SetName(filename).SaveX(ctx)
		}
	}
	prepared.FilePublicID = filex.PublicID.String()

	if err := qq.addFileVersion(ctx, filex, storedFilex); err != nil {
		log.Println(err)
		return err
	}
	if updateMimeType {
		qq.updateMimeTypeAfterCommit(ctx, prepared.StoredFileID)
	}

	return nil
}

func (qq *S3FileSystem) updateMimeTypeAfterCommit(ctx ctxx.Context, storedFileID int64) {
	ctx.TenantCtx().TTx.OnCommit(func(next enttenant.Committer) enttenant.Committer {
		return enttenant.CommitFunc(func(commitCtx context.Context, tx *enttenant.Tx) error {
			if err := next.Commit(commitCtx, tx); err != nil {
				return err
			}
			if _, err := qq.UpdateMimeTypeAfterFinalization(ctx, true, storedFileID); err != nil {
				log.Println(err)
			}
			return nil
		})
	})
}

func (qq *S3FileSystem) RemoveTemporaryObject(ctx context.Context, storagePath string, storageFilename string) error {
	if storagePath == "" || storageFilename == "" {
		return nil
	}

	if qq.bucketName == "" {
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Bucket name is empty.")
	}

	objectName, err := securejoin.SecureJoin(storagePath, storageFilename)
	if err != nil {
		log.Println(err)
		return err
	}

	err = qq.client.RemoveObject(ctx, qq.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		minioErr := minio.ToErrorResponse(err)
		if minioErr.Code == "NoSuchKey" {
			return nil
		}
		log.Println(err)
		return err
	}

	return nil
}

func (qq *S3FileSystem) SaveDerivedPDF(
	ctx context.Context,
	tenantPublicID string,
	tenantX25519Identity *age.X25519Identity,
	pdf io.Reader,
	filename string,
) (*minio.UploadInfo, string, int64, string, string, error) {
	storageFilenameWithoutExt := util.NewPublicID()
	temporaryStoragePath := pathx.S3TemporaryStoragePrefix(tenantPublicID)
	if temporaryStoragePath == "" {
		return nil, "", 0, "", "", e.NewHTTPErrorf(http.StatusInternalServerError, "Storage path is empty.")
	}

	fileInfo, storageFilename, result, err := qq.saveFile(
		ctx,
		tenantX25519Identity,
		pdf,
		filename,
		storageFilenameWithoutExt,
		temporaryStoragePath,
		nil,
	)
	if err != nil {
		storageFilename = qq.storageFilename(filename, storageFilenameWithoutExt)
		if cleanupErr := qq.RemoveTemporaryObject(ctx, temporaryStoragePath, storageFilename); cleanupErr != nil {
			log.Println(cleanupErr)
		}
	}
	return fileInfo, storageFilename, result.fileSize, result.contentSHA256, result.storageCRC32C, err
}

func (qq *S3FileSystem) RemoveTenantStoredFileObjects(
	ctx context.Context,
	filex *enttenant.StoredFile,
) error {
	filem := storedfilemodel.NewStoredFile(filex)
	objectNames := make([]string, 0, 2)
	finalObjectName, err := filem.UnsafeFinalObjectNameWithPrefix()
	if err != nil {
		return err
	}
	objectNames = append(objectNames, finalObjectName)
	temporaryObjectName, err := filem.UnsafeTempObjectNameWithPrefix()
	if err != nil {
		return err
	}
	if temporaryObjectName != finalObjectName {
		objectNames = append(objectNames, temporaryObjectName)
	}

	for _, objectName := range objectNames {
		err = qq.client.RemoveObject(ctx, qq.bucketName, objectName, minio.RemoveObjectOptions{})
		if err == nil {
			continue
		}
		minioErr := minio.ToErrorResponse(err)
		if minioErr.Code == "NoSuchKey" {
			continue
		}
		return err
	}
	return nil
}

func (qq *S3FileSystem) storedFileUploadHeartbeat(ctx ctxx.Context, storedFileID int64) func(time.Time) {
	if !ctx.IsTenantCtx() || storedFileID == 0 {
		return nil
	}
	tenantDB, ok := ctx.TenantCtx().UnsafeTenantDB()
	if !ok {
		return nil
	}
	return func(now time.Time) {
		go func() {
			heartbeatCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			ctxWithIncomplete := tenantprivacy.DecisionContext(
				enttenantschema.WithUnfinishedUploads(heartbeatCtx),
				tenantprivacy.Allow,
			)
			err := tenantDB.ReadWriteConn.StoredFile.UpdateOneID(storedFileID).
				SetUploadLastProgressAt(now).
				Exec(ctxWithIncomplete)
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				log.Println(err)
			}
		}()
	}
}

func (qq *S3FileSystem) temporaryFileUploadHeartbeat(ctx ctxx.Context, temporaryFileID int64) func(time.Time) {
	if temporaryFileID == 0 {
		return nil
	}
	return func(now time.Time) {
		go func() {
			heartbeatCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			ctxWithIncomplete := entmainschema.WithUnfinishedUploads(heartbeatCtx)
			err := ctx.MainCtx().UnsafeMainDB().ReadWriteConn.TemporaryFile.UpdateOneID(temporaryFileID).
				SetUploadLastProgressAt(now).
				Exec(ctxWithIncomplete)
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				log.Println(err)
			}
		}()
	}
}

func (qq *S3FileSystem) verifyObject(
	ctx context.Context,
	objectName string,
	expectedSize int64,
	expectedSHA256 string,
	expectedCRC32C string,
) error {
	info, err := qq.client.StatObject(ctx, qq.bucketName, objectName, minio.StatObjectOptions{
		Checksum: true,
	})
	if err != nil {
		return err
	}
	if info.Size != expectedSize {
		return fmt.Errorf("stored object size mismatch: got %d, want %d", info.Size, expectedSize)
	}
	if expectedSHA256 == "" && expectedCRC32C == "" {
		return nil
	}
	if expectedCRC32C != "" && info.ChecksumMode == minio.ChecksumFullObjectMode.String() &&
		info.ChecksumCRC32C == expectedCRC32C {
		return nil
	}
	if expectedSHA256 == "" {
		return errors.New("stored object sha256 is missing")
	}

	// S3-compatible backends differ in their checksum metadata, so verify the stored bytes.
	object, err := qq.client.GetObject(ctx, qq.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	hasher := sha256.New()
	size, readErr := io.Copy(hasher, object)
	closeErr := object.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if size != expectedSize {
		return fmt.Errorf("stored object size mismatch: got %d, want %d", size, expectedSize)
	}
	if hex.EncodeToString(hasher.Sum(nil)) != expectedSHA256 {
		return fmt.Errorf("stored object sha256 mismatch")
	}
	return nil
}

func nilableString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nilableNonEmptyString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func randomConversionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (qq *S3FileSystem) copyVerifiedTemporaryTenantObject(
	ctx context.Context,
	filex *enttenant.StoredFile,
	tmpObjectName string,
	destObjectName string,
) error {
	expectedSHA256 := filex.Sha256
	expectedCRC32C := nilableString(filex.StorageCrc32c)
	_, err := qq.client.CopyObject(ctx, minio.CopyDestOptions{
		Bucket: qq.bucketName,
		Object: destObjectName,
	}, minio.CopySrcOptions{
		Bucket: qq.bucketName,
		Object: tmpObjectName,
	})
	if err != nil {
		return err
	}

	verificationErr := qq.verifyObject(
		ctx,
		destObjectName,
		filex.SizeInStorage,
		expectedSHA256,
		expectedCRC32C,
	)
	if verificationErr == nil {
		return nil
	}
	if err := qq.removeObjectIfExists(ctx, destObjectName); err != nil {
		return errors.Join(verificationErr, err)
	}
	err = qq.putTransformedObjectWithChecksum(
		ctx,
		tmpObjectName,
		destObjectName,
		filex.SizeInStorage,
		expectedSHA256,
		expectedCRC32C,
	)
	if err == nil {
		return nil
	}
	return errors.Join(err, qq.removeObjectIfExists(ctx, destObjectName))
}

func (qq *S3FileSystem) removeObjectIfExists(ctx context.Context, objectName string) error {
	err := qq.client.RemoveObject(ctx, qq.bucketName, objectName, minio.RemoveObjectOptions{})
	if minio.ToErrorResponse(err).Code == "NoSuchKey" {
		return nil
	}
	return err
}

func (qq *S3FileSystem) putTransformedObjectWithChecksum(
	ctx context.Context,
	sourceObjectName string,
	destObjectName string,
	expectedSize int64,
	expectedSHA256 string,
	expectedCRC32C string,
) error {
	source, err := qq.client.GetObject(ctx, qq.bucketName, sourceObjectName, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err := source.Close(); err != nil {
			log.Println(err)
		}
	}()

	_, err = qq.client.PutObject(ctx, qq.bucketName, destObjectName, source, expectedSize, minio.PutObjectOptions{
		NumThreads: 2,
		PartSize:   8 * 1024 * 1024,
		Checksum:   minio.ChecksumFullObjectCRC32C,
	})
	if err != nil {
		return err
	}
	return qq.verifyObject(ctx, destObjectName, expectedSize, expectedSHA256, expectedCRC32C)
}

func (qq *S3FileSystem) storageFilename(originalFilename, storageFilenameWithoutExt string) string {
	fileExtension := filepath.Ext(originalFilename)
	return storageFilenameWithoutExt + fileExtension + ".gz.age"
}

func (qq *S3FileSystem) saveFile(
	ctx context.Context,
	// passed in because PersistTemporaryTenantFile has no TenantContext
	x25519Identity *age.X25519Identity,
	fileToSave io.Reader,
	originalFilename string,
	storageFilenameWithoutExt string,
	storagePrefix string,
	onRawProgress func(time.Time),
) (*minio.UploadInfo, string, savedFileResult, error) {
	originalFilename = filepath.Clean(originalFilename)
	if !filenamex.IsAllowed(originalFilename) {
		log.Println("invalid filename")
		return nil, "", savedFileResult{}, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid filename.")
	}
	if qq.client == nil || qq.bucketName == "" {
		return nil, "", savedFileResult{}, e.NewHTTPErrorf(http.StatusInternalServerError, couldNotSaveFileMessage)
	}

	fileExtension := filepath.Ext(originalFilename)
	if fileExtension == "" {
		// commented because files don't need an extension...
		// log.Println("invalid filename")
		// return nil, "", 0, e.NewHTTPErrorf(http.StatusBadRequest, "File has no extension.")
	}

	// FIXME PublicID or private ID? does anybody see filenames?
	//		 public could be useful if somebody gets access to storage, but has no list priviledges
	//		 so he cannot iterate over all files;
	//		 public id also has advantage that we don't run into conflicts if a transaction fails
	//		 and we get on orphaned file in object storage;
	//		 PublicID also protects better against accessing files of another tenant through
	//		 programming mistackes
	storageFilename := storageFilenameWithoutExt + fileExtension + ".gz.age"

	objectName, err := securejoin.SecureJoin(storagePrefix, storageFilename)
	if err != nil {
		log.Println(err)
		return nil, "", savedFileResult{}, err
	}

	if qq.bucketName == "" {
		return nil, "", savedFileResult{}, e.NewHTTPErrorf(http.StatusInternalServerError, "Bucket name is empty.")
	}

	// database contraints should verify that each file just exists once, but this just
	// provides additional safety against accidental overwriting a file, for example if the
	// same PublicID is generated twice
	_, err = qq.client.StatObject(ctx, qq.bucketName, objectName, minio.StatObjectOptions{})
	if err == nil {
		log.Printf("filename already exists, should never happen, objectName was %s", objectName)
		return nil, "", savedFileResult{}, e.NewHTTPErrorf(http.StatusInternalServerError, "Filename already exists.")
	}

	// can maybe be further optimized by using a pipe for each step, but separating
	// the slow network write from encryption and compression should already bring the biggest
	// benefit, see also:
	// https://chatgpt.com/c/67f29855-e4ac-8000-9305-a5b63137e799
	pipeReader, pipeWriter := io.Pipe()
	resultCh := make(chan savedFileResult, 1)
	go func() {
		result := savedFileResult{}
		defer func() {
			if r := recover(); r != nil {
				result.err = fmt.Errorf("save file transform panic: %v", r)
				log.Println(result.err)
				_ = pipeWriter.CloseWithError(result.err)
			}
			resultCh <- result
		}()

		contentHasher := sha256.New()
		storageHasher := sha256.New()
		crc32cHasher := crc32.New(crc32.MakeTable(crc32.Castagnoli))
		metricWriter := &hashingWriter{
			w:      pipeWriter,
			sha256: storageHasher,
			crc32c: crc32cHasher,
		}

		rawReader := fileToSave
		if onRawProgress != nil {
			rawReader = &heartbeatReader{r: rawReader, onProgress: onRawProgress}
		}

		var gzipInputWriter io.Writer
		gzipInputWriter = metricWriter
		var encryptorx io.WriteCloser // necessary outside condition to close in correct order

		if !qq.disableFileEncryption {
			// gzip first, then encrypt because encryption randomizes data and is less efficient to
			// compress than the file directly
			encryptorx, err = age.Encrypt(metricWriter, x25519Identity.Recipient())
			if err != nil {
				log.Println(err)
				result.err = err
				_ = pipeWriter.CloseWithError(err)
				return
			}
			gzipInputWriter = encryptorx
		}

		gzipWriter := gzip.NewWriter(gzipInputWriter)

		var errs firstErr
		result.fileSize, err = io.Copy(gzipWriter, io.TeeReader(rawReader, contentHasher))
		errs.set(err)
		if err != nil {
			log.Println(err)
		}

		// IMPORTANT
		// order is important; pipeWriter last, not sure about gzipWriter and encryptor...
		err = gzipWriter.Close()
		errs.set(err)
		if !qq.disableFileEncryption {
			err = encryptorx.Close()
			errs.set(err)
		}
		if errs.err != nil {
			_ = pipeWriter.CloseWithError(errs.err)
			result.err = errs.err
			return
		}
		errs.set(pipeWriter.Close())

		result.err = errs.err
		result.contentSHA256 = hex.EncodeToString(contentHasher.Sum(nil))
		result.storageSize = metricWriter.count
		result.storageSHA256 = hex.EncodeToString(storageHasher.Sum(nil))
		result.storageCRC32C = base64.StdEncoding.EncodeToString(crc32cHasher.Sum(nil))
		if result.err != nil {
			log.Println(result.err)
		}
	}()

	// log.Println("debug: 001b")

	// TODO prefix
	// TODO checksum automatically?
	fileInfo, err := qq.client.PutObject(ctx, qq.bucketName, objectName, pipeReader, -1, minio.PutObjectOptions{
		ContentType:     "", // we don't have it yet... request header is not reliable and has strange values like `application/*`
		ContentEncoding: "", // gzip is not correct because encrypted... // TODO octet-stream or nothing?
		// ContentEncoding: "gzip",
		// ServerSideEncryption: nil,
		// https://min.io/docs/minio/linux/administration/object-management/object-retention.html#minio-object-locking-retention-modes
		// legal hold is complementary to retention
		// LegalHold: "", // TODO optional? per space?
		// Mode:      "", // retention
		Progress: nil,

		// reduces memory usage significantly
		NumThreads:            2,
		PartSize:              8 * 1024 * 1024, // default is 16 MB with 4 workers, 5MB is minimum, reduces memory usage
		ConcurrentStreamParts: true,
		Checksum:              minio.ChecksumFullObjectCRC32C,

		/*
			UserMetadata:            nil,
			UserTags:                nil,
			Progress:                nil, // can be used for progress bar, may require objectSize to be set
			ContentDisposition:      "",
			ContentLanguage:         "",
			CacheControl:            "",
			Expires:                 time.Time{},
			RetainUntilDate:         time.Time{},
			NumThreads:              0,
			StorageClass:            "",
			WebsiteRedirectLocation: "",
			PartSize:                0,
			SendContentMd5:          false,
			DisableContentSha256:    false,
			DisableMultipart:        false,

			AutoChecksum:          0,
			Checksum:              0,
			ConcurrentStreamParts: false,
			Internal:              minio.AdvancedPutOptions{},
		*/
	})
	if err != nil {
		_ = pipeReader.CloseWithError(err)
		result := <-resultCh
		if result.err != nil {
			log.Println(result.err)
			if errors.Is(result.err, errUploadTooLarge) {
				return nil, "", result, result.err
			}
			if isBadUploadReadError(result.err) {
				return nil, "", result, e.NewHTTPErrorf(http.StatusBadRequest, "Malformed upload body.")
			}
		}
		log.Println(err)
		return nil, "", savedFileResult{}, e.NewHTTPErrorf(http.StatusInternalServerError, couldNotSaveFileMessage)
	}
	_ = pipeReader.Close()

	// log.Println("debug: 002b")

	result := <-resultCh
	if result.err != nil {
		_ = qq.RemoveTemporaryObject(ctx, storagePrefix, storageFilename)
		if errors.Is(result.err, errUploadTooLarge) {
			return nil, "", result, result.err
		}
		if isBadUploadReadError(result.err) {
			return nil, "", result, e.NewHTTPErrorf(http.StatusBadRequest, "Malformed upload body.")
		}
		return nil, "", result, e.NewHTTPErrorf(http.StatusInternalServerError, couldNotSaveFileMessage)
	}

	fileInfo.Size = result.storageSize
	fileInfo.ChecksumSHA256 = result.storageSHA256
	fileInfo.ChecksumCRC32C = result.storageCRC32C
	fileInfo.ChecksumMode = minio.ChecksumFullObjectMode.String()

	if err := qq.verifyObject(
		ctx,
		objectName,
		result.storageSize,
		result.storageSHA256,
		result.storageCRC32C,
	); err != nil {
		log.Println(err)
		if cleanupErr := qq.RemoveTemporaryObject(ctx, storagePrefix, storageFilename); cleanupErr != nil {
			log.Println(cleanupErr)
		}
		return nil, "", result, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify stored file.")
	}

	return &fileInfo, storageFilename, result, nil
}

func isBadUploadReadError(err error) bool {
	return errors.Is(err, io.ErrUnexpectedEOF)
}

func (qq *S3FileSystem) x25519Identity(ctx ctxx.Context, objectNameOrStoragePrefix string) (*age.X25519Identity, error) {
	// is this implementation robust enough?

	if ctx.IsTenantCtx() && strings.HasPrefix(objectNameOrStoragePrefix, pathx.S3TenantPrefix()) {
		return ctx.TenantCtx().Tenant.X25519IdentityEncrypted.Identity(), nil
	}

	if strings.HasPrefix(objectNameOrStoragePrefix, pathx.S3AccountPrefix()) &&
		encryptor.NilableX25519MainIdentity != nil {
		return encryptor.NilableX25519MainIdentity, nil
	}

	return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not get x25519 identity.")
}

func (qq *S3FileSystem) PrepareTemporaryAccountUpload(
	ctx ctxx.Context,
	mainTx *entmain.Tx,
	originalFilename string,
	uploadToken string,
	fileIndex int,
	expiresAt time.Time,
) (*PreparedAccountUpload, error) {
	return qq.PrepareTemporaryAccountUploadWithSource(
		ctx,
		mainTx,
		originalFilename,
		uploadToken,
		fileIndex,
		expiresAt,
		filesource.PWAOSOpen,
	)
}

func (qq *S3FileSystem) PrepareTemporaryAccountUploadWithSource(
	ctx ctxx.Context,
	mainTx *entmain.Tx,
	originalFilename string,
	uploadToken string,
	fileIndex int,
	expiresAt time.Time,
	source filesource.FileSource,
) (*PreparedAccountUpload, error) {
	originalFilename = filepath.Clean(originalFilename)
	if !filenamex.IsAllowed(originalFilename) {
		log.Println("invalid filename")
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid filename.")
	}

	storageFilenameWithoutExt := fmt.Sprintf("%s-%06d", uploadToken, fileIndex)
	storagePrefix := pathx.S3TemporaryAccountStoragePrefix(ctx.MainCtx().Account.PublicID.String())
	storageFilename := qq.storageFilename(originalFilename, storageFilenameWithoutExt)

	now := time.Now()
	temporaryFile := mainTx.TemporaryFile.Create().
		SetOwner(ctx.MainCtx().Account).
		SetFilename(originalFilename).
		SetSource(source).
		SetSizeInStorage(0).
		SetStorageType(storagetype.S3).
		SetBucketName(qq.bucketName).
		SetStoragePath(storagePrefix).
		SetStorageFilename(storageFilename).
		SetUploadToken(uploadToken).
		SetUploadStartedAt(now).
		SetUploadLastProgressAt(now).
		SetExpiresAt(expiresAt).
		SaveX(ctx)

	return &PreparedAccountUpload{
		TemporaryFileID:           temporaryFile.ID,
		OriginalFilename:          originalFilename,
		StorageFilenameWithoutExt: storageFilenameWithoutExt,
		StorageFilename:           storageFilename,
		StoragePath:               storagePrefix,
	}, nil
}

func (qq *S3FileSystem) UploadPreparedTemporaryAccountFile(
	ctx ctxx.Context,
	fileToSave io.Reader,
	prepared *PreparedAccountUpload,
) (*PreparedUploadResult, error) {
	nilableUploadLimitBytes, err := qq.NilableEffectiveUploadSizeLimitBytes(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	limitedFileToSave := fileToSave
	if nilableUploadLimitBytes != nil {
		limitedFileToSave = &maxBytesReader{r: fileToSave, max: *nilableUploadLimitBytes}
	}

	fileInfo, fileSize, contentSHA256, storageCRC32C, err := qq.uploadPreparedFileWithParams(
		ctx,
		limitedFileToSave,
		prepared.OriginalFilename,
		prepared.StorageFilenameWithoutExt,
		prepared.StoragePath,
		prepared.StorageFilename,
		qq.temporaryFileUploadHeartbeat(ctx, prepared.TemporaryFileID),
	)
	if err != nil {
		if errors.Is(err, errUploadTooLarge) && nilableUploadLimitBytes != nil {
			return nil, qq.uploadTooLargeError(*nilableUploadLimitBytes)
		}
		return nil, err
	}
	if fileSize == 0 {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, emptyUploadMessage)
	}

	if nilableUploadLimitBytes != nil && fileSize > *nilableUploadLimitBytes {
		return nil, qq.uploadTooLargeError(*nilableUploadLimitBytes)
	}

	return &PreparedUploadResult{
		FileInfo:      fileInfo,
		FileSize:      fileSize,
		ContentSHA256: contentSHA256,
		StorageCRC32C: storageCRC32C,
	}, nil
}

func (qq *S3FileSystem) uploadPreparedFileWithParams(
	ctx ctxx.Context,
	fileToSave io.Reader,
	originalFilename string,
	storageFilenameWithoutExt string,
	storagePath string,
	expectedStorageFilename string,
	onRawProgress func(time.Time),
) (*minio.UploadInfo, int64, string, string, error) {
	x25519Identity, err := qq.x25519Identity(ctx, storagePath)
	if err != nil {
		log.Println(err)
		return nil, 0, "", "", err
	}

	fileInfo, storageFilename, result, err := qq.saveFile(
		ctx,
		x25519Identity,
		fileToSave,
		originalFilename,
		storageFilenameWithoutExt,
		storagePath,
		onRawProgress,
	)
	if err != nil {
		log.Println(err)
		return nil, 0, "", "", err
	}

	if storageFilename != expectedStorageFilename {
		log.Println("storage filename mismatch", storageFilename, expectedStorageFilename)
		return nil, 0, "", "", e.NewHTTPErrorf(http.StatusInternalServerError, "Storage filename mismatch.")
	}

	return fileInfo, result.fileSize, result.contentSHA256, result.storageCRC32C, nil
}

func (qq *S3FileSystem) FinalizePreparedTemporaryAccountUpload(
	ctx ctxx.Context,
	mainTx *entmain.Tx,
	prepared *PreparedAccountUpload,
	result *PreparedUploadResult,
) error {
	ctxWithIncomplete := entmainschema.WithUnfinishedUploads(ctx)
	mainTx.TemporaryFile.
		UpdateOneID(prepared.TemporaryFileID).
		SetSize(result.FileSize).
		SetSizeInStorage(result.FileInfo.Size).
		SetSha256(result.FileInfo.ChecksumSHA256).
		SetContentSha256(result.ContentSHA256).
		SetStorageCrc32c(result.StorageCRC32C).
		SetUploadSucceededAt(time.Now()).
		SaveX(ctxWithIncomplete)

	return nil
}

func (qq *S3FileSystem) PreparePersistingTemporaryAccountFile(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	parentDirFileID int64,
	isInInbox bool,
) (*enttenant.File, error) {
	return qq.PersistTemporaryAccountFile(ctx, tmpFile, parentDirFileID, isInInbox)
}

func (qq *S3FileSystem) PersistTemporaryAccountFile(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	parentDirFileID int64,
	isInInbox bool,
) (*enttenant.File, error) {
	err := qq.EnsureUploadSizeLimit(ctx, tmpFile.Size)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = qq.EnsureTenantStorageLimit(ctx, tmpFile.Size)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return qq.persistTemporaryAccountFile(ctx, tmpFile, parentDirFileID, isInInbox)
}

func (qq *S3FileSystem) persistTemporaryAccountFile(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	parentDirFileID int64,
	isInInbox bool,
) (*enttenant.File, error) {
	if filex, ok, err := qq.nilableSuccessfulAccountConversion(ctx, tmpFile); err != nil {
		return nil, err
	} else if ok {
		return filex, nil
	}
	if filex, ok, err := qq.nilableUnlinkedSuccessfulAccountConversion(
		ctx,
		tmpFile,
		parentDirFileID,
		isInInbox,
	); err != nil {
		return nil, err
	} else if ok {
		return filex, nil
	}

	claimToken, err := randomConversionToken()
	if err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not claim uploaded file.")
	}

	claimedTmpFile, err := qq.claimTemporaryAccountConversion(ctx, tmpFile, claimToken)
	if err != nil {
		return nil, err
	}
	tmpFile = claimedTmpFile
	conversionDone := false
	defer func() {
		if conversionDone {
			return
		}
		if err := qq.releaseTemporaryAccountConversionClaim(ctx, tmpFile.ID, claimToken); err != nil {
			log.Println(err)
		}
	}()

	storedFilex, err := qq.prepareClaimedAccountConversion(ctx, tmpFile, claimToken)
	if err != nil {
		return nil, err
	}
	defer func() {
		if conversionDone {
			return
		}
		if err := qq.cleanupClaimedAccountConversion(ctx, storedFilex, claimToken); err != nil {
			log.Println(err)
		}
	}()

	accountObjectName, err := securejoin.SecureJoin(tmpFile.StoragePath, tmpFile.StorageFilename)
	if err != nil {
		return nil, err
	}
	if err := qq.verifyObjectStrict(
		ctx,
		accountObjectName,
		tmpFile.SizeInStorage,
		tmpFile.Sha256,
		nilableString(tmpFile.StorageCrc32c),
	); err != nil {
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify staged file.")
	}

	mainX25519Identity := encryptor.NilableX25519MainIdentity
	if mainX25519Identity == nil {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "App not unlocked yet.")
	}

	accountFile := &enttenant.StoredFile{
		Filename:                 tmpFile.Filename,
		TemporaryStoragePath:     tmpFile.StoragePath,
		TemporaryStorageFilename: tmpFile.StorageFilename,
	}
	plaintext, err := qq.UnsafeOpenFile(ctx, mainX25519Identity, storedfilemodel.NewStoredFile(accountFile))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer func() {
		if err := plaintext.Close(); err != nil {
			log.Println(err)
		}
	}()

	storageFilenameWithoutExt := strings.TrimSuffix(storedFilex.StorageFilename, ".gz.age")
	storageFilenameWithoutExt = strings.TrimSuffix(storageFilenameWithoutExt, filepath.Ext(storageFilenameWithoutExt))
	fileInfo, storageFilename, result, err := qq.saveFile(
		ctx,
		ctx.TenantCtx().Tenant.X25519IdentityEncrypted.Identity(),
		plaintext,
		storedFilex.Filename,
		storageFilenameWithoutExt,
		storedFilex.TemporaryStoragePath,
		qq.accountConversionHeartbeat(ctx, tmpFile.ID, storedFilex.ID, claimToken),
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if storageFilename != storedFilex.TemporaryStorageFilename {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Storage filename mismatch.")
	}
	if result.fileSize != tmpFile.Size || result.contentSHA256 != tmpFile.ContentSha256 {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Staged file integrity mismatch.")
	}

	filex, err := qq.finalizeClaimedAccountConversion(
		ctx,
		tmpFile,
		storedFilex.ID,
		claimToken,
		parentDirFileID,
		isInInbox,
		fileInfo,
		result,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	conversionDone = true
	return filex, nil
}

func (qq *S3FileSystem) claimTemporaryAccountConversion(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	claimToken string,
) (*entmain.TemporaryFile, error) {
	now := time.Now()
	staleBefore := now.Add(-accountConversionTakeoverAfter)
	updated, err := ctx.MainCtx().UnsafeMainDB().ReadWriteConn.TemporaryFile.UpdateOneID(tmpFile.ID).
		Where(
			temporaryfile.ConvertedToStoredFileAtIsNil(),
			temporaryfile.DeletedAtIsNil(),
			temporaryfile.Or(
				temporaryfile.PersistenceClaimTokenIsNil(),
				temporaryfile.PersistenceLastProgressAtIsNil(),
				temporaryfile.PersistenceLastProgressAtLT(staleBefore),
			),
		).
		SetPersistenceClaimToken(claimToken).
		SetPersistenceTenantID(ctx.TenantCtx().Tenant.ID).
		SetPersistenceLastProgressAt(now).
		ClearExpiresAt().
		Save(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			if filex, ok, reuseErr := qq.nilableSuccessfulAccountConversion(ctx, tmpFile); reuseErr != nil {
				return nil, reuseErr
			} else if ok && filex != nil {
				return ctx.MainCtx().UnsafeMainDB().ReadOnlyConn.TemporaryFile.Get(ctx, tmpFile.ID)
			}
			return nil, e.NewHTTPErrorf(http.StatusConflict, "Uploaded file is already being processed.")
		}
		log.Println(err)
		return nil, err
	}
	return updated, nil
}

func (qq *S3FileSystem) nilableSuccessfulAccountConversion(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
) (*enttenant.File, bool, error) {
	ctxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(ctx),
		tenantprivacy.Allow,
	)
	tenantDB, ok := ctx.TenantCtx().UnsafeTenantDB()
	if !ok {
		return nil, false, e.NewHTTPErrorf(http.StatusInternalServerError, tenantDatabaseNotFoundMessage)
	}
	storedFilex, err := tenantDB.ReadOnlyConn.StoredFile.Query().
		Where(
			storedfile.SourceTemporaryFilePublicID(entx.NewCIText(tmpFile.PublicID.String())),
			storedfile.UploadSucceededAtNotNil(),
		).
		Only(ctxWithIncomplete)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, false, nil
		}
		log.Println(err)
		return nil, false, err
	}

	fileVersionx, err := tenantDB.ReadOnlyConn.FileVersion.Query().
		Where(fileversion.StoredFileID(storedFilex.ID)).
		WithFile().
		Only(ctxWithIncomplete)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, false, nil
		}
		log.Println(err)
		return nil, false, err
	}

	if err := qq.markTemporaryAccountConversionDone(ctx, tmpFile.ID); err != nil {
		return nil, false, err
	}
	return fileVersionx.Edges.File, true, nil
}

func (qq *S3FileSystem) nilableUnlinkedSuccessfulAccountConversion(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	parentDirFileID int64,
	isInInbox bool,
) (*enttenant.File, bool, error) {
	tenantDB, ok := ctx.TenantCtx().UnsafeTenantDB()
	if !ok {
		return nil, false, e.NewHTTPErrorf(http.StatusInternalServerError, tenantDatabaseNotFoundMessage)
	}
	ctxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(ctx),
		tenantprivacy.Allow,
	)
	storedFilex, err := tenantDB.ReadOnlyConn.StoredFile.Query().
		Where(
			storedfile.SourceTemporaryFilePublicID(entx.NewCIText(tmpFile.PublicID.String())),
			storedfile.UploadSucceededAtNotNil(),
			storedfile.Not(storedfile.HasFileVersions()),
		).
		Only(ctxWithIncomplete)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, false, nil
		}
		log.Println(err)
		return nil, false, err
	}

	tenantTx, err := tenantDB.Tx(ctx, false)
	if err != nil {
		log.Println(err)
		return nil, false, err
	}
	committed := false
	defer func() {
		if !committed {
			if err := tenantTx.Rollback(); err != nil {
				log.Println(err)
			}
		}
	}()

	writeTenantCtx := ctxx.NewTenantContext(ctx.MainCtx(), tenantTx, ctx.TenantCtx().Tenant, false)
	writeSpace := tenantTx.Space.GetX(writeTenantCtx, ctx.SpaceCtx().Space.ID)
	writeSpaceCtx := ctxx.NewSpaceContext(writeTenantCtx, writeSpace)
	writeCtxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(writeSpaceCtx),
		tenantprivacy.Allow,
	)
	storedFilex = tenantTx.StoredFile.GetX(writeCtxWithIncomplete, storedFilex.ID)
	filex := tenantTx.File.Create().
		SetName(tmpFile.Filename).
		SetSource(tmpFile.Source).
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		SetParentID(parentDirFileID).
		SetSpaceID(writeSpace.ID).
		SetIsInInbox(isInInbox).
		SaveX(writeSpaceCtx)
	if err := qq.addFileVersion(writeSpaceCtx, filex, storedFilex); err != nil {
		return nil, false, err
	}
	if err := tenantTx.Commit(); err != nil {
		log.Println(err)
		return nil, false, err
	}
	committed = true

	if err := qq.markTemporaryAccountConversionDone(ctx, tmpFile.ID); err != nil {
		return nil, false, err
	}
	return filex, true, nil
}

func (qq *S3FileSystem) prepareClaimedAccountConversion(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	claimToken string,
) (*enttenant.StoredFile, error) {
	ctxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(ctx),
		tenantprivacy.Allow,
	)
	tenantDB, ok := ctx.TenantCtx().UnsafeTenantDB()
	if !ok {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, tenantDatabaseNotFoundMessage)
	}
	storedFilex, err := tenantDB.ReadOnlyConn.StoredFile.Query().
		Where(storedfile.SourceTemporaryFilePublicID(entx.NewCIText(tmpFile.PublicID.String()))).
		Only(ctxWithIncomplete)
	if err == nil {
		if storedFilex.UploadSucceededAt != nil {
			return storedFilex, nil
		}
		if nilableString(storedFilex.SourceConversionClaimToken) != claimToken {
			if err := qq.cleanupClaimedAccountConversion(ctx, storedFilex, nilableString(storedFilex.SourceConversionClaimToken)); err != nil {
				return nil, err
			}
			return qq.createClaimedAccountConversion(ctx, tmpFile, claimToken)
		}
		return storedFilex, nil
	}
	if !enttenant.IsNotFound(err) {
		log.Println(err)
		return nil, err
	}
	return qq.createClaimedAccountConversion(ctx, tmpFile, claimToken)
}

func (qq *S3FileSystem) createClaimedAccountConversion(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	claimToken string,
) (*enttenant.StoredFile, error) {
	finalStoragePath := pathx.S3StoragePrefix(ctx.TenantCtx().TenantID)
	temporaryStoragePath := pathx.S3TemporaryStoragePrefix(ctx.TenantCtx().TenantID)
	storageFilenameWithoutExt := util.NewPublicID()
	storageFilename := qq.storageFilename(tmpFile.Filename, storageFilenameWithoutExt)
	now := time.Now()
	ctxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(ctx),
		tenantprivacy.Allow,
	)
	tenantDB, ok := ctx.TenantCtx().UnsafeTenantDB()
	if !ok {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, tenantDatabaseNotFoundMessage)
	}
	return tenantDB.ReadWriteConn.StoredFile.Create().
		SetFilename(tmpFile.Filename).
		SetSize(0).
		SetSizeInStorage(0).
		SetStorageType(storagetype.S3).
		SetBucketName(qq.bucketName).
		SetStoragePath(finalStoragePath).
		SetStorageFilename(storageFilename).
		SetTemporaryStoragePath(temporaryStoragePath).
		SetTemporaryStorageFilename(storageFilename).
		SetSourceTemporaryFilePublicID(entx.NewCIText(tmpFile.PublicID.String())).
		SetSourceConversionClaimToken(claimToken).
		SetUploadStartedAt(now).
		SetUploadLastProgressAt(now).
		Save(ctxWithIncomplete)
}

func (qq *S3FileSystem) finalizeClaimedAccountConversion(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	storedFileID int64,
	claimToken string,
	parentDirFileID int64,
	isInInbox bool,
	fileInfo *minio.UploadInfo,
	result savedFileResult,
) (*enttenant.File, error) {
	mainDB := ctx.MainCtx().UnsafeMainDB()
	tenantDB, ok := ctx.TenantCtx().UnsafeTenantDB()
	if !ok {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, tenantDatabaseNotFoundMessage)
	}

	mainTx, err := mainDB.Tx(ctx, false)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	committedMain := false
	defer func() {
		if !committedMain {
			if err := mainTx.Rollback(); err != nil {
				log.Println(err)
			}
		}
	}()

	claimedTmpFile, err := mainTx.TemporaryFile.Query().
		Where(
			temporaryfile.ID(tmpFile.ID),
			temporaryfile.PersistenceClaimToken(claimToken),
			temporaryfile.ConvertedToStoredFileAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusConflict, "Uploaded file processing was taken over.")
		}
		log.Println(err)
		return nil, err
	}

	filex, err := qq.finalizeClaimedAccountConversionInTenant(
		ctx,
		tenantDB,
		claimedTmpFile,
		storedFileID,
		claimToken,
		parentDirFileID,
		isInInbox,
		fileInfo,
		result,
	)
	if err != nil {
		return nil, err
	}

	if err := mainTx.TemporaryFile.UpdateOneID(claimedTmpFile.ID).
		Where(temporaryfile.PersistenceClaimToken(claimToken)).
		SetConvertedToStoredFileAt(time.Now()).
		ClearPersistenceClaimToken().
		ClearPersistenceTenantID().
		ClearPersistenceLastProgressAt().
		ClearExpiresAt().
		Exec(ctx); err != nil {
		log.Println(err)
		return nil, err
	}

	if err := mainTx.Commit(); err != nil {
		log.Println(err)
		return nil, err
	}
	committedMain = true

	return filex, nil
}

func (qq *S3FileSystem) finalizeClaimedAccountConversionInTenant(
	ctx ctxx.Context,
	tenantDB *sqlx.TenantDB,
	claimedTmpFile *entmain.TemporaryFile,
	storedFileID int64,
	claimToken string,
	parentDirFileID int64,
	isInInbox bool,
	fileInfo *minio.UploadInfo,
	result savedFileResult,
) (*enttenant.File, error) {
	tenantTx, err := tenantDB.Tx(ctx, false)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			if err := tenantTx.Rollback(); err != nil {
				log.Println(err)
			}
		}
	}()

	writeTenantCtx := ctxx.NewTenantContext(ctx.MainCtx(), tenantTx, ctx.TenantCtx().Tenant, false)
	writeSpace := tenantTx.Space.GetX(writeTenantCtx, ctx.SpaceCtx().Space.ID)
	writeSpaceCtx := ctxx.NewSpaceContext(writeTenantCtx, writeSpace)
	ctxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(writeSpaceCtx),
		tenantprivacy.Allow,
	)

	storedFilex, err := tenantTx.StoredFile.Query().
		Where(
			storedfile.ID(storedFileID),
			storedfile.SourceTemporaryFilePublicID(entx.NewCIText(claimedTmpFile.PublicID.String())),
			storedfile.SourceConversionClaimToken(claimToken),
		).
		Only(ctxWithIncomplete)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusConflict, "Uploaded file processing was taken over.")
		}
		log.Println(err)
		return nil, err
	}

	if storedFilex.UploadSucceededAt == nil {
		storedFilex, err = storedFilex.Update().
			SetSize(result.fileSize).
			SetSizeInStorage(fileInfo.Size).
			SetSha256(fileInfo.ChecksumSHA256).
			SetContentSha256(result.contentSHA256).
			SetStorageCrc32c(result.storageCRC32C).
			SetUploadSucceededAt(time.Now()).
			Save(ctxWithIncomplete)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	}

	filex := tenantTx.File.Create().
		SetName(claimedTmpFile.Filename).
		SetSource(claimedTmpFile.Source).
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		SetParentID(parentDirFileID).
		SetSpaceID(writeSpace.ID).
		SetIsInInbox(isInInbox).
		SaveX(writeSpaceCtx)

	if err := qq.addFileVersion(writeSpaceCtx, filex, storedFilex); err != nil {
		log.Println(err)
		return nil, err
	}

	if err := tenantTx.Commit(); err != nil {
		log.Println(err)
		return nil, err
	}
	committed = true

	return filex, nil
}

func (qq *S3FileSystem) markTemporaryAccountConversionDone(ctx ctxx.Context, temporaryFileID int64) error {
	return ctx.MainCtx().UnsafeMainDB().ReadWriteConn.TemporaryFile.UpdateOneID(temporaryFileID).
		SetConvertedToStoredFileAt(time.Now()).
		ClearPersistenceClaimToken().
		ClearPersistenceTenantID().
		ClearPersistenceLastProgressAt().
		ClearExpiresAt().
		Exec(ctx)
}

func (qq *S3FileSystem) accountConversionHeartbeat(
	ctx ctxx.Context,
	temporaryFileID int64,
	storedFileID int64,
	claimToken string,
) func(time.Time) {
	tenantDB, ok := ctx.TenantCtx().UnsafeTenantDB()
	if !ok {
		return nil
	}
	return func(now time.Time) {
		go func() {
			heartbeatCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			mainErr := ctx.MainCtx().UnsafeMainDB().ReadWriteConn.TemporaryFile.UpdateOneID(temporaryFileID).
				Where(temporaryfile.PersistenceClaimToken(claimToken)).
				SetPersistenceLastProgressAt(now).
				Exec(heartbeatCtx)
			if mainErr != nil && !errors.Is(mainErr, context.DeadlineExceeded) {
				log.Println(mainErr)
			}

			ctxWithIncomplete := tenantprivacy.DecisionContext(
				enttenantschema.WithUnfinishedUploads(heartbeatCtx),
				tenantprivacy.Allow,
			)
			tenantErr := tenantDB.ReadWriteConn.StoredFile.UpdateOneID(storedFileID).
				Where(storedfile.SourceConversionClaimToken(claimToken)).
				SetUploadLastProgressAt(now).
				Exec(ctxWithIncomplete)
			if tenantErr != nil && !errors.Is(tenantErr, context.DeadlineExceeded) {
				log.Println(tenantErr)
			}
		}()
	}
}

func (qq *S3FileSystem) cleanupClaimedAccountConversion(
	ctx ctxx.Context,
	storedFilex *enttenant.StoredFile,
	claimToken string,
) error {
	if storedFilex == nil || claimToken == "" {
		return nil
	}
	ctxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(ctx),
		tenantprivacy.Allow,
	)
	tenantDB, ok := ctx.TenantCtx().UnsafeTenantDB()
	if !ok {
		return e.NewHTTPErrorf(http.StatusInternalServerError, tenantDatabaseNotFoundMessage)
	}
	current, err := tenantDB.ReadOnlyConn.StoredFile.Query().
		Where(
			storedfile.ID(storedFilex.ID),
			storedfile.SourceConversionClaimToken(claimToken),
			storedfile.UploadSucceededAtIsNil(),
		).
		Only(ctxWithIncomplete)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil
		}
		log.Println(err)
		return err
	}
	if err := qq.RemoveTemporaryObject(ctx, current.TemporaryStoragePath, current.TemporaryStorageFilename); err != nil {
		log.Println(err)
		return err
	}
	return tenantDB.ReadWriteConn.StoredFile.DeleteOneID(current.ID).Exec(ctxWithIncomplete)
}

func (qq *S3FileSystem) releaseTemporaryAccountConversionClaim(
	ctx ctxx.Context,
	temporaryFileID int64,
	claimToken string,
) error {
	err := ctx.MainCtx().UnsafeMainDB().ReadWriteConn.TemporaryFile.UpdateOneID(temporaryFileID).
		Where(temporaryfile.PersistenceClaimToken(claimToken)).
		ClearPersistenceClaimToken().
		ClearPersistenceTenantID().
		ClearPersistenceLastProgressAt().
		SetExpiresAt(time.Now().Add(temporaryAccountFileExpiry)).
		Exec(ctx)
	if entmain.IsNotFound(err) {
		return nil
	}
	return err
}

func (qq *S3FileSystem) verifyObjectStrict(
	ctx context.Context,
	objectName string,
	expectedSize int64,
	expectedSHA256 string,
	expectedCRC32C string,
) error {
	if expectedSHA256 == "" {
		return errors.New("stored object sha256 is missing")
	}
	return qq.verifyObject(ctx, objectName, expectedSize, expectedSHA256, expectedCRC32C)
}

func (qq *S3FileSystem) PersistTemporaryTenantFile(
	ctx context.Context,
	tenantX25519Identity *age.X25519Identity,
	filex *enttenant.StoredFile,
) error {
	filem := storedfilemodel.NewStoredFile(filex)

	destObjectName, err := filem.UnsafeFinalObjectNameWithPrefix()
	if err != nil {
		log.Println(err)
		return err
	}

	tmpObjectName, err := filem.UnsafeTempObjectNameWithPrefix()
	if err != nil {
		log.Println(err)
		return err
	}

	// A verified destination means the copy succeeded before its database update.
	if err = qq.verifyObject(
		ctx,
		destObjectName,
		filex.SizeInStorage,
		filex.Sha256,
		nilableString(filex.StorageCrc32c),
	); err == nil {
		log.Printf("dest file already exists and is verified, marking stored file as copied, tmpObjectName: %s, destObjectName: %s", tmpObjectName, destObjectName)
		return filex.Update().SetCopiedToFinalDestinationAt(time.Now()).Exec(ctx)
	}
	minioErr := minio.ToErrorResponse(err)
	if minioErr.Code != "NoSuchKey" {
		log.Println(err)
		if cleanupErr := qq.removeObjectIfExists(ctx, destObjectName); cleanupErr != nil {
			log.Println(cleanupErr)
			return errors.Join(err, cleanupErr)
		}
	}

	if strings.HasPrefix(destObjectName, pathx.S3TenantPrefix()) && strings.HasPrefix(tmpObjectName, pathx.S3TenantPrefix()) {
		if err := qq.copyVerifiedTemporaryTenantObject(ctx, filex, tmpObjectName, destObjectName); err != nil {
			log.Println(err)
			return err
		}
	} else if strings.HasPrefix(destObjectName, pathx.S3TenantPrefix()) && strings.HasPrefix(tmpObjectName, pathx.S3AccountPrefix()) {
		mainX25519Identity := encryptor.NilableX25519MainIdentity
		if mainX25519Identity == nil {
			log.Println("App not unlocked yet.")
			return e.NewHTTPErrorf(http.StatusInternalServerError, "App not unlocked yet.")
		}

		// reencrypt with tenant key
		tmpFile, err := qq.UnsafeOpenFile(ctx, mainX25519Identity, filem)
		if err != nil {
			log.Println(err)
			return err
		}
		defer func() {
			err = tmpFile.Close()
			if err != nil {
				log.Println(err)
			}
		}()

		storageFilenameWithoutExt := strings.TrimSuffix(filem.Data.StorageFilename, ".gz.age")
		// remove file extension of original file, for example pdf
		storageFilenameWithoutExt = strings.TrimSuffix(storageFilenameWithoutExt, filepath.Ext(storageFilenameWithoutExt))

		fileInfo, _, result, err := qq.saveFile(
			ctx,
			tenantX25519Identity,
			tmpFile,
			filem.Data.Filename,
			storageFilenameWithoutExt,
			filem.Data.StoragePath,
			nil,
		)
		if err != nil {
			log.Println(err)
			return err
		}

		filex = filex.Update().
			SetSizeInStorage(fileInfo.Size).
			SetSha256(fileInfo.ChecksumSHA256).
			SetContentSha256(result.contentSHA256).
			SetStorageCrc32c(result.storageCRC32C).
			SaveX(ctx)
	} else {
		err = e.NewHTTPErrorf(http.StatusInternalServerError, "Could not copy temporary file.")
		log.Println(err, "may need manual cleanup or missing configuration")
		return err
	}

	err = filex.Update().SetCopiedToFinalDestinationAt(time.Now()).Exec(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (qq *S3FileSystem) addFileVersion(ctx ctxx.Context, filex *enttenant.File, storedFilex *enttenant.StoredFile) error {
	latestVersion, err := ctx.TenantCtx().TTx.FileVersion.Query().
		Where(fileversion.FileID(filex.ID)).
		Order(fileversion.ByVersionNumber(sql.OrderDesc())).
		First(ctx)
	if err != nil && !enttenant.IsNotFound(err) {
		log.Println(err)
		return err
	}
	versionNumber := 1
	if err == nil {
		versionNumber = latestVersion.VersionNumber + 1
	}
	ctx.TenantCtx().TTx.FileVersion.Create().
		SetFileID(filex.ID).
		SetStoredFileID(storedFilex.ID).
		SetVersionNumber(versionNumber).
		SaveX(ctx)

	filex.Update().
		SetOcrContent("").
		ClearOcrSuccessAt().
		SetOcrRetryCount(0).
		SetOcrLastTriedAt(time.Time{}).
		ExecX(ctx)
	return nil
}

// near duplicate in FileSystem
func (qq *S3FileSystem) UpdateMimeType(ctx ctxx.Context, force bool, filex *storedfilemodel.StoredFile) (string, error) {
	if filex.Data.MimeType != "" && !force {
		return filex.Data.MimeType, nil
	}

	mimeType, err := qq.DetectMimeType(ctx, filex)
	if err != nil {
		return "", err
	}
	filex.Data = filex.Data.Update().SetMimeType(mimeType).SaveX(ctx)

	return mimeType, nil
}

func (qq *S3FileSystem) UpdateMimeTypeAfterFinalization(
	ctx ctxx.Context,
	force bool,
	storedFileID int64,
) (string, error) {
	tenantDB, ok := ctx.TenantCtx().UnsafeTenantDB()
	if !ok {
		return "", e.NewHTTPErrorf(http.StatusInternalServerError, tenantDatabaseNotFoundMessage)
	}

	storedFilex, err := tenantDB.ReadOnlyConn.StoredFile.Get(ctx, storedFileID)
	if err != nil {
		log.Println(err)
		return "", err
	}
	if storedFilex.MimeType != "" && !force {
		return storedFilex.MimeType, nil
	}

	mimeType, err := qq.DetectMimeType(ctx, storedfilemodel.NewStoredFile(storedFilex))
	if err != nil {
		return "", err
	}

	if err := tenantDB.ReadWriteConn.StoredFile.UpdateOneID(storedFileID).
		SetMimeType(mimeType).
		Exec(ctx); err != nil && !enttenant.IsNotFound(err) {
		log.Println(err)
		return "", err
	}

	return mimeType, nil
}

func (qq *S3FileSystem) DetectMimeType(ctx ctxx.Context, filex *storedfilemodel.StoredFile) (string, error) {
	obj, err := qq.OpenFile(ctx, filex)
	if err != nil {
		log.Println(err)
		return "", e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}
	defer obj.Close()

	mimeType, err := detectMIME(obj)
	if err != nil {
		log.Println(err)
		return "", e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}

	return mimeType, nil
}

func (qq *S3FileSystem) EnsureTenantStorageLimit(ctx ctxx.Context, incomingUploadedBytes int64) error {
	return qq.storageQuota.EnsureTenantStorageLimit(ctx, incomingUploadedBytes)
}

func (qq *S3FileSystem) EnsureUploadSizeLimit(ctx ctxx.Context, incomingUploadedBytes int64) error {
	if incomingUploadedBytes <= 0 {
		return nil
	}

	nilableUploadLimitBytes, err := qq.NilableEffectiveUploadSizeLimitBytes(ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	if nilableUploadLimitBytes == nil {
		return nil
	}

	if incomingUploadedBytes > *nilableUploadLimitBytes {
		return qq.uploadTooLargeError(*nilableUploadLimitBytes)
	}

	return nil
}

func (qq *S3FileSystem) NilableEffectiveUploadSizeLimitBytes(ctx ctxx.Context) (*int64, error) {
	globalLimitBytes, err := qq.globalUploadSizeLimitBytes(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	effectiveLimitBytes := globalLimitBytes
	if ctx.IsTenantCtx() {
		nilableTenantOverride := ctx.TenantCtx().Tenant.MaxUploadSizeMibOverride
		if nilableTenantOverride != nil {
			effectiveLimitBytes = *nilableTenantOverride * bytesPerMiB
		}
	}

	if effectiveLimitBytes <= 0 {
		return nil, nil
	}

	return &effectiveLimitBytes, nil
}

func (qq *S3FileSystem) globalUploadSizeLimitBytes(ctx ctxx.Context) (int64, error) {
	client := ctx.MainCtx().MainTx.SystemConfig
	if mainDB := ctx.MainCtx().UnsafeMainDB(); mainDB != nil {
		client = mainDB.ReadOnlyConn.SystemConfig
	}
	systemConfigx, err := client.Query().
		Select(systemconfig.FieldMaxUploadSizeMib).
		First(ctx)
	if err != nil {
		log.Println(err)
		return 0, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify upload size limit.")
	}

	if systemConfigx.MaxUploadSizeMib <= 0 {
		return 0, nil
	}

	return systemConfigx.MaxUploadSizeMib * bytesPerMiB, nil
}

func (qq *S3FileSystem) uploadTooLargeError(maxUploadSizeBytes int64) error {
	if maxUploadSizeBytes <= 0 {
		return e.NewHTTPErrorf(http.StatusRequestEntityTooLarge, "Upload is too large.")
	}

	return e.NewHTTPErrorf(
		http.StatusRequestEntityTooLarge,
		"Upload is too large. Maximum allowed size is %s.",
		fileutil.FormatSize(maxUploadSizeBytes),
	)
}
