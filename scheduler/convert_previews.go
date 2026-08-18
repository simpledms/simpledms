package scheduler

import (
	"context"
	"errors"
	"io"
	"log"
	"runtime/debug"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"filippo.io/age"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	previewconversion "github.com/simpledms/simpledms/db/enttenant/previewconversion"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/internal/gotenberg"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
	previewmodel "github.com/simpledms/simpledms/model/tenant/previewconversion"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/pathx"
)

const (
	previewSchedulerInterval = 5 * time.Second
	previewClaimTimeout      = 15 * time.Minute
	previewBatchSize         = 1
)

func (qq *Scheduler) convertPreviews() {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("%v: %s", recovered, debug.Stack())
			log.Println("preview conversion loop recovered")
			time.Sleep(time.Minute)
			qq.convertPreviews()
		}
	}()

	for {
		ctx := tenantprivacy.DecisionContext(context.Background(), tenantprivacy.Allow)
		qq.convertPreviewsOnce(ctx)
		time.Sleep(previewSchedulerInterval)
	}
}

func (qq *Scheduler) convertPreviewsOnce(ctx context.Context) {
	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		tenantx, err := qq.mainDB.ReadOnlyConn.Tenant.Query().Where(tenant.ID(tenantID)).Only(ctx)
		if err != nil {
			if enttenant.IsNotFound(err) {
				log.Println("tenant not found", tenantID)
				return true
			}
			log.Println(err)
			return true
		}

		qq.discoverPreviewConversions(ctx, tenantDB)
		qq.reconcilePreviewConversions(ctx, tenantDB, tenantx.X25519IdentityEncrypted.Identity())
		qq.recoverInvalidReadyPreviewConversions(ctx, tenantDB)
		qq.recoverStalePreviewClaims(ctx, tenantDB)
		qq.cleanupOrphanedPreviewConversions(ctx, tenantDB)
		qq.processDuePreviewConversions(ctx, tenantDB, tenantx)
		return true
	})
}

func (qq *Scheduler) discoverPreviewConversions(ctx context.Context, tenantDB *sqlx.TenantDB) {
	if qq.previewDiscoveryCursor == nil {
		qq.previewDiscoveryCursor = make(map[*sqlx.TenantDB]int64)
	}
	// Final storage is the readiness marker; legacy files have no upload status timestamps.
	sourceQuery := tenantDB.ReadOnlyConn.StoredFile.Query().
		Where(
			storedfile.CopiedToFinalDestinationAtNotNil(),
			storedfile.HasFileVersionsWith(
				fileversion.HasFileWith(file.IsDirectory(false), file.DeletedAtIsNil()),
			),
			func(selector *entsql.Selector) {
				conversionTable := entsql.Table(previewconversion.Table)
				selector.Where(
					entsql.NotIn(
						selector.C(storedfile.FieldID),
						entsql.Select(conversionTable.C(previewconversion.FieldSourceStoredFileID)).
							From(conversionTable),
					),
				)
			},
		)
	if cursor := qq.previewDiscoveryCursor[tenantDB]; cursor != 0 {
		sourceQuery.Where(storedfile.IDGT(cursor))
	}
	sources := sourceQuery.
		Order(storedfile.ByID(entsql.OrderAsc())).
		Limit(defaultSchedulerBatchSize).
		AllX(ctx)
	if len(sources) == 0 {
		delete(qq.previewDiscoveryCursor, tenantDB)
		return
	}
	qq.previewDiscoveryCursor[tenantDB] = sources[len(sources)-1].ID

	for _, source := range sources {
		if _, eligible := previewmodel.Classify(source.MimeType, source.Filename, false); !eligible {
			continue
		}
		if tenantDB.ReadOnlyConn.PreviewConversion.Query().
			Where(previewconversion.SourceStoredFileID(source.ID)).
			ExistX(ctx) {
			continue
		}

		_, err := tenantDB.ReadWriteConn.PreviewConversion.Create().
			SetSourceID(source.ID).
			SetStatus(previewmodel.Pending).
			Save(ctx)
		if err != nil && !enttenant.IsConstraintError(err) {
			log.Println(err)
		}
	}
}

