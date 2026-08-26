package scheduler

import (
	"context"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/privacy"
	"github.com/minio/minio-go/v7"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/enttenant"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	enttenantwebdavresource "github.com/simpledms/simpledms/db/enttenant/webdavresource"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	temporaryfilemodel "github.com/simpledms/simpledms/model/main/temporaryfile"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	webdavresourcemodel "github.com/simpledms/simpledms/model/tenant/webdavresource"
	"github.com/simpledms/simpledms/pathx"
)

const (
	orphanTemporaryObjectGrace        = 24 * time.Hour
	orphanTemporaryObjectScanInterval = time.Hour
	temporaryAccountFileExpiry        = 15 * time.Minute
)

func (qq *Scheduler) processFiles() {
	defer func() {
		// tested and works
		if r := recover(); r != nil {
			log.Printf("%v: %s", r, debug.Stack())
			log.Println("trying to recover")

			// TODO what is a good interval
			time.Sleep(1 * time.Minute)

			// tested and works, automatically restarts loop
			qq.processFiles()
		}
	}()

	for {
		ctx := context.Background()
		ctx = privacy.DecisionContext(ctx, privacy.Allow)

		qq.copyTempFilesToFinalDest(ctx)
		qq.deleteProcessedTempFiles(ctx)
		qq.deleteTempAccountFiles(ctx)
		qq.recoverStaleAccountConversions(ctx)
		qq.recoverStaleTenantUploads(ctx)
		qq.releaseStaleWebDAVAliases(ctx)
		if qq.shouldScanOrphanTemporaryObjects(time.Now()) {
			qq.deleteOrphanTemporaryObjects(ctx)
		}

		// TODO is this to short? how expensive is this in larger instances?
		time.Sleep(5 * time.Second)
	}
}

func (qq *Scheduler) recoverStaleTenantUploads(ctx context.Context) {
	ctxWithIncomplete := enttenantschema.WithUnfinishedUploads(ctx)
	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		threshold := time.Now().Add(-time.Hour)
		resources := tenantDB.ReadOnlyConn.WebDAVResource.Query().
			Where(
				enttenantwebdavresource.StateIn(
					webdavresourcemodel.Uploading,
					webdavresourcemodel.CleanupPending,
				),
				enttenantwebdavresource.LastProgressAtLT(threshold),
			).
			Order(enttenantwebdavresource.ByLastProgressAt(sql.OrderAsc()), enttenantwebdavresource.ByID()).
			Limit(defaultSchedulerBatchSize).
			AllX(ctx)
		for _, resource := range resources {
			qq.cleanupStaleWebDAVResource(ctx, tenantDB, resource)
		}

		files := tenantDB.ReadOnlyConn.StoredFile.Query().
			Where(
				storedfile.UploadSucceededAtIsNil(),
				storedfile.UploadLastProgressAtLT(threshold),
			).
			Order(storedfile.ByUploadLastProgressAt(sql.OrderAsc()), storedfile.ByID()).
			Limit(defaultSchedulerBatchSize).
			AllX(ctxWithIncomplete)
		for _, filex := range files {
			qq.cleanupStaleStoredFile(ctxWithIncomplete, tenantDB, filex, threshold)
		}
		return true
	})
}

func (qq *Scheduler) cleanupStaleWebDAVResource(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	resource *enttenant.WebDAVResource,
) {
	ctxWithIncomplete := enttenantschema.WithUnfinishedUploads(ctx)
	if resource.State == webdavresourcemodel.Uploading {
		claimed, err := tenantDB.ReadWriteConn.WebDAVResource.Update().
			Where(
				enttenantwebdavresource.ID(resource.ID),
				enttenantwebdavresource.StateEQ(webdavresourcemodel.Uploading),
				enttenantwebdavresource.LastProgressAtLT(time.Now().Add(-time.Hour)),
			).
			SetState(webdavresourcemodel.CleanupPending).
			SetLastProgressAt(time.Now()).
			Save(ctx)
		if err != nil {
			log.Println(err)
			return
		}
		if claimed != 1 {
			return
		}
	}
	if resource.StoredFileID != nil {
		storedFilex, err := tenantDB.ReadOnlyConn.StoredFile.Get(ctxWithIncomplete, *resource.StoredFileID)
		if err == nil && storedFilex.UploadSucceededAt == nil {
			if !qq.cleanupStaleStoredFile(ctxWithIncomplete, tenantDB, storedFilex, time.Now().Add(-time.Hour)) {
				_ = tenantDB.ReadWriteConn.WebDAVResource.UpdateOneID(resource.ID).
					SetState(webdavresourcemodel.CleanupPending).
					Exec(ctx)
				return
			}
		} else if err != nil && !enttenant.IsNotFound(err) {
			log.Println(err)
			return
		}
	}
	if err := tenantDB.ReadWriteConn.WebDAVResource.DeleteOneID(resource.ID).Exec(ctx); err != nil && !enttenant.IsNotFound(err) {
		log.Println(err)
	}
}

