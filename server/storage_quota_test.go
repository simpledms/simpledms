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
	"github.com/simpledms/simpledms/model/main/common/country"
	"github.com/simpledms/simpledms/model/main/common/language"
	"github.com/simpledms/simpledms/model/main/common/plan"
	signupmodel "github.com/simpledms/simpledms/model/main/signup"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

const testTenantQuotaTrialBytes int64 = 1 * 1024 * 1024 * 1024
const testTenantQuotaProPerUserBytes int64 = 5 * 1024 * 1024 * 1024
const testTenantQuotaUnlimitedBytes int64 = 500 * 1024 * 1024 * 1024

func TestUploadFileCmdRejectsWhenTenantStorageLimitExceeded(t *testing.T) {
	tests := []struct {
		name     string
		plan     plan.Plan
		usedSize int64
	}{
		{name: "trial", plan: plan.Trial, usedSize: testTenantQuotaTrialBytes},
		{name: "pro", plan: plan.Pro, usedSize: testTenantQuotaProPerUserBytes},
		{name: "unlimited", plan: plan.Unlimited, usedSize: testTenantQuotaUnlimitedBytes},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
				testUploadFileCmdRejectsWhenTenantStorageLimitExceeded(
					t,
					disableEncryption,
					tt.plan,
					tt.usedSize,
				)
			})
		})
	}
}

func testUploadFileCmdRejectsWhenTenantStorageLimitExceeded(
	t *testing.T,
	disableEncryption bool,
	tenantPlan plan.Plan,
	usedSize int64,
) {
	t.Helper()
	harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

	accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	var parentDirID string
	var rootDirID int64
	var spaceID int64

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		mainTx *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		mainTx.Tenant.UpdateOneID(tenantCtx.Tenant.ID).
			SetPlan(tenantPlan).
			ExecX(tenantCtx)

		spaceName := "Quota Space"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceID = spacex.ID
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootDir := spaceCtx.SpaceRootDir()
		parentDirID = rootDir.PublicID.String()
		rootDirID = rootDir.ID

		return seedTenantStorageUsage(harness, spaceCtx, rootDir.ID, usedSize)
	})
	if err != nil {
		t.Fatalf("setup tenant content: %v", err)
	}

	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)
	mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(
		harness,
		accountx,
		tenantx,
		tenantDB,
	)
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
	assertStorageQuotaError(t, handlerErr, "expected quota error")

	err = withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		return verifyStorageQuotaFileCount(tenantCtx, spaceID, rootDirID, 1)
	})
	if err != nil {
		t.Fatalf("verify rejected upload: %v", err)
	}
}

func TestUploadFileCmdRejectsWhenPlanDowngradeLeavesTenantOverLimit(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		testUploadFileCmdRejectsWhenPlanDowngradeLeavesTenantOverLimit(t, disableEncryption)
	})
}

func testUploadFileCmdRejectsWhenPlanDowngradeLeavesTenantOverLimit(
	t *testing.T,
	disableEncryption bool,
) {
	t.Helper()
	harness := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)

	accountx, tenantx := signUpAccount(t, harness, "owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	var parentDirID string
	var rootDirID int64
	var spaceID int64

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		mainTx *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		mainTx.Tenant.UpdateOneID(tenantCtx.Tenant.ID).
			SetPlan(plan.Pro).
			ExecX(tenantCtx)

		spaceName := "Downgrade Quota Space"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceID = spacex.ID
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootDir := spaceCtx.SpaceRootDir()
		parentDirID = rootDir.PublicID.String()
		rootDirID = rootDir.ID

		if err := seedTenantStorageUsage(
			harness,
			spaceCtx,
			rootDir.ID,
			testTenantQuotaTrialBytes+1,
		); err != nil {
			return err
		}

		mainTx.Tenant.UpdateOneID(tenantCtx.Tenant.ID).
			SetPlan(plan.Trial).
			ExecX(tenantCtx)
		return nil
	})
	if err != nil {
		t.Fatalf("setup tenant content: %v", err)
	}

	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)
	mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(
		harness,
		accountx,
		tenantx,
		tenantDB,
	)
	if err != nil {
		t.Fatalf("new tenant context for upload: %v", err)
	}
	defer func() {
		_ = tenantTx.Rollback()
		_ = mainTx.Rollback()
	}()

	spacex := tenantCtx.TTx.Space.Query().Where(space.ID(spaceID)).OnlyX(tenantCtx)
	spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)

	req, err := newUploadRequest(parentDirID, "over-limit-after-downgrade.txt", []byte("x"))
	if err != nil {
		t.Fatalf("new upload request: %v", err)
	}

	rr := httptest.NewRecorder()
	handlerErr := harness.actions.Browse.UploadFileCmd.Handler(
		httpx.NewResponseWriter(rr),
		httpx.NewRequest(req),
		spaceCtx,
	)
	assertStorageQuotaError(t, handlerErr, "expected quota error after plan downgrade")

	err = withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		return verifyStorageQuotaFileCount(tenantCtx, spaceID, rootDirID, 1)
	})
	if err != nil {
		t.Fatalf("verify rejected upload: %v", err)
	}
}

