package scheduler

import (
	"context"
	"errors"
	"log"
	"runtime/debug"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/privacy"
	"filippo.io/age"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/internal/xberg"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/util/ocrutil"
)

// applyOCRMarkdown stores Xberg markdown, which keeps tables, for LLMs (MCP). It runs
// independently of the Tika OCR in applyOCR, which provides the full-text search content.
func (qq *Scheduler) applyOCRMarkdown() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%v: %s", r, debug.Stack())
			log.Println("trying to recover")

			time.Sleep(1 * time.Minute)

			qq.applyOCRMarkdown()
		}
	}()

	for {
		ctx := context.Background()
		ctx = privacy.DecisionContext(ctx, privacy.Allow)

		qq.applyOCRMarkdownx(ctx)

		time.Sleep(5 * time.Second)
	}
}

func (qq *Scheduler) applyOCRMarkdownx(ctx context.Context) {
	dateThreshold := time.Now().Add(-12 * time.Hour)

	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		qq.applyOCRMarkdownForTenant(ctx, tenantID, tenantDB, dateThreshold)
		return true
	})
}

func (qq *Scheduler) applyOCRMarkdownForTenant(
	ctx context.Context,
	tenantID int64,
	tenantDB *sqlx.TenantDB,
	dateThreshold time.Time,
) {
	tenantIdentity, found := qq.ocrTenantIdentity(ctx, tenantID)
	if !found {
		return
	}

	// see applyOCRForTenant for why the pending condition is written verbatim
	filesToProcess := tenantDB.ReadOnlyConn.File.
		Query().
		Where(
			func(selector *entsql.Selector) {
				selector.Where(entsql.ExprP(schema.FileOCRMarkdownPendingCondition))
			},
			file.Or(
				file.OcrMarkdownLastTriedAtIsNil(),
				file.OcrMarkdownLastTriedAtLT(dateThreshold),
			),
			hasVersionAtFinalDestination,
		).
		Order(file.ByID(entsql.OrderAsc())).
		Limit(defaultSchedulerBatchSize).
		AllX(ctx)

	for _, fileToProcess := range filesToProcess {
		if !qq.applyOCRMarkdownPendingFile(ctx, tenantDB, tenantIdentity, fileToProcess) {
			return
		}
	}
}

// The result indicates whether processing can continue with the next pending file.
func (qq *Scheduler) applyOCRMarkdownPendingFile(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	tenantIdentity *age.X25519Identity,
	fileToProcess *enttenant.File,
) bool {
	currentVersion := filemodel.NewFile(fileToProcess).CurrentVersion(ctx)
	if ocrutil.IsFileTooLarge(currentVersion.Data.Size) {
		qq.saveOCRMarkdownTooLarge(ctx, tenantDB, fileToProcess)
		return true
	}
	if !currentVersion.IsMovedToFinalDestination() {
		return true
	}

	markdown, err := qq.extractOCRMarkdown(ctx, tenantIdentity, currentVersion)
	return qq.saveOCRMarkdownResult(ctx, tenantDB, fileToProcess, markdown, err)
}

func (qq *Scheduler) extractOCRMarkdown(
	ctx context.Context,
	tenantIdentity *age.X25519Identity,
	currentVersion *storedfilemodel.StoredFile,
) (string, error) {
	openedFile, err := qq.infra.FileSystem().UnsafeOpenFile(ctx, tenantIdentity, currentVersion)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer func() {
		err := openedFile.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	markdown, err := qq.xbergClientNilable.ExtractMarkdown(
		ctx,
		currentVersion.Data.Filename,
		currentVersion.Data.MimeType,
		openedFile,
	)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return markdown, nil
}

// saveOCRMarkdownResult stores the markdown and, without Tika, also the search content if the
// file has none yet; applyOCR replaces it once Tika is configured. Unsupported formats are stored as empty markdown because retrying cannot
// succeed. The result indicates whether processing can continue with the next pending file.
func (qq *Scheduler) saveOCRMarkdownResult(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	fileToProcess *enttenant.File,
	markdown string,
	extractErr error,
) bool {
	if extractErr != nil && !errors.Is(extractErr, xberg.ErrUnsupportedFormat) {
		err := tenantDB.ReadWriteConn.File.UpdateOneID(fileToProcess.ID).
			SetOcrMarkdownRetryCount(fileToProcess.OcrMarkdownRetryCount + 1).
			SetOcrMarkdownLastTriedAt(time.Now()).
			Exec(ctx)
		if err != nil {
			log.Println(err)
		}
		return false
	}

	now := time.Now()
	update := tenantDB.ReadWriteConn.File.UpdateOneID(fileToProcess.ID).
		SetOcrMarkdown(markdown).
		SetOcrMarkdownSuccessAt(now).
		SetOcrMarkdownRetryCount(0).
		ClearOcrMarkdownLastTriedAt()
	if qq.tikaClientNilable == nil && fileToProcess.OcrSuccessAt == nil {
		update.
			SetOcrContent(removeAllWhitespace(markdown)).
			SetIsOcrContentFromMarkdown(true).
			SetOcrSuccessAt(now).
			SetOcrRetryCount(0).
			SetOcrLastTriedAt(time.Time{})
	}
	err := update.Exec(ctx)
	if err != nil {
		log.Println(err)
	}
	return true
}

func (qq *Scheduler) saveOCRMarkdownTooLarge(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	fileToProcess *enttenant.File,
) {
	now := time.Now()
	// retry count 3 marks the file as done, like applyOCRPendingFile does
	update := tenantDB.ReadWriteConn.File.UpdateOneID(fileToProcess.ID).
		SetOcrMarkdown("").
		SetOcrMarkdownRetryCount(3).
		SetOcrMarkdownLastTriedAt(now)
	if qq.tikaClientNilable == nil && fileToProcess.OcrSuccessAt == nil {
		update.
			SetOcrContent("").
			SetOcrRetryCount(3).
			SetOcrLastTriedAt(now)
	}
	err := update.Exec(ctx)
	if err != nil {
		log.Println(err)
	}
}
