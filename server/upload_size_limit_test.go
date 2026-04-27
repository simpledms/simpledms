package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestUploadFileCmdRejectsWhenGlobalUploadLimitExceeded(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		accountx, tenantx, tenantDB, parentDirID, spaceID := setupUploadTestSpace(t, harness)

		harness.mainDB.ReadWriteConn.SystemConfig.Update().
			SetMaxUploadSizeMib(1).
			ExecX(context.Background())

		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(harness, accountx, tenantx, tenantDB)
		if err != nil {
			t.Fatalf("new tenant context for upload: %v", err)
		}
		defer func() {
			_ = tenantTx.Rollback()
			_ = mainTx.Rollback()
		}()

		spacex := tenantCtx.TTx.Space.Query().Where(space.ID(spaceID)).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

		req, err := newUploadRequest(parentDirID, "too-large.txt", oversizedPayload())
		if err != nil {
			t.Fatalf("new upload request: %v", err)
		}

		rr := httptest.NewRecorder()
		handlerErr := harness.actions.Browse.UploadFileCmd.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			spaceCtx,
		)
		if handlerErr == nil {
			t.Fatal("expected upload size limit error")
		}

		httpErr, ok := handlerErr.(*e.HTTPError)
		if !ok {
			t.Fatalf("expected HTTPError, got %T", handlerErr)
		}
		if httpErr.StatusCode() != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, httpErr.StatusCode())
		}
	})
}

func TestUploadFileCmdRejectsWhenTenantUploadLimitOverrideIsLower(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		accountx, tenantx, tenantDB, parentDirID, spaceID := setupUploadTestSpace(t, harness)

		harness.mainDB.ReadWriteConn.SystemConfig.Update().
			SetMaxUploadSizeMib(10).
			ExecX(context.Background())
		harness.mainDB.ReadWriteConn.Tenant.UpdateOneID(tenantx.ID).
			SetMaxUploadSizeMibOverride(1).
			ExecX(context.Background())

		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(harness, accountx, tenantx, tenantDB)
		if err != nil {
			t.Fatalf("new tenant context for upload: %v", err)
		}
		defer func() {
			_ = tenantTx.Rollback()
			_ = mainTx.Rollback()
		}()

		spacex := tenantCtx.TTx.Space.Query().Where(space.ID(spaceID)).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

		req, err := newUploadRequest(parentDirID, "too-large-for-tenant.txt", oversizedPayload())
		if err != nil {
			t.Fatalf("new upload request: %v", err)
		}

		rr := httptest.NewRecorder()
		handlerErr := harness.actions.Browse.UploadFileCmd.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			spaceCtx,
		)
		if handlerErr == nil {
			t.Fatal("expected tenant upload size override error")
		}

		httpErr, ok := handlerErr.(*e.HTTPError)
		if !ok {
			t.Fatalf("expected HTTPError, got %T", handlerErr)
		}
		if httpErr.StatusCode() != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, httpErr.StatusCode())
		}
	})
}

func TestUploadFileCmdAllowsUnlimitedTenantOverride(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		accountx, tenantx, tenantDB, parentDirID, spaceID := setupUploadTestSpace(t, harness)

		harness.mainDB.ReadWriteConn.SystemConfig.Update().
			SetMaxUploadSizeMib(1).
			ExecX(context.Background())
		harness.mainDB.ReadWriteConn.Tenant.UpdateOneID(tenantx.ID).
			SetMaxUploadSizeMibOverride(0).
			ExecX(context.Background())

		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(harness, accountx, tenantx, tenantDB)
		if err != nil {
			t.Fatalf("new tenant context for upload: %v", err)
		}
		defer func() {
			_ = tenantTx.Rollback()
			_ = mainTx.Rollback()
		}()

		spacex := tenantCtx.TTx.Space.Query().Where(space.ID(spaceID)).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

		req, err := newUploadRequest(parentDirID, "allowed-by-unlimited-override.txt", oversizedPayload())
		if err != nil {
			t.Fatalf("new upload request: %v", err)
		}

		rr := httptest.NewRecorder()
		handlerErr := harness.actions.Browse.UploadFileCmd.Handler(
			httpx.NewResponseWriter(rr),
			httpx.NewRequest(req),
			spaceCtx,
		)
		if handlerErr != nil {
			t.Fatalf("expected upload success with unlimited tenant override, got %v", handlerErr)
		}
	})
}

func TestUploadFilesCmdRejectsWhenGlobalUploadLimitExceeded(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		email := fmt.Sprintf("shared-limit-%t@example.com", disableEncryption)
		createAccount(t, harness.mainDB, email, "shared-secret")

		harness.mainDB.ReadWriteConn.SystemConfig.Update().
			SetMaxUploadSizeMib(1).
			ExecX(context.Background())

		accountx := harness.mainDB.ReadWriteConn.Account.Query().
			Where(account.EmailEQ(entx.NewCIText(email))).
			OnlyX(context.Background())

		var handlerErr error
		err := withMainContext(t, harness, accountx, func(_ *entmain.Tx, mainCtx *ctxx.MainContext) error {
			req := newSharedUploadRequest(t, map[string]string{
				"too-large.txt": strings.Repeat("x", 2*1024*1024),
			})

			rr := httptest.NewRecorder()
			handlerErr = harness.actions.OpenFile.UploadFilesCmd.Handler(
				httpx.NewResponseWriter(rr),
				httpx.NewRequest(req),
				mainCtx,
			)
			return nil
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		if handlerErr == nil {
			t.Fatal("expected upload size limit error")
		}

		httpErr, ok := handlerErr.(*e.HTTPError)
		if !ok {
			t.Fatalf("expected HTTPError, got %T", handlerErr)
		}
		if httpErr.StatusCode() != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, httpErr.StatusCode())
		}
	})
}

func oversizedPayload() []byte {
	return bytes.Repeat([]byte("x"), 2*1024*1024)
}

func setupUploadTestSpace(
	t *testing.T,
	harness *actionTestHarness,
) (*entmain.Account, *entmain.Tenant, *sqlx.TenantDB, string, int64) {
	t.Helper()

	accountx, tenantx := signUpAccount(t, harness, "upload-limit-owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	var parentDirID string
	var spaceID int64

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
		spaceName := "Upload Limit Space"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceID = spacex.ID
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		parentDirID = spaceCtx.SpaceRootDir().PublicID.String()

		return nil
	})
	if err != nil {
		t.Fatalf("setup tenant content: %v", err)
	}

	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	return accountx, tenantx, tenantDB, parentDirID, spaceID
}