func (qq *Scheduler) reconcilePreviewConversions(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	tenantX25519Identity *age.X25519Identity,
) {
	conversions := tenantDB.ReadOnlyConn.PreviewConversion.Query().
		Where(
			previewconversion.StatusEQ(previewmodel.Processing),
			previewconversion.PreviewStoredFileIDNotNil(),
		).
		WithPreview().
		AllX(ctx)

	for _, conversion := range conversions {
		preview := conversion.Edges.Preview
		if preview == nil || preview.CopiedToFinalDestinationAt == nil {
			continue
		}
		if err := qq.verifyPreviewReadable(ctx, tenantX25519Identity, preview); err != nil {
			log.Println(err)
			qq.failPreviewConversion(ctx, tenantDB, conversion, gotenberg.FailureCategoryInvalidResponse)
			continue
		}

		err := tenantDB.ReadWriteConn.PreviewConversion.UpdateOneID(conversion.ID).
			SetStatus(previewmodel.Ready).
			ClearProcessingStartedAt().
			ClearNextAttemptAt().
			ClearFailureCategory().
			Exec(ctx)
		if err != nil {
			log.Println(err)
		}
	}
}

func (qq *Scheduler) recoverInvalidReadyPreviewConversions(ctx context.Context, tenantDB *sqlx.TenantDB) {
	conversions := tenantDB.ReadOnlyConn.PreviewConversion.Query().
		Where(
			previewconversion.StatusEQ(previewmodel.Ready),
			previewconversion.Or(
				previewconversion.PreviewStoredFileIDIsNil(),
				previewconversion.Not(previewconversion.HasPreview()),
				previewconversion.HasPreviewWith(storedfile.CopiedToFinalDestinationAtIsNil()),
			),
		).
		Limit(defaultSchedulerBatchSize).
		AllX(ctx)

	for _, conversion := range conversions {
		var preview *enttenant.StoredFile
		if conversion.PreviewStoredFileID != nil {
			var err error
			previewCtx := tenantprivacy.DecisionContext(ctx, tenantprivacy.Allow)
			preview, err = tenantDB.ReadOnlyConn.StoredFile.Get(previewCtx, *conversion.PreviewStoredFileID)
			if err != nil && !enttenant.IsNotFound(err) {
				log.Println(err)
				continue
			}
		}
		if preview != nil && preview.CopiedToFinalDestinationAt != nil {
			continue
		}

		update := tenantDB.ReadWriteConn.PreviewConversion.UpdateOneID(conversion.ID).
			ClearNextAttemptAt().
			ClearFailureCategory()
		if preview == nil {
			update.ClearPreviewStoredFileID().
				SetStatus(previewmodel.Pending).
				SetRetryCount(0).
				ClearLastAttemptedAt().
				ClearProcessingStartedAt()
		} else {
			update.SetStatus(previewmodel.Processing).
				SetProcessingStartedAt(time.Now())
		}
		if err := update.Exec(ctx); err != nil {
			log.Println(err)
		}
	}
}

func (qq *Scheduler) recoverStalePreviewClaims(ctx context.Context, tenantDB *sqlx.TenantDB) {
	threshold := time.Now().Add(-previewClaimTimeout)
	conversions := tenantDB.ReadOnlyConn.PreviewConversion.Query().
		Where(
			previewconversion.StatusEQ(previewmodel.Processing),
			previewconversion.ProcessingStartedAtLT(threshold),
		).
		WithPreview().
		AllX(ctx)

	for _, conversion := range conversions {
		if conversion.Edges.Preview != nil {
			if err := qq.discardPreview(ctx, tenantDB, conversion.Edges.Preview); err != nil {
				log.Println(err)
				continue
			}
		}
		update := tenantDB.ReadWriteConn.PreviewConversion.UpdateOneID(conversion.ID).
			ClearPreviewStoredFileID().
			SetStatus(previewmodel.Pending).
			ClearProcessingStartedAt()
		if err := update.Exec(ctx); err != nil {
			log.Println(err)
		}
	}
}

func (qq *Scheduler) cleanupOrphanedPreviewConversions(ctx context.Context, tenantDB *sqlx.TenantDB) {
	conversions := tenantDB.ReadOnlyConn.PreviewConversion.Query().
		WithPreview().
		Limit(defaultSchedulerBatchSize).
		AllX(ctx)
	for _, conversion := range conversions {
		if tenantDB.ReadOnlyConn.FileVersion.Query().
			Where(fileversion.StoredFileID(conversion.SourceStoredFileID)).
			ExistX(ctx) {
			continue
		}
		if conversion.Edges.Preview != nil {
			if err := qq.discardPreview(ctx, tenantDB, conversion.Edges.Preview); err != nil {
				log.Println(err)
				continue
			}
		}
		if err := tenantDB.ReadWriteConn.PreviewConversion.DeleteOneID(conversion.ID).Exec(ctx); err != nil {
			log.Println(err)
		}
	}
}