func TestUploadFileCmdSkipsTenantStorageLimitWhenSaaSDisabled(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		testUploadFileCmdSkipsTenantStorageLimitWhenSaaSDisabled(t, disableEncryption)
	})
}

func testUploadFileCmdSkipsTenantStorageLimitWhenSaaSDisabled(
	t *testing.T,
	disableEncryption bool,
) {
	t.Helper()
	harness := newActionTestHarnessWithSaaSAndS3(t, false)

	email := fmt.Sprintf("non-saas-owner-%t@example.com", disableEncryption)
	accountx, tenantx := signUpAccountWithoutSaaSGating(t, harness, email)
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	var parentDirID string
	var rootDirID int64
	var spaceID int64

	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		spaceName := "Quota Bypass Space"
		createSpaceViaCmd(t, harness.actions, tenantCtx, spaceName)

		spacex := tenantCtx.TTx.Space.Query().Where(space.Name(spaceName)).OnlyX(tenantCtx)
		spaceID = spacex.ID
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		rootDir := spaceCtx.SpaceRootDir()
		parentDirID = rootDir.PublicID.String()
		rootDirID = rootDir.ID

		return seedTenantStorageUsage(
			harness,
			spaceCtx,
			rootDir.ID,
			testTenantQuotaProPerUserBytes,
		)
	})
	if err != nil {
		t.Fatalf("setup tenant content: %v", err)
	}

	mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(
		harness,
		accountx,
		tenantx,
		tenantDB,
	)
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

	err = withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		return verifyStorageQuotaFileCount(tenantCtx, spaceID, rootDirID, 2)
	})
	if err != nil {
		t.Fatalf("verify upload success in non-saas mode: %v", err)
	}
}

func seedTenantStorageUsage(
	harness *actionTestHarness,
	spaceCtx *ctxx.SpaceContext,
	rootDirID int64,
	usedSize int64,
) error {
	prepared, err := harness.infra.FileSystem().PrepareFileUpload(
		spaceCtx,
		"seed.txt",
		rootDirID,
		false,
	)
	if err != nil {
		return fmt.Errorf("prepare seed upload: %w", err)
	}

	seedFileContent := []byte("seed")
	uploadResult, err := harness.infra.FileSystem().UploadPreparedFileWithExpectedSize(
		spaceCtx,
		bytes.NewReader(seedFileContent),
		prepared,
		int64(len(seedFileContent)),
	)
	if err != nil {
		return fmt.Errorf("upload seed file: %w", err)
	}

	err = harness.infra.FileSystem().FinalizePreparedUpload(spaceCtx, prepared, uploadResult)
	if err != nil {
		return fmt.Errorf("finalize seed file: %w", err)
	}

	spaceCtx.TTx.StoredFile.UpdateOneID(prepared.StoredFileID).
		SetSize(usedSize).
		ExecX(spaceCtx)
	return nil
}

func assertStorageQuotaError(t *testing.T, handlerErr error, message string) {
	t.Helper()
	if handlerErr == nil {
		t.Fatal(message)
	}

	httpErr, ok := handlerErr.(*e.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", handlerErr)
	}
	if httpErr.StatusCode() != http.StatusRequestEntityTooLarge {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusRequestEntityTooLarge,
			httpErr.StatusCode(),
		)
	}
}

func verifyStorageQuotaFileCount(
	tenantCtx *ctxx.TenantContext,
	spaceID int64,
	rootDirID int64,
	expected int,
) error {
	spacex := tenantCtx.TTx.Space.Query().Where(space.ID(spaceID)).OnlyX(tenantCtx)
	spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
	fileCount := spaceCtx.TTx.File.Query().Where(
		file.ParentID(rootDirID),
		file.IsDirectory(false),
	).CountX(spaceCtx)
	if fileCount != expected {
		return fmt.Errorf("expected %d files, got %d", expected, fileCount)
	}
	return nil
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
		false,
		harness.infra.SystemConfig().CommercialLicenseEnabled(),
	)

	_, err = signupmodel.NewSignUpService().SignUp(
		ctx,
		email,
		"Quota Tenant",
		"Test",
		"Owner",
		country.Switzerland,
		language.English,
		false,
		true,
		"",
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
