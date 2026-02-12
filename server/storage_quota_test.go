package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

const testTenantQuotaPerUserBytes int64 = 5 * 1024 * 1024 * 1024

func TestUploadFileCmdRejectsWhenTenantStorageLimitExceeded(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

		accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
		tenantDB := initTenantDB(t, harness, tenantx)
		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		var parentDirID string
		var rootDirID int64
		var spaceID int64

		err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
			spaceName := "Quota Space"
			createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

			spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
			spaceID = spacex.ID
			spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
			rootDir := spaceCtx.SpaceRootDir()
			parentDirID = rootDir.PublicID.String()
			rootDirID = rootDir.ID

			prepared, _, err := harness.infra.FileSystem().PrepareFileUpload(spaceCtx, "seed.txt", rootDir.ID, false)
			if err != nil {
				return fmt.Errorf("prepare seed upload: %w", err)
			}

			seedFileContent := []byte("seed")
			fileInfo, fileSize, err := harness.infra.FileSystem().UploadPreparedFileWithExpectedSize(
				spaceCtx,
				bytes.NewReader(seedFileContent),
				prepared,
				int64(len(seedFileContent)),
			)
			if err != nil {
				return fmt.Errorf("upload seed file: %w", err)
			}

			err = harness.infra.FileSystem().FinalizePreparedUpload(spaceCtx, prepared, fileInfo, fileSize)
			if err != nil {
				return fmt.Errorf("finalize seed file: %w", err)
			}

			spaceCtx.TTx.StoredFile.UpdateOneID(prepared.StoredFileID).
				SetSize(testTenantQuotaPerUserBytes).
				ExecX(spaceCtx)

			return nil
		})
		if err != nil {
			t.Fatalf("setup tenant content: %v", err)
		}

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

		req, err := newUploadRequest(parentDirID, "over-limit.txt", []byte("x"))
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
			t.Fatal("expected quota error")
		}

		httpErr, ok := handlerErr.(*e.HTTPError)
		if !ok {
			t.Fatalf("expected HTTPError, got %T", handlerErr)
		}
		if httpErr.StatusCode() != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, httpErr.StatusCode())
		}

		err = withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
			spacex := tenantCtx.TTx.Space.Query().Where(space.ID(spaceID)).OnlyX(tenantCtx)
			spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

			fileCount := spaceCtx.TTx.File.Query().Where(
				file.ParentID(rootDirID),
				file.IsDirectory(false),
			).CountX(spaceCtx)
			if fileCount != 1 {
				return fmt.Errorf("expected one file after rejected upload, got %d", fileCount)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("verify rejected upload: %v", err)
		}
	})
}

func TestUploadFileCmdSkipsTenantStorageLimitWhenSaaSDisabled(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		harness := newActionTestHarnessWithSaaSAndS3(t, false)

		email := fmt.Sprintf("non-saas-owner-%t@example.com", disableEncryption)
		accountx, tenantx := signUpAccountWithoutSaaSGating(t, harness, email)
		tenantDB := initTenantDB(t, harness, tenantx)
		tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

		var parentDirID string
		var rootDirID int64
		var spaceID int64

		err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
			spaceName := "Quota Bypass Space"
			createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

			spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
			spaceID = spacex.ID
			spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
			rootDir := spaceCtx.SpaceRootDir()
			parentDirID = rootDir.PublicID.String()
			rootDirID = rootDir.ID

			prepared, _, err := harness.infra.FileSystem().PrepareFileUpload(spaceCtx, "seed.txt", rootDir.ID, false)
			if err != nil {
				return fmt.Errorf("prepare seed upload: %w", err)
			}

			seedFileContent := []byte("seed")
			fileInfo, fileSize, err := harness.infra.FileSystem().UploadPreparedFileWithExpectedSize(
				spaceCtx,
				bytes.NewReader(seedFileContent),
				prepared,
				int64(len(seedFileContent)),
			)
			if err != nil {
				return fmt.Errorf("upload seed file: %w", err)
			}

			err = harness.infra.FileSystem().FinalizePreparedUpload(spaceCtx, prepared, fileInfo, fileSize)
			if err != nil {
				return fmt.Errorf("finalize seed file: %w", err)
			}

			spaceCtx.TTx.StoredFile.UpdateOneID(prepared.StoredFileID).
				SetSize(testTenantQuotaPerUserBytes).
				ExecX(spaceCtx)

			return nil
		})
		if err != nil {
			t.Fatalf("setup tenant content: %v", err)
		}

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

		req, err := newUploadRequest(parentDirID, "over-limit-in-non-saas.txt", []byte("x"))
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
			t.Fatalf("expected upload success in non-saas mode, got %v", handlerErr)
		}

		err = withTenantContext(t, harness, accountx, tenantx, tenantDB, func(_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext) error {
			spacex := tenantCtx.TTx.Space.Query().Where(space.ID(spaceID)).OnlyX(tenantCtx)
			spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

			fileCount := spaceCtx.TTx.File.Query().Where(
				file.ParentID(rootDirID),
				file.IsDirectory(false),
			).CountX(spaceCtx)
			if fileCount != 2 {
				return fmt.Errorf("expected two files in non-saas mode, got %d", fileCount)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("verify upload success in non-saas mode: %v", err)
		}
	})
}

func signUpAccountWithoutSaaSGating(
	t *testing.T,
	harness *actionTestHarness,
	email string,
) (*entmain.Account, *entmain.Tenant) {
	t.Helper()

	mainTx, err := harness.mainDB.ReadWriteConn.Tx(context.Background())
	if err != nil {
		t.Fatalf("start main tx: %v", err)
	}

	ctx := ctxx.NewVisitorContext(
		context.Background(),
		mainTx,
		harness.i18n,
		"",
		"",
		true,
		harness.infra.SystemConfig().CommercialLicenseEnabled(),
	)

	_, err = modelmain.NewSignUpService().SignUp(
		ctx,
		email,
		"Quota Tenant",
		"Test",
		"Owner",
		country.Switzerland,
		language.English,
		false,
		true,
	)
	if err != nil {
		_ = mainTx.Rollback()
		t.Fatalf("create tenant and account without saas gating: %v", err)
	}

	if err := mainTx.Commit(); err != nil {
		t.Fatalf("commit main tx: %v", err)
	}

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())
	tenantx := accountx.QueryTenants().OnlyX(context.Background())

	return accountx, tenantx
}
