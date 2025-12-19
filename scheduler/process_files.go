package scheduler

import (
	"context"
	"log"
	"runtime/debug"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/privacy"
	"github.com/minio/minio-go/v7"

	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/model/modelmain"
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

		// some delay between copying and deletion in case someone is reading temp file at the moment;
		// not a problem that high because user can access files anyway and in the meantime
		// newly started reads read from final destination
		time.Sleep(5 * time.Minute)

		qq.deleteProcessedTempFiles(ctx)
		qq.deleteTempAccountFiles(ctx)

		// no sleep necessary because of Sleep above
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

		filesToCopy := tenantDB.ReadWriteConn.StoredFile.Query().Where(
			storedfile.CopiedToFinalDestinationAtIsNil(),
			storedfile.DeletedTemporaryFileAtIsNil(), // not necessary, just for safety
		).AllX(ctx)

		tenantx := qq.mainDB.ReadWriteConn.Tenant.Query().Where(tenant.ID(tenantID)).OnlyX(ctx)

		for _, fileToCopy := range filesToCopy {
			err := qq.infra.FileSystem().PersistTemporaryTenantFile(
				ctx,
				tenantx.X25519IdentityEncrypted.Identity(),
				fileToCopy,
			)
			if err != nil {
				log.Println(err)
				return true // continue
			}
		}

		return true
	})
}

func (qq *Scheduler) deleteProcessedTempFiles(ctx context.Context) {
	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		filesToDelete := tenantDB.ReadWriteConn.StoredFile.Query().Where(
			storedfile.CopiedToFinalDestinationAtNotNil(), // already copied
			storedfile.DeletedTemporaryFileAtIsNil(),      // not deleted yet
		).AllX(ctx)

		for _, fileToDelete := range filesToDelete {
			filem := model.NewStoredFile(fileToDelete)

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

			err = fileToDelete.Update().SetDeletedTemporaryFileAt(time.Now()).Exec(ctx)
			if err != nil {
				log.Println(err, "we have an orphan db entry now")
				return true // continue
			}
		}

		return true
	})
}

func (qq *Scheduler) deleteTempAccountFiles(ctx context.Context) {
	expiredTmpFiles := qq.mainDB.ReadWriteConn.TemporaryFile.
		Query().
		Where(
			// if not nil, file deletion is handled by procedures for stored files
			temporaryfile.ConvertedToStoredFileAtIsNil(),
			temporaryfile.ExpiresAtLT(time.Now()), // TODO is nil ignored?
		).
		Order(temporaryfile.ByCreatedAt(sql.OrderDesc())).
		AllX(ctx)

	for _, tmpFile := range expiredTmpFiles {
		tmpFilem := modelmain.NewTemporaryFile(tmpFile)

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

		err = tmpFile.Update().SetDeletedAt(time.Now()).Exec(ctx)
		if err != nil {
			log.Println(err, "we have an orphan db entry now")
			continue
		}
	}

}