func (qq *Scheduler) processDuePreviewConversions(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	tenantx *entmain.Tenant,
) {
	conversions := tenantDB.ReadOnlyConn.PreviewConversion.Query().
		Where(
			previewconversion.StatusEQ(previewmodel.Pending),
			previewconversion.Or(
				previewconversion.NextAttemptAtIsNil(),
				previewconversion.NextAttemptAtLTE(time.Now()),
			),
		).
		Order(previewconversion.ByID(entsql.OrderAsc())).
		Limit(previewBatchSize).
		WithSource().
		AllX(ctx)

	for _, conversion := range conversions {
		claimed := tenantDB.ReadWriteConn.PreviewConversion.UpdateOneID(conversion.ID).
			Where(
				previewconversion.StatusEQ(previewmodel.Pending),
				previewconversion.Or(
					previewconversion.NextAttemptAtIsNil(),
					previewconversion.NextAttemptAtLTE(time.Now()),
				),
			).
			SetStatus(previewmodel.Processing).
			SetLastAttemptedAt(time.Now()).
			SetProcessingStartedAt(time.Now())
		_, err := claimed.Save(ctx)
		if err != nil {
			if !enttenant.IsNotFound(err) {
				log.Println(err)
			}
			continue
		}

		qq.convertOnePreview(ctx, tenantDB, tenantx, conversion)
	}
}

func (qq *Scheduler) convertOnePreview(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	tenantx *entmain.Tenant,
	conversion *enttenant.PreviewConversion,
) {
	source := conversion.Edges.Source
	if source == nil {
		qq.failOrRetryPreview(ctx, tenantDB, conversion, &gotenberg.ConversionError{
			Category: gotenberg.FailureCategorySourceUnreadable,
		})
		return
	}

	classification, eligible := previewmodel.Classify(source.MimeType, source.Filename, false)
	if !eligible || source.CopiedToFinalDestinationAt == nil {
		qq.failOrRetryPreview(ctx, tenantDB, conversion, &gotenberg.ConversionError{
			Category:  gotenberg.FailureCategorySourceUnreadable,
			Retryable: false,
		})
		return
	}

	identity := tenantx.X25519IdentityEncrypted.Identity()
	openedSource, err := qq.infra.FileSystem().UnsafeOpenFile(ctx, identity, storedfilemodel.NewStoredFile(source))
	if err != nil {
		log.Println(err)
		qq.failOrRetryPreview(ctx, tenantDB, conversion, &gotenberg.ConversionError{
			Category: gotenberg.FailureCategorySourceUnreadable,
			Err:      err,
		})
		return
	}
	defer func() {
		if err := openedSource.Close(); err != nil {
			log.Println(err)
		}
	}()

	pdf, err := qq.gotenbergClientNilable.Convert(ctx, classification, openedSource)
	if err != nil {
		log.Println(err)
		qq.failOrRetryPreview(ctx, tenantDB, conversion, err)
		return
	}

	fileInfo, storageFilename, fileSize, contentSHA256, err := qq.infra.FileSystem().SaveDerivedPDF(
		ctx,
		tenantx.PublicID.String(),
		identity,
		pdf,
		classification.OutputFilename,
	)
	closeErr := pdf.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		log.Println(err)
		qq.failOrRetryPreview(ctx, tenantDB, conversion, &gotenberg.ConversionError{
			Category:  gotenberg.FailureCategoryService,
			Retryable: true,
			Err:       err,
		})
		return
	}

	temporaryStoragePath := pathx.S3TemporaryStoragePrefix(tenantx.PublicID.String())
	finalStoragePath := pathx.S3StoragePrefix(tenantx.PublicID.String())
	storedPreview, err := tenantDB.ReadWriteConn.StoredFile.Create().
		SetFilename(classification.OutputFilename).
		SetSize(fileSize).
		SetSizeInStorage(fileInfo.Size).
		SetSha256(fileInfo.ChecksumSHA256).
		SetContentSha256(contentSHA256).
		SetMimeType("application/pdf").
		SetStorageType(storagetype.S3).
		SetBucketName(qq.bucketName).
		SetStoragePath(finalStoragePath).
		SetStorageFilename(storageFilename).
		SetTemporaryStoragePath(temporaryStoragePath).
		SetTemporaryStorageFilename(storageFilename).
		SetUploadStartedAt(time.Now()).
		SetUploadSucceededAt(time.Now()).
		Save(ctx)
	if err != nil {
		log.Println(err)
		if cleanupErr := qq.infra.FileSystem().RemoveTemporaryObject(ctx, temporaryStoragePath, storageFilename); cleanupErr != nil {
			log.Println(cleanupErr)
		}
		qq.failOrRetryPreview(ctx, tenantDB, conversion, &gotenberg.ConversionError{
			Category:  gotenberg.FailureCategoryService,
			Retryable: true,
			Err:       err,
		})
		return
	}

	conversionUpdate := tenantDB.ReadWriteConn.PreviewConversion.UpdateOneID(conversion.ID).
		SetPreviewID(int64(storedPreview.ID)).
		SetStatus(previewmodel.Processing).
		SetProcessingStartedAt(time.Now()).
		ClearNextAttemptAt().
		ClearFailureCategory()
	if err := conversionUpdate.Exec(ctx); err != nil {
		log.Println(err)
		if cleanupErr := qq.discardPreview(ctx, tenantDB, storedPreview); cleanupErr != nil {
			log.Println(cleanupErr)
		}
		return
	}

	log.Printf(
		"preview conversion stored source=%d preview=%d route=%s",
		source.ID,
		storedPreview.ID,
		classification.Route,
	)
}