func (qq *Scheduler) cleanupStaleStoredFile(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	filex *enttenant.StoredFile,
	threshold time.Time,
) bool {
	ctx = enttenantschema.WithUnfinishedUploads(ctx)
	if filex.UploadFailedAt == nil {
		claimed, err := tenantDB.ReadWriteConn.StoredFile.Update().
			Where(
				storedfile.ID(filex.ID),
				storedfile.UploadSucceededAtIsNil(),
				storedfile.UploadFailedAtIsNil(),
				storedfile.UploadLastProgressAtLT(threshold),
			).
			SetUploadFailedAt(time.Now()).
			SetUploadLastProgressAt(time.Now()).
			Save(ctx)
		if err != nil {
			log.Println(err)
			return false
		}
		if claimed != 1 {
			return false
		}
	}
	filem := storedfilemodel.NewStoredFile(filex)
	objectName, err := filem.UnsafeTempObjectNameWithPrefix()
	if err != nil {
		log.Println(err)
		return false
	}
	if qq.s3Client != nil && objectName != "" {
		if err := qq.s3Client.RemoveObject(ctx, qq.bucketName, objectName, minio.RemoveObjectOptions{}); err != nil {
			minioErr := minio.ToErrorResponse(err)
			if minioErr.Code != "NoSuchKey" {
				log.Println(err)
				return false
			}
		}
	}
	return true
}

func (qq *Scheduler) releaseStaleWebDAVAliases(ctx context.Context) {
	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		resources := tenantDB.ReadOnlyConn.WebDAVResource.Query().
			Where(enttenantwebdavresource.StateEQ(webdavresourcemodel.Active)).
			Order(enttenantwebdavresource.ByID()).
			Limit(defaultSchedulerBatchSize).
			AllX(ctx)
		for _, resource := range resources {
			if resource.FileID == nil {
				qq.deleteWebDAVResource(ctx, tenantDB, resource.ID)
				continue
			}
			filex, err := tenantDB.ReadOnlyConn.File.Get(ctx, *resource.FileID)
			if err != nil {
				if enttenant.IsNotFound(err) {
					qq.deleteWebDAVResource(ctx, tenantDB, resource.ID)
				} else {
					log.Println(err)
				}
				continue
			}
			if !filex.DeletedAt.IsZero() || !filex.IsInInbox || filex.SpaceID != resource.SpaceID || filex.IsDirectory {
				qq.deleteWebDAVResource(ctx, tenantDB, resource.ID)
			}
		}
		return true
	})
}

func (qq *Scheduler) deleteOrphanTemporaryObjects(ctx context.Context) {
	if qq.s3Client == nil || qq.mainDB == nil || qq.tenantDBs == nil || qq.bucketName == "" {
		return
	}

	liveObjects := qq.liveTemporaryObjectNames(ctx)
	prefixes := qq.knownTemporaryObjectPrefixes(ctx)
	now := time.Now()
	remaining := defaultSchedulerBatchSize
	for _, prefix := range prefixes {
		if remaining <= 0 {
			return
		}
		remaining -= qq.scanOrphanTemporaryPrefix(ctx, prefix, liveObjects, now, remaining)
	}
}

func (qq *Scheduler) shouldScanOrphanTemporaryObjects(now time.Time) bool {
	if qq.lastOrphanObjectScan.IsZero() ||
		!now.Before(qq.lastOrphanObjectScan.Add(orphanTemporaryObjectScanInterval)) {
		qq.lastOrphanObjectScan = now
		return true
	}
	return false
}

