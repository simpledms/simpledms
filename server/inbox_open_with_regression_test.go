package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/model/common/storagetype"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestInboxWithSelectionPageDoesNotPanicWhenStoredFileEdgeIsMissing(t *testing.T) {
	harness := newActionTestHarnessWithSaaS(t, true)

	accountx, tenantx := signUpAccount(t, harness, "owner-open-with@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) (err error) {
		spaceName := "Open With Regression"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

		rootDir := spaceCtx.SpaceRootDir()

		inboxFile := spaceCtx.TTx.File.Create().
			SetName("open-with-regression.pdf").
			SetIsDirectory(false).
			SetIndexedAt(time.Now()).
			SetParentID(rootDir.ID).
			SetSpaceID(spacex.ID).
			SetIsInInbox(true).
			SaveX(spaceCtx)

		storedFile := spaceCtx.TTx.StoredFile.Create().
			SetFilename(inboxFile.Name).
			SetSize(128).
			SetSizeInStorage(128).
			SetStorageType(storagetype.S3).
			SetStoragePath("tenant/storage").
			SetStorageFilename("open-with-regression.pdf.gz.age").
			SetTemporaryStoragePath("tenant/tmp").
			SetTemporaryStorageFilename("open-with-regression.pdf.gz.age").
			SetUploadStartedAt(time.Now()).
			SaveX(spaceCtx)

		spaceCtx.TTx.FileVersion.Create().
			SetFileID(inboxFile.ID).
			SetStoredFileID(storedFile.ID).
			SetVersionNumber(1).
			SaveX(spaceCtx)

		urlPath := fmt.Sprintf(
			"/org/%s/space/%s/inbox/%s",
			tenantx.PublicID.String(),
			spacex.PublicID.String(),
			inboxFile.PublicID.String(),
		)
		req := httptest.NewRequest(http.MethodGet, urlPath, nil)
		req.SetPathValue("tenant_id", tenantx.PublicID.String())
		req.SetPathValue("space_id", spacex.PublicID.String())
		req.SetPathValue("file_id", inboxFile.PublicID.String())
		req.Header.Set("HX-Request", "true")

		rr := httptest.NewRecorder()

		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf(
					"inbox with selection handler panicked: %v",
					recovered,
				)
			}
		}()

		handlerErr := harness.actions.Inbox.InboxWithSelectionPage.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			spaceCtx,
		)
		if handlerErr != nil {
			return fmt.Errorf("inbox with selection handler error: %w", handlerErr)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("inbox selection should not panic: %v", err)
	}
}