func (qq *Scheduler) failOrRetryPreview(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	conversion *enttenant.PreviewConversion,
	err error,
) {
	conversionErr := &gotenberg.ConversionError{Category: gotenberg.FailureCategoryService, Retryable: true, Err: err}
	if typedErr, ok := err.(*gotenberg.ConversionError); ok {
		conversionErr = typedErr
	}
	log.Printf(
		"preview conversion failed source=%d attempt=%d category=%s retryable=%t trace_id=%s: %v",
		conversion.SourceStoredFileID,
		conversion.RetryCount+1,
		conversionErr.Category,
		conversionErr.Retryable,
		conversionErr.TraceID,
		conversionErr,
	)

	if conversion.PreviewStoredFileID != nil {
		preview, getErr := tenantDB.ReadOnlyConn.StoredFile.Get(ctx, *conversion.PreviewStoredFileID)
		if getErr == nil {
			if discardErr := qq.discardPreview(ctx, tenantDB, preview); discardErr != nil {
				log.Println(discardErr)
				return
			}
		}
	}

	update := tenantDB.ReadWriteConn.PreviewConversion.UpdateOneID(conversion.ID).
		ClearPreviewStoredFileID().
		SetFailureCategory(conversionErr.Category).
		ClearProcessingStartedAt()
	if conversionErr.Retryable && conversion.RetryCount < 3 {
		retryCount := conversion.RetryCount + 1
		update.SetStatus(previewmodel.Pending).SetRetryCount(retryCount)
		nextAttemptAt := time.Now().Add(previewRetryDelay(retryCount))
		update.SetNextAttemptAt(nextAttemptAt)
	} else {
		update.SetStatus(previewmodel.Failed).ClearNextAttemptAt()
	}
	if updateErr := update.Exec(ctx); updateErr != nil {
		log.Println(updateErr)
	}
}

func (qq *Scheduler) failPreviewConversion(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	conversion *enttenant.PreviewConversion,
	category string,
) {
	if conversion.Edges.Preview != nil {
		if err := qq.discardPreview(ctx, tenantDB, conversion.Edges.Preview); err != nil {
			log.Println(err)
			return
		}
	}
	err := tenantDB.ReadWriteConn.PreviewConversion.UpdateOneID(conversion.ID).
		ClearPreviewStoredFileID().
		SetStatus(previewmodel.Failed).
		SetFailureCategory(category).
		ClearProcessingStartedAt().
		ClearNextAttemptAt().
		Exec(ctx)
	if err != nil {
		log.Println(err)
	}
}

func (qq *Scheduler) discardPreview(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	preview *enttenant.StoredFile,
) error {
	if err := qq.infra.FileSystem().RemoveTenantStoredFileObjects(ctx, preview); err != nil {
		return err
	}
	return tenantDB.ReadWriteConn.StoredFile.DeleteOneID(preview.ID).Exec(ctx)
}

func (qq *Scheduler) verifyPreviewReadable(
	ctx context.Context,
	tenantX25519Identity *age.X25519Identity,
	preview *enttenant.StoredFile,
) error {
	openedPreview, err := qq.infra.FileSystem().UnsafeOpenFile(ctx, tenantX25519Identity, storedfilemodel.NewStoredFile(preview))
	if err != nil {
		return err
	}
	defer func() {
		if err := openedPreview.Close(); err != nil {
			log.Println(err)
		}
	}()
	magic := make([]byte, 5)
	if _, err := io.ReadFull(openedPreview, magic); err != nil {
		return err
	}
	if string(magic) != "%PDF-" {
		return errors.New("stored preview is not a PDF")
	}
	return nil
}

func previewRetryDelay(retryCount int) time.Duration {
	switch retryCount {
	case 1:
		return time.Minute
	case 2:
		return 5 * time.Minute
	default:
		return 15 * time.Minute
	}
}