func (qq *Scheduler) knownTemporaryObjectPrefixes(ctx context.Context) []string {
	prefixes := make([]string, 0)
	accounts := qq.mainDB.ReadOnlyConn.Account.Query().AllX(ctx)
	for _, accountx := range accounts {
		prefixes = append(prefixes, pathx.S3TemporaryAccountStoragePrefix(accountx.PublicID.String()))
	}

	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		tenantx, err := qq.mainDB.ReadOnlyConn.Tenant.Get(ctx, tenantID)
		if err != nil {
			if !entmain.IsNotFound(err) {
				log.Println(err)
			}
			return true
		}
		prefixes = append(prefixes, pathx.S3TemporaryStoragePrefix(tenantx.PublicID.String()))
		return true
	})

	return prefixes
}

func (qq *Scheduler) liveTemporaryObjectNames(ctx context.Context) map[string]struct{} {
	liveObjects := make(map[string]struct{})
	if qq.mainDB != nil {
		files := qq.mainDB.ReadOnlyConn.TemporaryFile.Query().
			Where(temporaryfile.DeletedAtIsNil(), temporaryfile.ConvertedToStoredFileAtIsNil()).
			AllX(ctx)
		for _, tmpFile := range files {
			objectName, err := temporaryfilemodel.NewTemporaryFile(tmpFile).ObjectNameWithPrefix()
			if err != nil {
				log.Println(err)
				continue
			}
			liveObjects[objectName] = struct{}{}
		}
	}

	if qq.tenantDBs != nil {
		ctxWithIncomplete := enttenantschema.WithUnfinishedUploads(ctx)
		qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
			files := tenantDB.ReadOnlyConn.StoredFile.Query().
				Where(storedfile.DeletedTemporaryFileAtIsNil()).
				AllX(ctxWithIncomplete)
			for _, filex := range files {
				objectName, err := storedfilemodel.NewStoredFile(filex).UnsafeTempObjectNameWithPrefix()
				if err != nil {
					log.Println(err)
					continue
				}
				liveObjects[objectName] = struct{}{}
			}
			return true
		})
	}

	return liveObjects
}

func (qq *Scheduler) scanOrphanTemporaryPrefix(
	ctx context.Context,
	prefix string,
	liveObjects map[string]struct{},
	now time.Time,
	limit int,
) int {
	if !isKnownTemporaryPrefix(prefix) || limit <= 0 {
		return 0
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	objects := qq.s3Client.ListObjects(ctx, qq.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix + "/",
		Recursive: true,
		MaxKeys:   limit,
	})

	scanned := 0
	for object := range objects {
		if scanned >= limit {
			cancel()
			break
		}
		scanned++
		qq.deleteOrphanTemporaryObject(ctx, object, prefix, liveObjects, now)
	}
	return scanned
}

func (qq *Scheduler) deleteOrphanTemporaryObject(
	ctx context.Context,
	object minio.ObjectInfo,
	prefix string,
	liveObjects map[string]struct{},
	now time.Time,
) {
	if object.Err != nil {
		log.Println(object.Err)
		return
	}
	if !objectNameUnderPrefix(object.Key, prefix) {
		return
	}
	if _, ok := liveObjects[object.Key]; ok {
		return
	}
	if object.LastModified.IsZero() || object.LastModified.After(now.Add(-orphanTemporaryObjectGrace)) {
		return
	}
	if err := qq.s3Client.RemoveObject(ctx, qq.bucketName, object.Key, minio.RemoveObjectOptions{}); err != nil {
		minioErr := minio.ToErrorResponse(err)
		if minioErr.Code != "NoSuchKey" {
			log.Println(err)
		}
	}
}

func isKnownTemporaryPrefix(prefix string) bool {
	return (strings.HasPrefix(prefix, pathx.S3AccountPrefix()+"/") ||
		strings.HasPrefix(prefix, pathx.S3TenantPrefix()+"/")) && strings.HasSuffix(prefix, "/tmp")
}

func objectNameUnderPrefix(objectName string, prefix string) bool {
	return objectName != "" && strings.HasPrefix(objectName, prefix+"/")
}

func (qq *Scheduler) deleteWebDAVResource(ctx context.Context, tenantDB *sqlx.TenantDB, id int64) {
	if err := tenantDB.ReadWriteConn.WebDAVResource.DeleteOneID(id).Exec(ctx); err != nil && !enttenant.IsNotFound(err) {
		log.Println(err)
	}
}

