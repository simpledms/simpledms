package filesystem

import (
	"compress/gzip"
	"context"
	"fmt"
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
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/encryptor"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/model/common/storagetype"
	"github.com/simpledms/simpledms/pathx"
	"github.com/simpledms/simpledms/util"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
	"github.com/simpledms/simpledms/util/recoverx"
)

type S3FileSystem struct {
	*FileSystem
	client                *minio.Client
	bucketName            string
	disableFileEncryption bool
}

func NewS3FileSystem(client *minio.Client, bucketName string, fileSystem *FileSystem, disableFileEncryption bool) *S3FileSystem {
	return &S3FileSystem{
		FileSystem:            fileSystem,
		client:                client,
		bucketName:            bucketName,
		disableFileEncryption: disableFileEncryption,
	}
}

// caller has to close io.ReadCloser
// TODO OpenFile or CopyFile?
func (qq *S3FileSystem) OpenFile(ctx ctxx.Context, file *model.StoredFile) (io.ReadCloser, error) {
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
func (qq *S3FileSystem) UnsafeOpenFile(ctx context.Context, x25519Identity *age.X25519Identity, file *model.StoredFile) (io.ReadCloser, error) {
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

func (qq *S3FileSystem) PrepareFileUpload(
	ctx ctxx.Context,
	originalFilename string,
	parentDirFileID int64,
	isInInbox bool,
) (*PreparedUpload, *enttenant.File, error) {
	meta, err := qq.prepareUploadMetadata(ctx, originalFilename)
	if err != nil {
		return nil, nil, err
	}
	if err := qq.ensureFileDoesNotExistInFolderMode(ctx, meta.originalFilename, parentDirFileID, isInInbox); err != nil {
		return nil, nil, err
	}

	filex := ctx.TenantCtx().TTx.File.Create().
		SetName(meta.originalFilename).
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		SetParentID(parentDirFileID).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		SetIsInInbox(isInInbox).
		SaveX(ctx)

	storedFilex, prepared, err := qq.createStoredFileForPreparedUpload(ctx, meta)
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}

	if err := qq.addFileVersion(ctx, filex, storedFilex); err != nil {
		log.Println(err)
		return nil, nil, err
	}

	return prepared, filex, nil
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

	filex.Update().SetName(meta.originalFilename).SaveX(ctx)

	storedFilex, prepared, err := qq.createStoredFileForPreparedUpload(ctx, meta)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if err := qq.addFileVersion(ctx, filex, storedFilex); err != nil {
		log.Println(err)
		return nil, err
	}

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
) (*enttenant.StoredFile, *PreparedUpload, error) {
	finalStoragePrefix := pathx.S3StoragePrefix(ctx.TenantCtx().TenantID)
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
		SaveX(ctx)

	prepared := &PreparedUpload{
		StoredFileID:              storedFilex.ID,
		OriginalFilename:          meta.originalFilename,
		StorageFilenameWithoutExt: meta.storageFilenameWithoutExt,
		StorageFilename:           meta.storageFilename,
		TemporaryStoragePath:      meta.temporaryStoragePath,
		TemporaryStorageFilename:  meta.storageFilename,
	}

	return storedFilex, prepared, nil
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
func (qq *S3FileSystem) UploadPreparedFile(
	ctx ctxx.Context,
	fileToSave io.Reader,
	prepared *PreparedUpload,
) (*minio.UploadInfo, int64, error) {
	return qq.uploadPreparedFileWithParams(
		ctx,
		fileToSave,
		prepared.OriginalFilename,
		prepared.StorageFilenameWithoutExt,
		prepared.TemporaryStoragePath,
		prepared.StorageFilename,
	)
}

func (qq *S3FileSystem) FinalizePreparedUpload(
	ctx ctxx.Context,
	prepared *PreparedUpload,
	fileInfo *minio.UploadInfo,
	fileSize int64,
) error {
	ctxWithIncomplete := enttenantschema.WithUnfinishedUploads(ctx)
	storedFilex := ctx.TenantCtx().TTx.StoredFile.GetX(ctxWithIncomplete, prepared.StoredFileID)
	storedFilex = storedFilex.Update().
		SetSize(fileSize).
		SetSizeInStorage(fileInfo.Size).
		SetSha256(fileInfo.ChecksumSHA256).
		SetUploadSucceededAt(time.Now()).
		SaveX(ctxWithIncomplete)

	_, err := qq.UpdateMimeType(ctx, false, model.NewStoredFile(storedFilex))
	if err != nil {
		log.Println(err)
	}

	return nil
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
) (*minio.UploadInfo, string, int64, error) {
	originalFilename = filepath.Clean(originalFilename)
	if !filenamex.IsAllowed(originalFilename) {
		log.Println("invalid filename")
		return nil, "", 0, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid filename.")
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
		return nil, "", 0, err
	}

	if qq.bucketName == "" {
		return nil, "", 0, e.NewHTTPErrorf(http.StatusInternalServerError, "Bucket name is empty.")
	}

	// database contraints should verify that each file just exists once, but this just
	// provides additional safety against accidental overwriting a file, for example if the
	// same PublicID is generated twice
	_, err = qq.client.StatObject(ctx, qq.bucketName, objectName, minio.StatObjectOptions{})
	if err == nil {
		log.Printf("filename already exists, should never happen, objectName was %s", objectName)
		return nil, "", 0, e.NewHTTPErrorf(http.StatusInternalServerError, "Filename already exists.")
	}

	// can maybe be further optimized by using a pipe for each step, but separating
	// the slow network write from encryption and compression should already bring the biggest
	// benefit, see also:
	// https://chatgpt.com/c/67f29855-e4ac-8000-9305-a5b63137e799
	pipeReader, pipeWriter := io.Pipe()
	defer func() {
		err := pipeReader.Close()
		if err != nil {
			log.Println(err)
		}
	}()
	var fileSize int64
	go func() {
		defer recoverx.Recover("saveFile")

		var gzipInputWriter io.Writer
		gzipInputWriter = pipeWriter
		var encryptorx io.WriteCloser // necessary outside condition to close in correct order

		if !qq.disableFileEncryption {
			// gzip first, then encrypt because encryption randomizes data and is less efficient to
			// compress than the file directly
			encryptorx, err = age.Encrypt(pipeWriter, x25519Identity.Recipient())
			if err != nil {
				log.Println(err)

				err = encryptorx.Close()
				if err != nil {
					log.Println(err)
				}
				erry := pipeWriter.CloseWithError(err)
				if erry != nil {
					log.Println(erry)
				}

				return
			}
			gzipInputWriter = encryptorx
		}

		gzipWriter := gzip.NewWriter(gzipInputWriter)

		fileSize, err = io.Copy(gzipWriter, fileToSave)
		if err != nil {
			log.Println(err)

			// IMPORTANT
			// order is important; pipeWriter last, not sure about gzipWriter and encryptor...
			err = gzipWriter.Close()
			if err != nil {
				log.Println(err)
			}
			if !qq.disableFileEncryption {
				err = encryptorx.Close()
				if err != nil {
					log.Println(err)
				}
			}
			erry := pipeWriter.CloseWithError(err)
			if erry != nil {
				log.Println(erry)
			}

			return
		}

		// IMPORTANT
		// order is important; pipeWriter last, not sure about gzipWriter and encryptor...
		err = gzipWriter.Close()
		if err != nil {
			log.Println(err)
		}
		if !qq.disableFileEncryption {
			err = encryptorx.Close()
			if err != nil {
				log.Println(err)
			}
		}
		err = pipeWriter.Close()
		if err != nil {
			log.Println(err)
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
		Progress: &progressWriter{},

		// reduces memory usage significantly
		NumThreads:            2,
		PartSize:              8 * 1024 * 1024, // default is 16 MB with 4 workers, 5MB is minimum, reduces memory usage
		ConcurrentStreamParts: true,

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
		log.Println(err)
		return nil, "", 0, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not save file.")
	}

	// log.Println("debug: 002b")

	// TODO verify checksum?

	return &fileInfo, storageFilename, fileSize, nil
}

type progressWriter struct {
	total int64
}

func (pw *progressWriter) Read(p []byte) (n int, err error) {
	pw.total += int64(len(p))
	// log.Printf("Uploaded %d bytes so far\n", pw.total)
	return len(p), nil
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
	originalFilename = filepath.Clean(originalFilename)
	if !filenamex.IsAllowed(originalFilename) {
		log.Println("invalid filename")
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid filename.")
	}

	storageFilenameWithoutExt := fmt.Sprintf("%s-%06d", uploadToken, fileIndex)
	storagePrefix := pathx.S3TemporaryAccountStoragePrefix(ctx.MainCtx().Account.PublicID.String())
	storageFilename := qq.storageFilename(originalFilename, storageFilenameWithoutExt)

	temporaryFile := mainTx.TemporaryFile.Create().
		SetOwner(ctx.MainCtx().Account).
		SetFilename(originalFilename).
		SetSizeInStorage(0).
		SetStorageType(storagetype.S3).
		SetBucketName(qq.bucketName).
		SetStoragePath(storagePrefix).
		SetStorageFilename(storageFilename).
		SetUploadToken(uploadToken).
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
) (*minio.UploadInfo, int64, error) {
	return qq.uploadPreparedFileWithParams(
		ctx,
		fileToSave,
		prepared.OriginalFilename,
		prepared.StorageFilenameWithoutExt,
		prepared.StoragePath,
		prepared.StorageFilename,
	)
}

func (qq *S3FileSystem) uploadPreparedFileWithParams(
	ctx ctxx.Context,
	fileToSave io.Reader,
	originalFilename string,
	storageFilenameWithoutExt string,
	storagePath string,
	expectedStorageFilename string,
) (*minio.UploadInfo, int64, error) {
	x25519Identity, err := qq.x25519Identity(ctx, storagePath)
	if err != nil {
		log.Println(err)
		return nil, 0, err
	}

	fileInfo, storageFilename, fileSize, err := qq.saveFile(
		ctx,
		x25519Identity,
		fileToSave,
		originalFilename,
		storageFilenameWithoutExt,
		storagePath,
	)
	if err != nil {
		log.Println(err)
		return nil, 0, err
	}

	if storageFilename != expectedStorageFilename {
		log.Println("storage filename mismatch", storageFilename, expectedStorageFilename)
		return nil, 0, e.NewHTTPErrorf(http.StatusInternalServerError, "Storage filename mismatch.")
	}

	return fileInfo, fileSize, nil
}

func (qq *S3FileSystem) FinalizePreparedTemporaryAccountUpload(
	ctx ctxx.Context,
	mainTx *entmain.Tx,
	prepared *PreparedAccountUpload,
	fileInfo *minio.UploadInfo,
	fileSize int64,
) error {
	ctxWithIncomplete := entmainschema.WithUnfinishedUploads(ctx)
	mainTx.TemporaryFile.
		UpdateOneID(prepared.TemporaryFileID).
		SetSize(fileSize).
		SetSizeInStorage(fileInfo.Size).
		SetSha256(fileInfo.ChecksumSHA256).
		SetUploadSucceededAt(time.Now()).
		SaveX(ctxWithIncomplete)

	return nil
}

// Prepares persistence of temporary account file. Doesn't move the file itself, but
// prepares it to get moved by scheduler.
func (qq *S3FileSystem) PreparePersistingTemporaryAccountFile(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	parentDirFileID int64,
	isInInbox bool,
) (*enttenant.File, error) {
	// create db entries
	filex := ctx.TenantCtx().TTx.File.Create().
		SetName(tmpFile.Filename). // TODO okay?
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		SetParentID(parentDirFileID).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		SetIsInInbox(isInInbox).
		SaveX(ctx)

	finalStoragePath := pathx.S3StoragePrefix(ctx.TenantCtx().TenantID)

	fileExtension := filepath.Ext(tmpFile.Filename)
	if fileExtension == "" {
		// nothing because files don't need an extension...
	}

	// don't use public id of model.File because a file has multiple versions
	// and thus it breaks if another version is added
	storedFilePublicID := util.NewPublicID()
	storageFilename := storedFilePublicID + fileExtension + ".gz.age"

	storedFilex := ctx.TenantCtx().TTx.StoredFile.Create().
		// SetPublicID(entx.NewCIText(storedFilePublicID)).
		SetFilename(tmpFile.Filename).
		SetSize(tmpFile.Size).                   // fileInfo.Size is gzipped size
		SetSizeInStorage(tmpFile.SizeInStorage). // gzipped size
		SetStorageType(storagetype.S3).
		SetBucketName(qq.bucketName).
		SetStoragePath(finalStoragePath).
		SetStorageFilename(storageFilename).
		// temporary because it gets moved by scheduler to prevent orphan files in object storage
		// if transaction fails
		SetTemporaryStoragePath(tmpFile.StoragePath).
		SetTemporaryStorageFilename(tmpFile.StorageFilename).
		SetSha256(tmpFile.Sha256).
		SaveX(ctx)

	if err := qq.addFileVersion(ctx, filex, storedFilex); err != nil {
		log.Println(err)
		return nil, err
	}

	// TODO not very clean; only in case contentType is empty
	_, err := qq.UpdateMimeType(ctx, false, model.NewStoredFile(storedFilex))
	if err != nil {
		log.Println(err)
		// not critical
	}

	tmpFile.Update().
		SetConvertedToStoredFileAt(time.Now()).
		ClearExpiresAt().
		ExecX(ctx)

	return filex, nil
}

func (qq *S3FileSystem) PersistTemporaryTenantFile(
	ctx context.Context,
	tenantX25519Identity *age.X25519Identity,
	filex *enttenant.StoredFile,
) error {
	filem := model.NewStoredFile(filex)

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

	// check if dest file exists, likely indicates that file was already moved, but writing to database failed
	_, err = qq.client.StatObject(ctx, qq.bucketName, destObjectName, minio.StatObjectOptions{})
	if err == nil {
		log.Printf("dest file already exists, skipping, needs manual cleanup, tmpObjectName: %s, destObjectName: %s", tmpObjectName, destObjectName)
		return err
	}
	minioErr := minio.ToErrorResponse(err)
	if minioErr.Code != "NoSuchKey" { // TODO can this be made more type safe?
		log.Println(err, "may need manual cleanup")
		return err
	}
	// file doesn't exists

	if strings.HasPrefix(destObjectName, pathx.S3TenantPrefix()) && strings.HasPrefix(tmpObjectName, pathx.S3TenantPrefix()) {
		_, err = qq.client.CopyObject(ctx, minio.CopyDestOptions{
			Bucket: qq.bucketName,
			Object: destObjectName,
			// https://min.io/docs/minio/linux/administration/object-management/object-retention.html#minio-object-locking-retention-modes
			// legal hold is complementary to retention; if legal hold is set for governance locked objects, mutation
			// is prevented even if the user has priviliges to bypass retention
			// TODO LegalHold:       minio.LegalHoldEnabled,
			// Mode:            "", // retention // FIXME depending on customer?
			// RetainUntilDate: time.Time{},
			// Size:     0,   // must be set for progress bar
			Progress: nil, // can be used for progress bar
		}, minio.CopySrcOptions{
			Bucket: qq.bucketName,
			Object: tmpObjectName,
		})
		if err != nil {
			log.Println(err, "may need manual cleanup")
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

		// fileSize should be identical because file didn't change
		// TODO verify that the same?
		fileInfo, _, _, err := qq.saveFile(
			ctx,
			tenantX25519Identity,
			tmpFile,
			filem.Data.Filename,
			storageFilenameWithoutExt,
			filem.Data.StoragePath,
		)
		if err != nil {
			log.Println(err)
			return err
		}

		filex = filex.Update().
			SetSizeInStorage(fileInfo.Size).
			SetSha256(fileInfo.ChecksumSHA256).
			SaveX(ctx)
	} else {
		err = e.NewHTTPErrorf(http.StatusInternalServerError, "Could not copy temporary file.")
		log.Println(err, "may need manual cleanup or missing configuration")
		return err
	}

	err = filex.Update().SetCopiedToFinalDestinationAt(time.Now()).Exec(ctx)
	if err != nil {
		log.Println(err, "; may need manual cleanup")
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
func (qq *S3FileSystem) UpdateMimeType(ctx ctxx.Context, force bool, filex *model.StoredFile) (string, error) {
	if filex.Data.MimeType != "" && !force {
		return filex.Data.MimeType, nil
	}

	obj, err := qq.OpenFile(ctx, filex)
	if err != nil {
		log.Println(err)
		return "", e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}
	defer obj.Close()

	buf := make([]byte, 512)
	n, err := obj.Read(buf)
	if err != nil && err != io.EOF {
		log.Println(err)
		return "", e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}

	// seems to be necessary to remove zero values which can cause false detection; not verified
	// by me, see:
	// https://gist.github.com/rayrutjes/db9b9ea8e02255d62ce2?permalink_comment_id=3418419#gistcomment-3418419
	buf = buf[:n]

	mimeType := http.DetectContentType(buf)
	filex.Data = filex.Data.Update().SetMimeType(mimeType).SaveX(ctx)

	/*  TODO necessary if closed directly? pipeReader doesn't implement Seeker
	// after probing mimetype
	_, err = obj.Seek(0, 0)
	if err != nil {
		log.Println(err)
		return "", e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}
	*/

	return mimeType, nil
}
