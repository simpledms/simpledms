package scheduler

import (
	"context"
	"io"
	"log"
	"runtime/debug"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/privacy"
	"filippo.io/age"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/db/sqlx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/util/ocrutil"
)

func (qq *Scheduler) applyOCR() {
	if qq.textExtractorNilable == nil {
		log.Println("neither Xberg nor Tika configured, OCR disabled")
		return
	}

	defer func() {
		// tested and works
		if r := recover(); r != nil {
			log.Printf("%v: %s", r, debug.Stack())
			log.Println("trying to recover")

			// TODO what is a good interval
			time.Sleep(1 * time.Minute)

			// tested and works, automatically restarts loop
			qq.applyOCR()
		}
	}()

	for {
		ctx := context.Background()
		ctx = privacy.DecisionContext(ctx, privacy.Allow)

		qq.applyOCRx(ctx)

		// TODO is this to short? how expensive is this in larger instances?
		time.Sleep(5 * time.Second)
	}
}

func (qq *Scheduler) applyOCRx(ctx context.Context) {
	dateThreshold := time.Now().Add(-12 * time.Hour)

	// iterate over all tenantDBs (or create one scheduler per tenant?)
	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		qq.applyOCRForTenant(ctx, tenantID, tenantDB, dateThreshold)
		return true
	})
}

func (qq *Scheduler) applyOCRForTenant(
	ctx context.Context,
	tenantID int64,
	tenantDB *sqlx.TenantDB,
	dateThreshold time.Time,
) {
	// TODO is tx necessary on mainDB?
	tenantx, err := qq.mainDB.ReadOnlyConn.Tenant.Query().Where(tenant.ID(tenantID)).Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			// can happen if tenant was deleted and not removed from tenantDBs yet
			log.Println("tenant not found", tenantID)
			return
		}
		log.Println(err)
		return
	}
	tenantIdentity := tenantx.X25519IdentityEncrypted.Identity()

	// TODO transaction? if so, make sure OCRRetryCount gets increased
	// TODO ensure that only files at final destination get processed;
	//		are inbox files at final destination?
	// Runs every few seconds for each tenant, mostly finding nothing. The pending condition
	// is written verbatim so the partial index stays usable, and EXISTS lets SQLite start
	// from the files instead of reading all file versions; generated predicates prevented
	// both.
	filesToProcess := tenantDB.ReadOnlyConn.File.
		Query().
		Where(
			func(selector *entsql.Selector) {
				selector.Where(entsql.ExprP(schema.FileOCRPendingCondition))
			},
			file.OcrLastTriedAtLT(dateThreshold), // TODO is this correct? what is value?
			// has to be rechecked later because current version has to be
			// at final destination. This query checks only for any version
			hasVersionAtFinalDestination,
		).
		Order(file.ByID(entsql.OrderAsc())).
		Limit(defaultSchedulerBatchSize).
		AllX(ctx)

	for _, fileToProcess := range filesToProcess {
		if !qq.applyOCRPendingFile(ctx, tenantDB, tenantIdentity, fileToProcess) {
			return
		}
	}
}

// The result indicates whether processing can continue with the next pending file.
func (qq *Scheduler) applyOCRPendingFile(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	tenantIdentity *age.X25519Identity,
	fileToProcess *enttenant.File,
) bool {
	currentVersion := filemodel.NewFile(fileToProcess).CurrentVersion(ctx)
	content, fileNotReady, fileTooLarge, err := qq.applyOCROneFile(ctx, tenantIdentity, currentVersion)
	if err != nil {
		log.Println(err)
		err = tenantDB.ReadWriteConn.File.UpdateOneID(fileToProcess.ID).
			SetOcrRetryCount(fileToProcess.OcrRetryCount + 1).
			SetOcrLastTriedAt(time.Now()).
			Exec(ctx)
		if err != nil {
			log.Println(err)
			// TODO continue or not
		}
		return false
	}
	if fileNotReady {
		return true
	}
	if fileTooLarge {
		// TODO find a more expressive solution to store in database
		//		that file is to large
		err = tenantDB.ReadWriteConn.File.UpdateOneID(fileToProcess.ID).
			SetOcrContent("").
			SetOcrRetryCount(3).
			SetOcrLastTriedAt(time.Now()).
			Exec(ctx)
		if err != nil {
			log.Println(err)
		}
		return true
	}

	// FIXME start transaction?
	err = tenantDB.ReadWriteConn.File.UpdateOneID(fileToProcess.ID).
		SetOcrRetryCount(0).
		SetOcrLastTriedAt(time.Time{}).
		SetOcrSuccessAt(time.Now()).
		SetOcrContent(content).
		Exec(ctx)
	if err != nil {
		log.Println(err)
	}
	return true
}

// bool return values indicate if file is not ready, for example not moved to final destination
// second bool value indicates if OCR was not applied because file is too large
func (qq *Scheduler) applyOCROneFile(
	ctx context.Context,
	tenantIdentity *age.X25519Identity,
	currentVersion *storedfilemodel.StoredFile,
) (string, bool, bool, error) {
	if ocrutil.IsFileTooLarge(currentVersion.Data.Size) {
		return "", false, true, nil
	}

	if !currentVersion.IsMovedToFinalDestination() {
		return "", true, false, nil
	}

	openFile := func() (io.ReadCloser, error) {
		return qq.infra.FileSystem().UnsafeOpenFile(ctx, tenantIdentity, currentVersion)
	}
	parsedContent, err := qq.textExtractorNilable.ExtractText(
		ctx,
		currentVersion.Data.Filename,
		currentVersion.Data.MimeType,
		openFile,
	)
	if err != nil {
		log.Println(err)
		return "", false, false, err
	}

	return parsedContent, false, false, nil
}

// hasVersionAtFinalDestination is file.HasVersionsWith(
// storedfile.CopiedToFinalDestinationAtNotNil()) as a correlated EXISTS, see applyOCRx.
func hasVersionAtFinalDestination(selector *entsql.Selector) {
	versions := entsql.Table(fileversion.Table)
	storedFiles := entsql.Table(storedfile.Table)
	selector.Where(entsql.Exists(
		entsql.Select(versions.C(fileversion.FieldFileID)).
			From(versions).
			Join(storedFiles).
			On(versions.C(fileversion.FieldStoredFileID), storedFiles.C(storedfile.FieldID)).
			Where(entsql.And(
				entsql.ColumnsEQ(versions.C(fileversion.FieldFileID), selector.C(file.FieldID)),
				entsql.NotNull(storedFiles.C(storedfile.FieldCopiedToFinalDestinationAt)),
			)),
	))
}