func (qq *Scheduler) recoverStaleAccountConversions(ctx context.Context) {
	if qq.mainDB == nil || qq.tenantDBs == nil {
		return
	}

	now := time.Now()
	for _, tmpFile := range qq.staleAccountConversions(ctx, now, defaultSchedulerBatchSize) {
		oldToken := ""
		if tmpFile.PersistenceClaimToken != nil {
			oldToken = *tmpFile.PersistenceClaimToken
		}
		if oldToken == "" || tmpFile.PersistenceTenantID == nil {
			continue
		}

		cleanupToken := "scheduler-cleanup-" + tmpFile.PublicID.String()
		staleBefore := now.Add(-time.Hour)
		updated, err := qq.mainDB.ReadWriteConn.TemporaryFile.UpdateOneID(tmpFile.ID).
			Where(
				temporaryfile.PersistenceClaimToken(oldToken),
				temporaryfile.PersistenceLastProgressAtLT(staleBefore),
			).
			SetPersistenceClaimToken(cleanupToken).
			SetPersistenceLastProgressAt(now).
			Save(ctx)
		if err != nil {
			if !entmain.IsNotFound(err) {
				log.Println(err)
			}
			continue
		}

		tenantDB, ok := qq.tenantDBs.Load(*updated.PersistenceTenantID)
		if !ok {
			qq.clearAccountConversionClaim(ctx, updated.ID, cleanupToken)
			continue
		}
		qq.recoverStaleAccountConversion(ctx, tenantDB, updated, oldToken, cleanupToken)
	}
}

func (qq *Scheduler) staleAccountConversions(
	ctx context.Context,
	now time.Time,
	maxFilesPerRun int,
) []*entmain.TemporaryFile {
	return qq.mainDB.ReadOnlyConn.TemporaryFile.Query().
		Where(
			temporaryfile.ConvertedToStoredFileAtIsNil(),
			temporaryfile.DeletedAtIsNil(),
			temporaryfile.PersistenceClaimTokenNotNil(),
			temporaryfile.PersistenceLastProgressAtLT(now.Add(-time.Hour)),
		).
		Order(temporaryfile.ByPersistenceLastProgressAt(sql.OrderAsc()), temporaryfile.ByID(sql.OrderAsc())).
		Limit(maxFilesPerRun).
		AllX(ctx)
}

func (qq *Scheduler) recoverStaleAccountConversion(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	tmpFile *entmain.TemporaryFile,
	oldToken string,
	cleanupToken string,
) {
	storedFilex, ok := qq.staleAccountConversionStoredFile(ctx, tenantDB, tmpFile, cleanupToken)
	if !ok {
		return
	}

	ctxWithIncomplete := enttenantschema.WithUnfinishedUploads(ctx)
	if storedFilex.UploadSucceededAt != nil || storedFilex.QueryFileVersions().ExistX(ctxWithIncomplete) {
		qq.completeRecoveredAccountConversion(ctx, tmpFile.ID, cleanupToken)
		return
	}

	if storedFilex.SourceConversionClaimToken == nil || *storedFilex.SourceConversionClaimToken != oldToken {
		qq.clearAccountConversionClaim(ctx, tmpFile.ID, cleanupToken)
		return
	}

	qq.cleanupIncompleteAccountConversion(
		ctx,
		ctxWithIncomplete,
		tenantDB,
		storedFilex,
		tmpFile.ID,
		cleanupToken,
	)
}

func (qq *Scheduler) staleAccountConversionStoredFile(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	tmpFile *entmain.TemporaryFile,
	cleanupToken string,
) (*enttenant.StoredFile, bool) {
	ctxWithIncomplete := enttenantschema.WithUnfinishedUploads(ctx)
	storedFilex, err := tenantDB.ReadOnlyConn.StoredFile.Query().
		Where(storedfile.SourceTemporaryFilePublicID(entx.NewCIText(tmpFile.PublicID.String()))).
		Only(ctxWithIncomplete)
	if err == nil {
		return storedFilex, true
	}
	if enttenant.IsNotFound(err) {
		qq.clearAccountConversionClaim(ctx, tmpFile.ID, cleanupToken)
	} else {
		log.Println(err)
	}
	return nil, false
}

