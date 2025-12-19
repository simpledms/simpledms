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

	"filippo.io/age"
	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/minio/minio-go/v7"

	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/encryptor"
	"github.com/simpledms/simpledms/app/simpledms/entmain"
	"github.com/simpledms/simpledms/app/simpledms/enttenant"
	"github.com/simpledms/simpledms/app/simpledms/enttenant/file"
	"github.com/simpledms/simpledms/app/simpledms/model"
	"github.com/simpledms/simpledms/app/simpledms/model/common/storagetype"
	"github.com/simpledms/simpledms/app/simpledms/pathx"
	"github.com/simpledms/simpledms/util"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
	"github.com/simpledms/simpledms/util/recoverx"
)

type S3FileSystem struct {
	*FileSystem
	client     *minio.Client
	bucketName string
}

func NewS3FileSystem(client *minio.Client, bucketName string, fileSystem *FileSystem) *S3FileSystem {
	return &S3FileSystem{
		FileSystem: fileSystem,
		client:     client,
		bucketName: bucketName,
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

		gzipReader, err := gzip.NewReader(decryptor)
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

// IMPORTANT
// very similar code in FileSystem
//
// caller has to close fileToSave
func (qq *S3FileSystem) SaveFile(
	ctx ctxx.Context,
	// fileToSave multipart.File,
	fileToSave io.Reader,
	filename string,
	isInInbox bool,
	parentDirFileID int64,
) (*enttenant.File, error) {
	// TODO check if bucket exists

	if ctx.SpaceCtx().Space.IsFolderMode {
		fileExists := ctx.SpaceCtx().Space.QueryFiles().
			Where(file.Name(filename), file.ParentID(parentDirFileID), file.IsInInbox(isInInbox)).
			ExistX(ctx)
		if fileExists {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File already exists.")
		}
	}

	// log.Println("debug: 001a")

	// FIXME handle transaction or let indexer handle such situations?
	filex := ctx.TenantCtx().TTx.File.Create().
		SetName(filename).
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		// TODO take mode time from uploadedFile if possible at all
		// SetModifiedAt(fileInfo.ModTime()). // TODO necessary?
		SetParentID(parentDirFileID).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		SetIsInInbox(isInInbox).
		// AddSpaceIDs(ctx.SpaceCtx().Space.ID).
		// AddVersions(fileVersionx).
		SaveX(ctx)

	// log.Println("debug: 002a")

	tmpStoragePrefix := pathx.S3TemporaryStoragePrefix(ctx.TenantCtx().TenantID) // ctx.TenantCtx().S3StoragePrefix
	if tmpStoragePrefix == "" {
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Storage path is empty.")
	}
	finalStoragePrefix := pathx.S3StoragePrefix(ctx.TenantCtx().TenantID)

	x25519Identity, err := qq.x25519Identity(ctx, tmpStoragePrefix)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	// don't use public id of model.File because a file has multiple versions
	// and thus it breaks if another version is added
	storedFilePublicID := util.NewPublicID()
	fileInfo, storageFilename, fileSize, err := qq.saveFile(
		ctx,
		x25519Identity,
		fileToSave,
		filename,
		storedFilePublicID,
		tmpStoragePrefix,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// log.Println("debug: 003a")

	storedFilex := ctx.TenantCtx().TTx.StoredFile.Create().
		// SetPublicID(entx.NewCIText(storedFilePublicID)).
		SetFilename(filename).
		SetSize(fileSize).               // fileInfo.Size is gzipped size
		SetSizeInStorage(fileInfo.Size). // gzipped size
		SetStorageType(storagetype.S3).
		SetBucketName(qq.bucketName).
		SetStoragePath(finalStoragePrefix).
		SetStorageFilename(storageFilename).
		// temporary because it gets moved by scheduler to prevent orphan files in object storage
		// if transaction fails
		SetTemporaryStoragePath(tmpStoragePrefix).
		SetTemporaryStorageFilename(storageFilename).
		SetSha256(fileInfo.ChecksumSHA256).
		// SetMimeType(contentType).
		SaveX(ctx)
	filex.Update().
		AddVersionIDs(storedFilex.ID).
		SaveX(ctx)

	// log.Println("debug: 004a")

	// TODO not very clean; only in case contentType is empty
	_, err = qq.UpdateMimeType(ctx, false, model.NewStoredFile(storedFilex))
	if err != nil {
		log.Println(err)
		// not critical
	}

	// log.Println("debug: 005a")

	return filex, nil
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
		// gzip first, then encrypt because encryption randomizes data and is less efficient to
		// compress than the file directly
		encryptorx, err := age.Encrypt(pipeWriter, x25519Identity.Recipient())
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

		gzipWriter := gzip.NewWriter(encryptorx)

		fileSize, err = io.Copy(gzipWriter, fileToSave)
		if err != nil {
			log.Println(err)

			// IMPORTANT
			// order is important; pipeWriter last, not sure about gzipWriter and encryptor...
			err = gzipWriter.Close()
			if err != nil {
				log.Println(err)
			}
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

		// IMPORTANT
		// order is important; pipeWriter last, not sure about gzipWriter and encryptor...
		err = gzipWriter.Close()
		if err != nil {
			log.Println(err)
		}
		err = encryptorx.Close()
		if err != nil {
			log.Println(err)
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

func (qq *S3FileSystem) SaveTemporaryFileToAccount(
	ctx ctxx.Context,
	fileToSave io.Reader,
	originalFilename string,
	uploadToken string,
	fileIndex int,
	expiresAt time.Time,
) (*entmain.TemporaryFile, error) {
	storageFilenameWithoutExt := fmt.Sprintf("%s-%06d", uploadToken, fileIndex)
	storagePrefix := pathx.S3TemporaryAccountStoragePrefix(ctx.MainCtx().Account.PublicID.String())

	// outside of go func() to return quickly
	x25519Identity, err := qq.x25519Identity(ctx, storagePrefix)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	fileInfo, storageFilename, fileSize, err := qq.saveFile(
		ctx,
		x25519Identity,
		fileToSave,
		originalFilename,
		storageFilenameWithoutExt,
		storagePrefix,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	temporaryFile := ctx.MainCtx().MainTx.TemporaryFile.Create().
		SetOwner(ctx.MainCtx().Account).
		SetFilename(originalFilename).
		SetSize(fileSize).               // fileInfo.Size is gzipped size
		SetSizeInStorage(fileInfo.Size). // gzipped size
		SetStorageType(storagetype.S3).
		SetBucketName(qq.bucketName).
		SetStoragePath(storagePrefix).
		SetStorageFilename(storageFilename).
		SetSha256(fileInfo.ChecksumSHA256).
		SetUploadToken(uploadToken).
		SetExpiresAt(expiresAt).
		// SetMimeType(contentType).
		SaveX(ctx)

	return temporaryFile, nil
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

	filex.Update().
		AddVersionIDs(storedFilex.ID).
		SaveX(ctx)

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
