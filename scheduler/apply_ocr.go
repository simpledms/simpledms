package scheduler

import (
	"context"
	"log"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"entgo.io/ent/privacy"
	"filippo.io/age"
	"github.com/marcobeierer/go-tika"

	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/util/ocrutil"
)

func (qq *Scheduler) applyOCR() {
	if qq.tikaClientNilable == nil {
		log.Println("tika client not initialized")
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
		time.Sleep(15 * time.Second)
	}
}

func (qq *Scheduler) applyOCRx(ctx context.Context) {
	dateThreshold := time.Now().Add(-12 * time.Hour)

	// iterate over all tenantDBs (or create one scheduler per tenant?)
	qq.tenantDBs.Range(func(tenantID int64, tenantDB *sqlx.TenantDB) bool {
		// TODO is tx necessary on mainDB?
		tenantx := qq.mainDB.ReadOnlyConn.Tenant.Query().Where(tenant.ID(tenantID)).OnlyX(ctx)
		tenantIdentity := tenantx.X25519IdentityEncrypted.Identity()

		// TODO transaction? if so, make sure OCRRetryCount gets increased
		// TODO ensure that only files at final destination get processed;
		//		are inbox files at final destination?
		filesToProcess := tenantDB.ReadWriteConn.File.
			Query().
			Where(
				file.OcrSuccessAtIsNil(),
				file.OcrRetryCountLT(3),
				file.OcrLastTriedAtLT(dateThreshold), // TODO is this correct? what is value?
				file.HasVersionsWith(
					// has to be rechecked later because current version has to be
					// at final destination. This query checks only for any version
					storedfile.CopiedToFinalDestinationAtNotNil(),
				),
				file.IsDirectory(false),
			).
			AllX(ctx)

		for _, fileToProcess := range filesToProcess {
			currentVersion := model.NewFile(fileToProcess).CurrentVersion(ctx)
			content, fileNotReady, fileTooLarge, err := qq.applyOCROneFile(ctx, tenantIdentity, currentVersion)
			if err != nil {
				log.Println(err)

				fileToProcess.Update().
					SetOcrRetryCount(fileToProcess.OcrRetryCount + 1).
					SetOcrLastTriedAt(time.Now()).
					ExecX(ctx)

				// TODO continue or not
				return true
			}
			if fileNotReady {
				return true // continue with next tenant
			}
			if fileTooLarge {
				// TODO find a more expressive solution to store in database
				//		that file is to large
				fileToProcess.Update().
					SetOcrContent("").
					SetOcrRetryCount(3).
					SetOcrLastTriedAt(time.Now()).
					ExecX(ctx)

				continue
			}

			// FIXME start transaction?
			fileToProcess.Update().
				SetOcrRetryCount(0).
				SetOcrLastTriedAt(time.Time{}).
				SetOcrSuccessAt(time.Now()).
				SetOcrContent(content).
				SaveX(ctx)
		}

		return true
	})
}

// bool return values indicate if file is not ready, for example not moved to final destination
// second bool value indicates if OCR was not applied because file is too large
func (qq *Scheduler) applyOCROneFile(
	ctx context.Context,
	tenantIdentity *age.X25519Identity,
	currentVersion *model.StoredFile,
) (string, bool, bool, error) {
	// TODO use language of user?
	tikaHeader := tika.NewHeader().AcceptText().SetOCRLanguage("eng+deu+fra+ita+spa")

	if ocrutil.IsFileTooLarge(currentVersion.Data.Size) {
		return "", false, true, nil
	}

	if !currentVersion.IsMovedToFinalDestination() {
		return "", true, false, nil
	}

	openedFile, err := qq.infra.FileSystem().UnsafeOpenFile(ctx, tenantIdentity, currentVersion)
	if err != nil {
		log.Println(err)
		return "", false, false, err
	}
	defer func() {
		err := openedFile.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	parsedContent, err := qq.tikaClientNilable.Parse(context.Background(), openedFile, tikaHeader)
	if err != nil {
		log.Println(err)
		return "", false, false, err
	}

	return removeAllWhitespace(parsedContent), false, false, nil
}

var regexpEndsAlphanumeric = regexp.MustCompile("[a-zA-Z0-9]$")

func removeAllWhitespace(text string) string {
	parsedContentSlice := strings.Split(text, "\n")
	var contentSlice []string

	// remove all whitespace
	for _, paragraph := range parsedContentSlice {
		trimmed := strings.TrimSpace(paragraph)

		if trimmed == "" {
			continue
		}

		// add dot if last char is alphanumeric
		// TODO not a perfect solution, sometimes to many dots are added,
		// 		for example if paragraph was not detected correctly by OCR
		if regexpEndsAlphanumeric.MatchString(trimmed) {
			trimmed += "."
		}

		contentSlice = append(contentSlice, trimmed)
	}

	return strings.Join(contentSlice, " ")
}