func (qq *Scheduler) cleanupIncompleteAccountConversion(
	ctx context.Context,
	ctxWithIncomplete context.Context,
	tenantDB *sqlx.TenantDB,
	storedFilex *enttenant.StoredFile,
	temporaryFileID int64,
	cleanupToken string,
) {
	filem := storedfilemodel.NewStoredFile(storedFilex)
	objectName, err := filem.UnsafeTempObjectNameWithPrefix()
	if err != nil {
		log.Println(err)
		return
	}
	if qq.s3Client != nil {
		if err := qq.s3Client.RemoveObject(ctx, qq.bucketName, objectName, minio.RemoveObjectOptions{}); err != nil {
			minioErr := minio.ToErrorResponse(err)
			if minioErr.Code != "NoSuchKey" {
				log.Println(err)
				return
			}
		}
	}

	if err := tenantDB.ReadWriteConn.StoredFile.DeleteOneID(storedFilex.ID).Exec(ctxWithIncomplete); err != nil {
		if !enttenant.IsNotFound(err) {
			log.Println(err)
		}
		return
	}
	qq.clearAccountConversionClaim(ctx, temporaryFileID, cleanupToken)
}

func (qq *Scheduler) completeRecoveredAccountConversion(
	ctx context.Context,
	temporaryFileID int64,
	cleanupToken string,
) {
	err := qq.mainDB.ReadWriteConn.TemporaryFile.UpdateOneID(temporaryFileID).
		Where(temporaryfile.PersistenceClaimToken(cleanupToken)).
		SetConvertedToStoredFileAt(time.Now()).
		ClearPersistenceClaimToken().
		ClearPersistenceTenantID().
		ClearPersistenceLastProgressAt().
		ClearExpiresAt().
		Exec(ctx)
	if err != nil && !entmain.IsNotFound(err) {
		log.Println(err)
	}
}

func (qq *Scheduler) clearAccountConversionClaim(ctx context.Context, temporaryFileID int64, cleanupToken string) {
	err := qq.mainDB.ReadWriteConn.TemporaryFile.UpdateOneID(temporaryFileID).
		Where(temporaryfile.PersistenceClaimToken(cleanupToken)).
		ClearPersistenceClaimToken().
		ClearPersistenceTenantID().
		ClearPersistenceLastProgressAt().
		SetExpiresAt(time.Now().Add(temporaryAccountFileExpiry)).
		Exec(ctx)
	if err != nil && !entmain.IsNotFound(err) {
		log.Println(err)
	}
}

func (qq *Scheduler) copyTempFilesToFinalDest(ctx context.Context) {
	// iterate over all tenantDBs (or create one scheduler per tenant?)
	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		// TODO
		// use transaction? current approach has problem that fileToCopy might no longer exists
		// at time of processing. Should be a big problem as long as just the scheduler writes
		// these columns... a transaction (especially if all are read at once) could block the
		// database to long and the user might not be able to write to the db...

		// Prioritize recent uploads so stale or orphaned rows cannot delay current work.
		filesToCopy := tenantDB.ReadOnlyConn.StoredFile.Query().
			Where(
				storedfile.UploadSucceededAtNotNil(),
				storedfile.CopiedToFinalDestinationAtIsNil(),
				storedfile.DeletedTemporaryFileAtIsNil(), // not necessary, just for safety
			).
			Order(storedfile.ByID(sql.OrderAsc())).
			Limit(defaultSchedulerBatchSize).
			AllX(ctx)

		tenantx := qq.mainDB.ReadOnlyConn.Tenant.Query().Where(tenant.ID(tenantID)).OnlyX(ctx)

		for _, fileToCopyCandidate := range filesToCopy {
			fileToCopy, err := tenantDB.ReadWriteConn.StoredFile.Get(ctx, fileToCopyCandidate.ID)
			if err != nil {
				if enttenant.IsNotFound(err) {
					continue
				}
				log.Println(err)
				continue
			}

			err = qq.infra.FileSystem().PersistTemporaryTenantFile(
				ctx,
				tenantx.X25519IdentityEncrypted.Identity(),
				fileToCopy,
			)
			if err != nil {
				log.Println(err)
				continue
			}
		}

		return true
	})
}

