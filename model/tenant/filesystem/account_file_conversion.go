package filesystem

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/minio/minio-go/v7"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/encryptor"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
	"github.com/simpledms/simpledms/model/main/tenantaccess"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/pathx"
	"github.com/simpledms/simpledms/util"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/txx"
)

const accountConversionTakeoverAfter = time.Hour

const temporaryAccountFileExpiry = 15 * time.Minute

// AccountFileConversion persists temporary account uploads, for example files uploaded before a
// destination was chosen, into a tenant. A conversion is claimed, heartbeated while it runs, and
// taken over by another worker if it becomes stale.
type AccountFileConversion struct {
	fileSystem *S3FileSystem
}

func NewAccountFileConversion(fileSystem *S3FileSystem) *AccountFileConversion {
	return &AccountFileConversion{
		fileSystem: fileSystem,
	}
}

func (qq *AccountFileConversion) Persist(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	parentDirFileID int64,
	isInInbox bool,
) (*enttenant.File, error) {
	err := qq.fileSystem.EnsureUploadSizeLimit(ctx, tmpFile.Size)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = qq.fileSystem.EnsureTenantStorageLimit(ctx, tmpFile.Size)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return qq.persistTemporaryAccountFile(ctx, tmpFile, parentDirFileID, isInInbox)
}

func (qq *AccountFileConversion) persistTemporaryAccountFile(
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
		if err := qq.releaseTemporaryAccountConversionClaim(ctx, tmpFile, claimToken); err != nil {
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
	tmpFile, err = qq.verifyOrEstablishTemporaryFileSHA256(ctx, tmpFile, claimToken, accountObjectName)
	if err != nil {
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
	plaintext, err := qq.fileSystem.UnsafeOpenFile(
		ctx,
		mainX25519Identity,
		storedfilemodel.NewStoredFile(accountFile),
	)
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
	fileInfo, storageFilename, result, err := qq.fileSystem.saveFile(
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
	if result.fileSize != tmpFile.Size ||
		(tmpFile.ContentSha256 != "" && result.contentSHA256 != tmpFile.ContentSha256) {
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

func (qq *AccountFileConversion) claimTemporaryAccountConversion(
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
			temporaryfile.Or(
				temporaryfile.PersistenceTenantIDIsNil(),
				temporaryfile.PersistenceTenantID(ctx.TenantCtx().Tenant.ID),
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

func (qq *AccountFileConversion) nilableSuccessfulAccountConversion(
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

func (qq *AccountFileConversion) nilableUnlinkedSuccessfulAccountConversion(
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

	filex, err := txx.WithFreshAuthorizedTenantWriteSpaceTx(
		ctx.SpaceCtx(),
		func(writeSpaceCtx *ctxx.SpaceContext) (*enttenant.File, error) {
			writeCtxWithIncomplete := tenantprivacy.DecisionContext(
				enttenantschema.WithUnfinishedUploads(writeSpaceCtx),
				tenantprivacy.Allow,
			)
			storedFilex, err = writeSpaceCtx.TTx.StoredFile.Get(writeCtxWithIncomplete, storedFilex.ID)
			if err != nil {
				return nil, err
			}
			filex, err := writeSpaceCtx.TTx.File.Create().
				SetName(tmpFile.Filename).
				SetSource(tmpFile.Source).
				SetIsDirectory(false).
				SetIndexedAt(time.Now()).
				SetParentID(parentDirFileID).
				SetSpaceID(writeSpaceCtx.Space.ID).
				SetIsInInbox(isInInbox).
				Save(writeSpaceCtx)
			if err != nil {
				return nil, err
			}
			if err := qq.fileSystem.addFileVersion(writeSpaceCtx, filex, storedFilex); err != nil {
				return nil, err
			}
			return filex, nil
		},
	)
	if err != nil {
		return nil, false, err
	}

	if err := qq.markTemporaryAccountConversionDone(ctx, tmpFile.ID); err != nil {
		return nil, false, err
	}
	return filex, true, nil
}

func (qq *AccountFileConversion) prepareClaimedAccountConversion(
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

func (qq *AccountFileConversion) createClaimedAccountConversion(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	claimToken string,
) (*enttenant.StoredFile, error) {
	finalStoragePath := pathx.S3StoragePrefix(ctx.TenantCtx().TenantID)
	temporaryStoragePath := pathx.S3TemporaryStoragePrefix(ctx.TenantCtx().TenantID)
	storageFilenameWithoutExt := util.NewPublicID()
	storageFilename := qq.fileSystem.storageFilename(tmpFile.Filename, storageFilenameWithoutExt)
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
		SetBucketName(qq.fileSystem.bucketName).
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

func (qq *AccountFileConversion) finalizeClaimedAccountConversion(
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
	hasAccess, err := tenantaccess.NewTenantAccessService().HasActiveTenantAssignmentForTenant(
		ctx,
		mainTx,
		ctx.MainCtx().Account.ID,
		ctx.TenantCtx().Tenant.ID,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if !hasAccess {
		return nil, e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this organization.")
	}

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
		SetContentSha256(result.contentSHA256).
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

func (qq *AccountFileConversion) finalizeClaimedAccountConversionInTenant(
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

	writeSpaceCtx, err := qq.accountConversionSpaceContext(ctx, tenantTx)
	if err != nil {
		return nil, err
	}
	ctxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(writeSpaceCtx),
		tenantprivacy.Allow,
	)
	if err := qq.fileSystem.EnsureTenantStorageLimit(writeSpaceCtx, result.fileSize); err != nil {
		return nil, err
	}

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
			SetNillableMimeType(nilableNonEmptyString(claimedTmpFile.MimeType)).
			SetUploadSucceededAt(time.Now()).
			Save(ctxWithIncomplete)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	}

	filex, err := tenantTx.File.Create().
		SetName(claimedTmpFile.Filename).
		SetSource(claimedTmpFile.Source).
		SetIsDirectory(false).
		SetIndexedAt(time.Now()).
		SetParentID(parentDirFileID).
		SetSpaceID(writeSpaceCtx.Space.ID).
		SetIsInInbox(isInInbox).
		Save(writeSpaceCtx)
	if err != nil {
		return nil, err
	}

	if err := qq.fileSystem.addFileVersion(writeSpaceCtx, filex, storedFilex); err != nil {
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

func (qq *AccountFileConversion) accountConversionSpaceContext(
	ctx ctxx.Context, tenantTx *enttenant.Tx,
) (*ctxx.SpaceContext, error) {
	userx, err := tenantTx.User.Query().Where(
		user.AccountID(ctx.MainCtx().Account.ID),
		user.DeletedAtIsNil(),
	).Only(ctx)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(
				http.StatusForbidden, "You are not allowed to access this organization.",
			)
		}
		return nil, err
	}
	writeTenantCtx := ctxx.NewTenantContextWithUser(
		ctx.MainCtx(),
		tenantTx,
		ctx.TenantCtx().Tenant,
		userx,
		false,
	)
	writeSpace, err := tenantTx.Space.Get(writeTenantCtx, ctx.SpaceCtx().Space.ID)
	if err != nil {
		return nil, err
	}
	return ctxx.NewSpaceContext(writeTenantCtx, writeSpace), nil
}

func (qq *AccountFileConversion) markTemporaryAccountConversionDone(ctx ctxx.Context, temporaryFileID int64) error {
	if !ctx.MainCtx().IsReadOnlyTx() {
		return ctx.MainCtx().MainTx.TemporaryFile.UpdateOneID(temporaryFileID).
			SetConvertedToStoredFileAt(time.Now()).
			ClearPersistenceClaimToken().
			ClearPersistenceTenantID().
			ClearPersistenceLastProgressAt().
			ClearExpiresAt().
			Exec(ctx)
	}
	return ctx.MainCtx().UnsafeMainDB().ReadWriteConn.TemporaryFile.UpdateOneID(temporaryFileID).
		SetConvertedToStoredFileAt(time.Now()).
		ClearPersistenceClaimToken().
		ClearPersistenceTenantID().
		ClearPersistenceLastProgressAt().
		ClearExpiresAt().
		Exec(ctx)
}

func (qq *AccountFileConversion) accountConversionHeartbeat(
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

func (qq *AccountFileConversion) cleanupClaimedAccountConversion(
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
	if err := qq.fileSystem.RemoveTemporaryObject(
		ctx,
		current.TemporaryStoragePath,
		current.TemporaryStorageFilename,
	); err != nil {
		log.Println(err)
		return err
	}
	return tenantDB.ReadWriteConn.StoredFile.DeleteOneID(current.ID).Exec(ctxWithIncomplete)
}

func (qq *AccountFileConversion) releaseTemporaryAccountConversionClaim(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	claimToken string,
) error {
	tenantDB, ok := ctx.TenantCtx().UnsafeTenantDB()
	if !ok {
		return e.NewHTTPErrorf(http.StatusInternalServerError, tenantDatabaseNotFoundMessage)
	}
	ctxWithIncomplete := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(ctx),
		tenantprivacy.Allow,
	)
	hasSuccessfulResult, err := tenantDB.ReadOnlyConn.StoredFile.Query().Where(
		storedfile.SourceTemporaryFilePublicID(entx.NewCIText(tmpFile.PublicID.String())),
		storedfile.SourceConversionClaimToken(claimToken),
		storedfile.UploadSucceededAtNotNil(),
	).Exist(ctxWithIncomplete)
	if err != nil {
		return err
	}
	if hasSuccessfulResult {
		return nil
	}

	err = ctx.MainCtx().UnsafeMainDB().ReadWriteConn.TemporaryFile.UpdateOneID(tmpFile.ID).
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

func (qq *AccountFileConversion) verifyOrEstablishTemporaryFileSHA256(
	ctx ctxx.Context,
	tmpFile *entmain.TemporaryFile,
	claimToken string,
	objectName string,
) (*entmain.TemporaryFile, error) {
	if tmpFile.Sha256 != "" {
		err := qq.fileSystem.verifyObject(ctx, objectName, tmpFile.SizeInStorage, tmpFile.Sha256)
		if err != nil {
			return nil, err
		}
		return tmpFile, nil
	}

	size, storageSHA256, err := qq.fileSystem.objectSHA256(ctx, objectName)
	if err != nil {
		return nil, err
	}
	if size != tmpFile.SizeInStorage {
		return nil, fmt.Errorf("stored object size mismatch: got %d, want %d", size, tmpFile.SizeInStorage)
	}
	return ctx.MainCtx().UnsafeMainDB().ReadWriteConn.TemporaryFile.UpdateOneID(tmpFile.ID).
		Where(temporaryfile.PersistenceClaimToken(claimToken)).
		SetSha256(storageSHA256).
		Save(ctx)
}