func (qq *Scheduler) deleteProcessedTempFiles(ctx context.Context) {
	// some delay between copying and deletion in case someone is reading temp file at the moment;
	// not a problem that high because user can access files anyway and in the meantime
	// newly started reads read from final destination
	deletionThreshold := time.Now().Add(-5 * time.Minute)

	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		filesToDelete := qq.processedTempFilesToDelete(
			ctx,
			tenantDB,
			deletionThreshold,
			defaultSchedulerBatchSize,
		)

		for _, fileToDelete := range filesToDelete {
			filem := storedfilemodel.NewStoredFile(fileToDelete)

			tmpObjectName, err := filem.UnsafeTempObjectNameWithPrefix()
			if err != nil {
				log.Println(err)
				return true // continue
			}

			_, err = qq.s3Client.StatObject(ctx, qq.bucketName, tmpObjectName, minio.StatObjectOptions{})
			if err != nil {
				minioErr := minio.ToErrorResponse(err)
				if minioErr.Code == "NoSuchKey" { // TODO can this be made more type safe?
					log.Println(err, "object does not exist, may need manual deletion of orphan db entry")
				} else {
					log.Println(err)
				}
				return true // continue
			}

			err = qq.s3Client.RemoveObject(ctx, qq.bucketName, tmpObjectName, minio.RemoveObjectOptions{})
			if err != nil {
				log.Println(err)
				return true // continue
			}

			err = tenantDB.ReadWriteConn.StoredFile.UpdateOneID(fileToDelete.ID).
				SetDeletedTemporaryFileAt(time.Now()).
				Exec(ctx)
			if err != nil {
				log.Println(err, "we have an orphan db entry now")
				return true // continue
			}
		}

		return true
	})
}

func (qq *Scheduler) processedTempFilesToDelete(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	deletionThreshold time.Time,
	maxFilesPerRun int,
) []*enttenant.StoredFile {
	return tenantDB.ReadOnlyConn.StoredFile.Query().
		Where(
			storedfile.CopiedToFinalDestinationAtLT(deletionThreshold), // already copied with safety margin
			storedfile.DeletedTemporaryFileAtIsNil(),                   // not deleted yet
		).
		Order(
			storedfile.ByCopiedToFinalDestinationAt(sql.OrderAsc()),
			storedfile.ByID(sql.OrderAsc()),
		).
		Limit(maxFilesPerRun).
		AllX(ctx)
}

func (qq *Scheduler) deleteTempAccountFiles(ctx context.Context) {
	expiredTmpFiles := qq.tempAccountFilesToDelete(ctx, time.Now(), defaultSchedulerBatchSize)

	for _, tmpFile := range expiredTmpFiles {
		tmpFilem := temporaryfilemodel.NewTemporaryFile(tmpFile)

		objectName, err := tmpFilem.ObjectNameWithPrefix()
		if err != nil {
			log.Println(err)
			continue
		}

		_, err = qq.s3Client.StatObject(ctx, qq.bucketName, objectName, minio.StatObjectOptions{})
		if err != nil {
			minioErr := minio.ToErrorResponse(err)
			if minioErr.Code == "NoSuchKey" { // TODO can this be made more type safe?
				log.Println(err, "object does not exist, may need manual deletion of orphan db entry")
			} else {
				log.Println(err)
			}
			continue
		}

		err = qq.s3Client.RemoveObject(ctx, qq.bucketName, objectName, minio.RemoveObjectOptions{})
		if err != nil {
			log.Println(err)
			continue
		}

		err = qq.mainDB.ReadWriteConn.TemporaryFile.UpdateOneID(tmpFile.ID).
			SetDeletedAt(time.Now()).
			Exec(ctx)
		if err != nil {
			log.Println(err, "we have an orphan db entry now")
			continue
		}
	}

}

func (qq *Scheduler) tempAccountFilesToDelete(
	ctx context.Context,
	now time.Time,
	maxFilesPerRun int,
) []*entmain.TemporaryFile {
	return qq.mainDB.ReadOnlyConn.TemporaryFile.
		Query().
		Where(
			// if not nil, file deletion is handled by procedures for stored files
			temporaryfile.ConvertedToStoredFileAtIsNil(),
			temporaryfile.ExpiresAtLT(now), // TODO is nil ignored?
			temporaryfile.DeletedAtIsNil(),
		).
		Order(temporaryfile.ByCreatedAt(sql.OrderDesc())).
		Limit(maxFilesPerRun).
		AllX(ctx)
}
