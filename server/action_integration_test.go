package server

import (
	"context"
	"encoding/json"
	"fmt"
	htmlstd "html"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	migratemain "github.com/simpledms/simpledms/db/entmain/migrate"
	"github.com/simpledms/simpledms/db/entmain/passkeycredential"
	_ "github.com/simpledms/simpledms/db/entmain/runtime"
	"github.com/simpledms/simpledms/db/entmain/session"
	"github.com/simpledms/simpledms/db/entx"

	"github.com/simpledms/simpledms/action"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	common2 "github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/db/sqlx"
	appmodel "github.com/simpledms/simpledms/core/model/app"
	"github.com/simpledms/simpledms/core/model/common/country"
	"github.com/simpledms/simpledms/core/model/common/language"
	"github.com/simpledms/simpledms/core/model/common/mainrole"
	"github.com/simpledms/simpledms/core/model/common/plan"
	"github.com/simpledms/simpledms/core/model/common/tenantrole"
	systemconfigmodel "github.com/simpledms/simpledms/core/model/systemconfig"
	"github.com/simpledms/simpledms/core/pathx"
	"github.com/simpledms/simpledms/core/pluginx"
	server2 "github.com/simpledms/simpledms/core/server"
	ui2 "github.com/simpledms/simpledms/core/ui"
	"github.com/simpledms/simpledms/core/ui/uix/route"
	"github.com/simpledms/simpledms/core/util/accountutil"
	"github.com/simpledms/simpledms/core/util/cookiex"
	"github.com/simpledms/simpledms/i18n"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
)

type actionTestHarness struct {
	tb        testing.TB
	mainDB    *sqlx.MainDB
	tenantDBs *tenantdbs.TenantDBs
	infra     *common2.Infra
	actions   *action.Actions
	router    *server2.Router
	metaPath  string
	i18n      *i18n.I18n
}

type testS3Config struct {
	client            *minio.Client
	bucketName        string
	disableEncryption bool
}

func newActionTestHarness(t testing.TB) *actionTestHarness {
	return newActionTestHarnessWithSaaS(t, true) // saas mode required for registration
}

func newActionTestHarnessWithSaaS(t testing.TB, isSaaSModeEnabled bool) *actionTestHarness {
	return newActionTestHarnessWithSaaSAndS3Config(t, isSaaSModeEnabled, nil)
}

func newActionTestHarnessWithS3(t testing.TB) *actionTestHarness {
	return newActionTestHarnessWithSaaSAndS3(t, true) // saas mode required for registration
}

func newActionTestHarnessWithS3AndEncryption(t testing.TB, disableEncryption bool) *actionTestHarness {
	s3Config := newTestS3Config(t)
	s3Config.disableEncryption = disableEncryption

	return newActionTestHarnessWithSaaSAndS3Config(t, true, s3Config) // saas mode required for registration
}

func runWithFileEncryptionModes(t *testing.T, testFn func(t *testing.T, disableEncryption bool)) {
	t.Helper()

	testCases := []struct {
		name              string
		disableEncryption bool
	}{
		{name: "encryption-enabled", disableEncryption: false},
		{name: "encryption-disabled", disableEncryption: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testFn(t, tc.disableEncryption)
		})
	}
}

func newActionTestHarnessWithSaaSAndS3(t testing.TB, isSaaSModeEnabled bool) *actionTestHarness {
	s3Config := newTestS3Config(t)
	return newActionTestHarnessWithSaaSAndS3Config(t, isSaaSModeEnabled, s3Config)
}

func newActionTestHarnessWithSaaSAndS3Config(t testing.TB, isSaaSModeEnabled bool, s3Config *testS3Config) *actionTestHarness {
	t.Helper()

	metaPath := t.TempDir()
	migrationsMainFS, err := migratemain.NewMigrationsMainFS()
	if err != nil {
		t.Fatal(err)
	}

	mainDB := dbMigrationsMainDB(true, metaPath, migrationsMainFS)
	t.Cleanup(func() {
		err := mainDB.Close()
		if err != nil {
			t.Fatalf("close main db: %v", err)
		}
	})
	if s3Config != nil {
		t.Cleanup(func() {
			cleanupS3TestObjects(t, mainDB, s3Config)
		})
	}

	publicOrigin := strings.TrimSpace(os.Getenv("SIMPLEDMS_PUBLIC_ORIGIN"))
	webauthnRPID := strings.TrimSpace(os.Getenv("SIMPLEDMS_WEBAUTHN_RP_ID"))
	webauthnRPName := strings.TrimSpace(os.Getenv("SIMPLEDMS_WEBAUTHN_RP_NAME"))

	systemConfig := initSystemConfig(t, mainDB, isSaaSModeEnabled, publicOrigin, webauthnRPID, webauthnRPName)

	templates := template.New("app")
	templates.Funcs(ui2.TemplateFuncMap(templates))
	templates, err = templates.ParseFS(ui2.WidgetFS, "widget/*.gohtml")
	if err != nil {
		t.Fatal(err)
	}

	renderer := ui2.NewRenderer(templates)
	i18nx := i18n.NewI18n()

	fileSystem := filesystem.NewFileSystem(metaPath)
	s3FileSystem := filesystem.NewS3FileSystem(
		nil,
		"",
		fileSystem,
		false,
		filesystem.NewStorageQuota(isSaaSModeEnabled),
	)
	if s3Config != nil {
		s3FileSystem = filesystem.NewS3FileSystem(
			s3Config.client,
			s3Config.bucketName,
			fileSystem,
			s3Config.disableEncryption,
			filesystem.NewStorageQuota(isSaaSModeEnabled),
		)
	}

	infra := common2.NewInfra(
		renderer,
		metaPath,
		s3FileSystem,
		common.NewFactory(),
		common.NewFileRepository(),
		pluginx.NewRegistry(),
		systemConfig,
	)

	tenantDBs := tenantdbs.NewTenantDBs()
	router := server2.NewRouter(mainDB, tenantDBs, infra, true, metaPath, i18nx)
	actions := action.NewActions(infra, tenantDBs, true)
	router.RegisterActions(actions)

	err = infra.PluginRegistry().RegisterActions(router)
	if err != nil {
		t.Fatal(err)
	}

	return &actionTestHarness{
		tb:        t,
		mainDB:    mainDB,
		tenantDBs: tenantDBs,
		infra:     infra,
		actions:   actions,
		router:    router,
		metaPath:  metaPath,
		i18n:      i18nx,
	}
}

func cleanupS3TestObjects(t testing.TB, mainDB *sqlx.MainDB, s3Config *testS3Config) {
	t.Helper()

	if s3Config.client == nil || s3Config.bucketName == "" {
		return
	}

	ctx := context.Background()
	accounts := mainDB.ReadOnlyConn.Account.Query().AllX(ctx)
	tenants := mainDB.ReadOnlyConn.Tenant.Query().AllX(ctx)

	prefixes := make([]string, 0, len(accounts)+len(tenants)*3)
	for _, accountx := range accounts {
		prefixes = append(prefixes, pathx.S3TemporaryAccountStoragePrefix(accountx.PublicID.String()))
	}
	for _, tenantx := range tenants {
		tenantID := tenantx.PublicID.String()
		prefixes = append(prefixes,
			pathx.S3StoragePrefix(tenantID),
			pathx.S3TemporaryStoragePrefix(tenantID),
			pathx.S3SqliteReplicationPrefix(tenantID),
		)
	}

	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		cleanupS3Prefix(t, s3Config.client, s3Config.bucketName, prefix)
	}
}

func cleanupS3Prefix(t testing.TB, client *minio.Client, bucketName, prefix string) {
	t.Helper()

	ctx := context.Background()
	for object := range client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if object.Err != nil {
			t.Logf("list objects %s: %v", prefix, object.Err)
			continue
		}

		if err := client.RemoveObject(ctx, bucketName, object.Key, minio.RemoveObjectOptions{}); err != nil {
			t.Logf("remove object %s: %v", object.Key, err)
		}
	}
}

func newTestS3Config(t testing.TB) *testS3Config {
	t.Helper()

	endpoint := envOrDefault("SIMPLEDMS_S3_ENDPOINT", "localhost:7070")
	accessKey := envOrDefault("SIMPLEDMS_S3_ACCESS_KEY_ID", "unsafe-placeholder-access-key-id")
	secretKey := envOrDefault("SIMPLEDMS_S3_SECRET_ACCESS_KEY", "unsafe-placeholder-secret-access-key")
	bucketName := envOrDefault("SIMPLEDMS_S3_BUCKET_NAME", "simpledms")
	useSSL := envOrDefaultBool("SIMPLEDMS_S3_USE_SSL", false)
	disableEncryption := envOrDefaultBool("SIMPLEDMS_DISABLE_FILE_ENCRYPTION", false)

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		t.Fatalf("init s3 client: %v", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		t.Fatalf("s3 bucket exists: %v", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			t.Fatalf("create s3 bucket: %v", err)
		}
	}

	return &testS3Config{
		client:            client,
		bucketName:        bucketName,
		disableEncryption: disableEncryption,
	}
}

func envOrDefault(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func envOrDefaultBool(key string, fallback bool) bool {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return parsed
}

func initSystemConfig(
	t testing.TB,
	mainDB *sqlx.MainDB,
	isSaaSModeEnabled bool,
	publicOrigin,
	webauthnRPID,
	webauthnRPName string,
) *systemconfigmodel.SystemConfig {
	t.Helper()

	ctx := context.Background()
	tx, err := mainDB.ReadWriteConn.Tx(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = appmodel.InitAppWithoutCustomContext(
		ctx,
		tx,
		"",
		true,
		appmodel.S3Config{},
		appmodel.TLSConfig{},
		appmodel.MailerConfig{},
		appmodel.OCRConfig{},
	)
	if err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	systemConfigx := mainDB.ReadWriteConn.SystemConfig.Query().FirstX(ctx)
	return systemconfigmodel.NewSystemConfig(
		systemConfigx,
		isSaaSModeEnabled,
		false,
		true,
		publicOrigin,
		webauthnRPID,
		webauthnRPName,
	)
}

func createAccount(t testing.TB, mainDB *sqlx.MainDB, email, password string) {
	createAccountWithRole(t, mainDB, email, password, mainrole.User)
}

func createAccountWithRole(
	t testing.TB,
	mainDB *sqlx.MainDB,
	email,
	password string,
	role mainrole.MainRole,
) {
	t.Helper()

	salt, ok := accountutil.RandomSalt()
	if !ok {
		t.Fatal("could not generate salt")
	}

	passwordHash := accountutil.PasswordHash(password, salt)

	mainDB.ReadWriteConn.Account.Create().
		SetEmail(entx.NewCIText(email)).
		SetFirstName("Test").
		SetLastName("User").
		SetLanguage(language.Unknown).
		SetRole(role).
		SetPasswordSalt(salt).
		SetPasswordHash(passwordHash).
		SaveX(context.Background())
}

func signInAndGetSessionCookie(t testing.TB, harness *actionTestHarness, email, password string) *http.Cookie {
	t.Helper()

	ctx := context.Background()
	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(ctx)
	if !accountx.QueryTenants().ExistX(ctx) {
		tenantx := createTenantWithPasskeyPolicy(
			t,
			harness.mainDB,
			"Test Tenant",
			false,
			true,
		)
		assignAccountToTenant(
			t,
			harness.mainDB,
			tenantx.ID,
			accountx.ID,
			tenantrole.Owner,
			false,
		)
	}

	form := url.Values{}
	form.Set("Email", email)
	form.Set("Password", password)

	req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected sign-in status %d, got %d", http.StatusOK, rr.Code)
	}

	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == cookiex.SessionCookieName() {
			return cookie
		}
	}

	t.Fatalf("expected session cookie %q", cookiex.SessionCookieName())
	return nil
}

func createTenantWithPasskeyPolicy(
	t testing.TB,
	mainDB *sqlx.MainDB,
	name string,
	enforcePasskeys bool,
	isInitialized bool,
) *entmain.Tenant {
	t.Helper()

	now := time.Now()
	tenantCreate := mainDB.ReadWriteConn.Tenant.Create().
		SetName(name).
		SetCountry(country.Unknown).
		SetPlan(plan.Unknown).
		SetTermsOfServiceAccepted(now).
		SetPrivacyPolicyAccepted(now).
		SetPasskeyAuthEnforced(enforcePasskeys)
	if isInitialized {
		tenantCreate = tenantCreate.SetInitializedAt(now)
	}

	return tenantCreate.SaveX(context.Background())
}

func assignAccountToTenant(
	t testing.TB,
	mainDB *sqlx.MainDB,
	tenantID int64,
	accountID int64,
	role tenantrole.TenantRole,
	isDefault bool,
) {
	t.Helper()

	tenantAccountAssignmentCreate := mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(tenantID).
		SetAccountID(accountID).
		SetRole(role)
	if isDefault {
		tenantAccountAssignmentCreate = tenantAccountAssignmentCreate.SetIsDefault(true)
	}

	tenantAccountAssignmentCreate.SaveX(context.Background())
}

func TestSignInCmdSetsSessionAndRedirect(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "user@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)
	harness.mainDB.ReadWriteConn.Account.Update().
		Where(account.EmailEQ(entx.NewCIText(email))).
		SetRole(mainrole.Admin).
		ExecX(context.Background())

	form := url.Values{}
	form.Set("Email", email)
	form.Set("Password", password)

	req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if redirect := rr.Header().Get("HX-Redirect"); redirect != route.Dashboard() {
		t.Fatalf("expected redirect %q, got %q", route.Dashboard(), redirect)
	}

	resp := rr.Result()
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie, got none")
	}

	hasSessionCookie := false
	for _, cookie := range cookies {
		if cookie.Name == cookiex.SessionCookieName() {
			hasSessionCookie = true
			break
		}
	}

	if !hasSessionCookie {
		t.Fatalf("expected cookie %q", cookiex.SessionCookieName())
	}

	sessionCount := harness.mainDB.ReadWriteConn.Session.Query().CountX(context.Background())
	if sessionCount != 1 {
		t.Fatalf("expected 1 session, got %d", sessionCount)
	}
}

func TestSignInCmdRejectsWrongPassword(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "wrong-password@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	form := url.Values{}
	form.Set("Email", email)
	form.Set("Password", "not-the-password")

	req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if redirect := rr.Header().Get("HX-Redirect"); redirect != "" {
		t.Fatalf("expected no redirect, got %q", redirect)
	}

	if !strings.Contains(rr.Body.String(), "Invalid credentials. Please try again.") {
		t.Fatalf("expected invalid-credentials hint in response body, got: %s", rr.Body.String())
	}

	sessionCount := harness.mainDB.ReadWriteConn.Session.Query().CountX(context.Background())
	if sessionCount != 0 {
		t.Fatalf("expected 0 sessions, got %d", sessionCount)
	}
}

func TestSignInCmdDoesNotEnumerateMissingAccount(t *testing.T) {
	harness := newActionTestHarness(t)

	form := url.Values{}
	form.Set("Email", "missing-account@example.com")
	form.Set("Password", "not-the-password")

	req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Invalid credentials. Please try again.") {
		t.Fatalf("expected invalid-credentials hint in response body, got: %s", body)
	}
	if strings.Contains(body, "Found no account for this email address.") {
		t.Fatal("response leaked account existence")
	}
}

func TestSignInCmdRateLimitedByIP(t *testing.T) {
	harness := newActionTestHarness(t)

	for qi := 0; qi < 20; qi++ {
		form := url.Values{}
		form.Set("Email", fmt.Sprintf("missing-ip-%d@example.com", qi))
		form.Set("Password", "wrong-password")

		req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.RemoteAddr = "198.51.100.20:1234"

		rr := httptest.NewRecorder()
		harness.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("warm-up request %d expected status %d, got %d", qi, http.StatusOK, rr.Code)
		}
	}

	blockedForm := url.Values{}
	blockedForm.Set("Email", "missing-ip-blocked@example.com")
	blockedForm.Set("Password", "wrong-password")

	blockedReq := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(blockedForm.Encode()))
	blockedReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	blockedReq.Header.Set("HX-Request", "true")
	blockedReq.RemoteAddr = "198.51.100.20:1234"

	blockedRR := httptest.NewRecorder()
	harness.router.ServeHTTP(blockedRR, blockedReq)

	if blockedRR.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, blockedRR.Code)
	}
}

func TestSignInCmdRateLimitedByEmail(t *testing.T) {
	harness := newActionTestHarness(t)

	for qi := 0; qi < 8; qi++ {
		form := url.Values{}
		form.Set("Email", "missing-email-rate-limit@example.com")
		form.Set("Password", "wrong-password")

		req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.RemoteAddr = fmt.Sprintf("198.51.100.%d:1234", qi+1)

		rr := httptest.NewRecorder()
		harness.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("warm-up request %d expected status %d, got %d", qi, http.StatusOK, rr.Code)
		}
	}

	blockedForm := url.Values{}
	blockedForm.Set("Email", "missing-email-rate-limit@example.com")
	blockedForm.Set("Password", "wrong-password")

	blockedReq := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(blockedForm.Encode()))
	blockedReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	blockedReq.Header.Set("HX-Request", "true")
	blockedReq.RemoteAddr = "203.0.113.10:9999"

	blockedRR := httptest.NewRecorder()
	harness.router.ServeHTTP(blockedRR, blockedReq)

	if blockedRR.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, blockedRR.Code)
	}
}

func TestSignInCmdRejectsPasswordWhenPasskeyEnabled(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "passkey-user@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	accountx.Update().SetPasskeyLoginEnabled(true).SaveX(context.Background())

	form := url.Values{}
	form.Set("Email", email)
	form.Set("Password", password)

	req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "Passkey sign-in is required for this account.") {
		t.Fatalf("expected passkey-required hint in response body, got: %s", rr.Body.String())
	}

	sessionCount := harness.mainDB.ReadWriteConn.Session.Query().CountX(context.Background())
	if sessionCount != 0 {
		t.Fatalf("expected 0 sessions, got %d", sessionCount)
	}
}

func TestResetPasswordCmdDoesNotEnumerateMissingAccount(t *testing.T) {
	harness := newActionTestHarness(t)

	createAccount(t, harness.mainDB, "known-account@example.com", "supersecret")

	baseMailCount := harness.mainDB.ReadWriteConn.Mail.Query().CountX(context.Background())

	missingForm := url.Values{}
	missingForm.Set("Email", "missing-account@example.com")

	missingReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/reset-password-cmd",
		strings.NewReader(missingForm.Encode()),
	)
	missingReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	missingReq.Header.Set("HX-Request", "true")

	missingRR := httptest.NewRecorder()
	harness.router.ServeHTTP(missingRR, missingReq)

	if missingRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, missingRR.Code)
	}

	missingBody := missingRR.Body.String()
	if strings.Contains(missingBody, "Account not found") {
		t.Fatal("response leaked account existence")
	}
	if !strings.Contains(missingBody, "If an account with this email exists") {
		t.Fatalf("expected generic confirmation in response body, got: %s", missingBody)
	}

	afterMissingMailCount := harness.mainDB.ReadWriteConn.Mail.Query().CountX(context.Background())
	if afterMissingMailCount != baseMailCount {
		t.Fatalf("expected no new mails for missing account, got %d", afterMissingMailCount-baseMailCount)
	}

	existingForm := url.Values{}
	existingForm.Set("Email", "known-account@example.com")

	existingReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/reset-password-cmd",
		strings.NewReader(existingForm.Encode()),
	)
	existingReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	existingReq.Header.Set("HX-Request", "true")

	existingRR := httptest.NewRecorder()
	harness.router.ServeHTTP(existingRR, existingReq)

	if existingRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, existingRR.Code)
	}

	existingBody := existingRR.Body.String()
	if !strings.Contains(existingBody, "If an account with this email exists") {
		t.Fatalf("expected generic confirmation in response body, got: %s", existingBody)
	}

	afterExistingMailCount := harness.mainDB.ReadWriteConn.Mail.Query().CountX(context.Background())
	if afterExistingMailCount != baseMailCount+1 {
		t.Fatalf("expected 1 new mail for existing account, got %d", afterExistingMailCount-baseMailCount)
	}
}

func TestResetPasswordCmdRateLimitedByIP(t *testing.T) {
	harness := newActionTestHarness(t)

	for qi := 0; qi < 10; qi++ {
		form := url.Values{}
		form.Set("Email", fmt.Sprintf("missing-reset-ip-%d@example.com", qi))

		req := httptest.NewRequest(http.MethodPost, "/-/auth/reset-password-cmd", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.RemoteAddr = "198.51.100.40:1234"

		rr := httptest.NewRecorder()
		harness.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("warm-up request %d expected status %d, got %d", qi, http.StatusOK, rr.Code)
		}
	}

	blockedForm := url.Values{}
	blockedForm.Set("Email", "missing-reset-ip-blocked@example.com")

	blockedReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/reset-password-cmd",
		strings.NewReader(blockedForm.Encode()),
	)
	blockedReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	blockedReq.Header.Set("HX-Request", "true")
	blockedReq.RemoteAddr = "198.51.100.40:1234"

	blockedRR := httptest.NewRecorder()
	harness.router.ServeHTTP(blockedRR, blockedReq)

	if blockedRR.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, blockedRR.Code)
	}
}

func TestResetPasswordCmdRateLimitedByEmail(t *testing.T) {
	harness := newActionTestHarness(t)

	for qi := 0; qi < 3; qi++ {
		form := url.Values{}
		form.Set("Email", "missing-reset-email@example.com")

		req := httptest.NewRequest(http.MethodPost, "/-/auth/reset-password-cmd", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.RemoteAddr = fmt.Sprintf("203.0.113.%d:5678", qi+1)

		rr := httptest.NewRecorder()
		harness.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("warm-up request %d expected status %d, got %d", qi, http.StatusOK, rr.Code)
		}
	}

	blockedForm := url.Values{}
	blockedForm.Set("Email", "missing-reset-email@example.com")

	blockedReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/reset-password-cmd",
		strings.NewReader(blockedForm.Encode()),
	)
	blockedReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	blockedReq.Header.Set("HX-Request", "true")
	blockedReq.RemoteAddr = "203.0.113.200:5678"

	blockedRR := httptest.NewRecorder()
	harness.router.ServeHTTP(blockedRR, blockedReq)

	if blockedRR.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, blockedRR.Code)
	}
}

func TestPasskeySignInFinishCmdRejectsInvalidCredential(t *testing.T) {
	harness := newActionTestHarness(t)

	beginReq := httptest.NewRequest(
		http.MethodPost,
		"http://localhost/-/auth/passkey-sign-in-begin-cmd",
		nil,
	)
	beginReq.Host = "localhost"
	beginReq.Header.Set("HX-Request", "true")

	beginRR := httptest.NewRecorder()
	harness.router.ServeHTTP(beginRR, beginReq)

	if beginRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, beginRR.Code)
	}

	var beginPayload struct {
		ChallengeID string `json:"challengeId"`
	}
	if err := json.Unmarshal(beginRR.Body.Bytes(), &beginPayload); err != nil {
		t.Fatalf("expected valid json response: %v", err)
	}
	if beginPayload.ChallengeID == "" {
		t.Fatal("expected challenge id in response")
	}

	finishReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/passkey-sign-in-finish-cmd",
		strings.NewReader(fmt.Sprintf(`{"challengeId":%q,"credential":{}}`, beginPayload.ChallengeID)),
	)
	finishReq.Header.Set("Content-Type", "application/json")
	finishReq.Header.Set("HX-Request", "true")

	finishRR := httptest.NewRecorder()
	harness.router.ServeHTTP(finishRR, finishReq)

	if finishRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, finishRR.Code)
	}

	replayReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/passkey-sign-in-finish-cmd",
		strings.NewReader(fmt.Sprintf(`{"challengeId":%q,"credential":{}}`, beginPayload.ChallengeID)),
	)
	replayReq.Header.Set("Content-Type", "application/json")
	replayReq.Header.Set("HX-Request", "true")

	replayRR := httptest.NewRecorder()
	harness.router.ServeHTTP(replayRR, replayReq)

	if replayRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected replay status %d, got %d", http.StatusUnauthorized, replayRR.Code)
	}

	sessionCount := harness.mainDB.ReadWriteConn.Session.Query().CountX(context.Background())
	if sessionCount != 0 {
		t.Fatalf("expected 0 sessions, got %d", sessionCount)
	}
}

func TestSignOutCmdInvalidatesSessionAndDeletesSessionRow(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "signout@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	beforeCount := harness.mainDB.ReadWriteConn.Session.Query().
		Where(session.Value(sessionCookie.Value)).
		CountX(context.Background())
	if beforeCount != 1 {
		t.Fatalf("expected 1 session row before sign out, got %d", beforeCount)
	}

	req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-out-cmd", nil)
	req.AddCookie(sessionCookie)
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if redirect := rr.Header().Get("HX-Redirect"); redirect != "/" {
		t.Fatalf("expected redirect %q, got %q", "/", redirect)
	}

	afterCount := harness.mainDB.ReadWriteConn.Session.Query().
		Where(session.Value(sessionCookie.Value)).
		CountX(context.Background())
	if afterCount != 0 {
		t.Fatalf("expected 0 session rows after sign out, got %d", afterCount)
	}
}

func TestRenamePasskeyCmdUpdatesOwnCredential(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "rename-own@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText(email))).
		OnlyX(context.Background())

	credentialx := harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(accountx.ID).
		SetCredentialID([]byte("test-credential-id-rename-own")).
		SetCredentialJSON([]byte(`{"id":"credential-rename-own"}`)).
		SetName("Old Name").
		SaveX(context.Background())

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	form := url.Values{}
	form.Set("PasskeyID", credentialx.PublicID.String())
	form.Set("Name", "New Name")

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/rename-passkey-cmd",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(sessionCookie)

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	renamedCredentialx := harness.mainDB.ReadWriteConn.PasskeyCredential.GetX(context.Background(), credentialx.ID)
	if renamedCredentialx.Name != "New Name" {
		t.Fatalf("expected credential name %q, got %q", "New Name", renamedCredentialx.Name)
	}
}

func TestRenamePasskeyCmdRejectsForeignCredential(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerEmail := "rename-owner@example.com"
	attackerEmail := "rename-attacker@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, ownerEmail, password)
	createAccount(t, harness.mainDB, attackerEmail, password)

	ownerAccountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText(ownerEmail))).
		OnlyX(context.Background())

	ownerCredentialx := harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(ownerAccountx.ID).
		SetCredentialID([]byte("test-credential-id-rename-foreign")).
		SetCredentialJSON([]byte(`{"id":"credential-rename-foreign"}`)).
		SetName("Owner Key").
		SaveX(context.Background())

	attackerSessionCookie := signInAndGetSessionCookie(t, harness, attackerEmail, password)

	form := url.Values{}
	form.Set("PasskeyID", ownerCredentialx.PublicID.String())
	form.Set("Name", "Hacked Name")

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/rename-passkey-cmd",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(attackerSessionCookie)

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	ownerCredentialAfterx := harness.mainDB.ReadWriteConn.PasskeyCredential.GetX(context.Background(), ownerCredentialx.ID)
	if ownerCredentialAfterx.Name != "Owner Key" {
		t.Fatalf("expected credential name %q, got %q", "Owner Key", ownerCredentialAfterx.Name)
	}
}

func TestDeletePasskeyCmdRejectsRemovingLastPasskeyWhenTenantRequiresPasskeys(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "delete-last-required@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText(email))).
		OnlyX(context.Background())
	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	tenantx := createTenantWithPasskeyPolicy(t, harness.mainDB, "Required Tenant", true, false)
	assignAccountToTenant(t, harness.mainDB, tenantx.ID, accountx.ID, tenantrole.Owner, true)

	credentialx := harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(accountx.ID).
		SetCredentialID([]byte("test-credential-id-delete-last-required")).
		SetCredentialJSON([]byte(`{"id":"credential-delete-last-required"}`)).
		SetName("Only Passkey").
		SaveX(context.Background())

	form := url.Values{}
	form.Set("PasskeyID", credentialx.PublicID.String())

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/delete-passkey-cmd",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(sessionCookie)

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	remainingCount := harness.mainDB.ReadWriteConn.PasskeyCredential.Query().
		Where(passkeycredential.AccountID(accountx.ID)).
		CountX(context.Background())
	if remainingCount != 1 {
		t.Fatalf("expected 1 remaining passkey, got %d", remainingCount)
	}
}

func TestClearPasskeysCmdRemovesCredentialsAndDisablesPasskeyLogin(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "clear-passkeys@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText(email))).
		OnlyX(context.Background())

	harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(accountx.ID).
		SetCredentialID([]byte("test-credential-id-clear-1")).
		SetCredentialJSON([]byte(`{"id":"credential-clear-1"}`)).
		SetName("Passkey 1").
		SaveX(context.Background())
	harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(accountx.ID).
		SetCredentialID([]byte("test-credential-id-clear-2")).
		SetCredentialJSON([]byte(`{"id":"credential-clear-2"}`)).
		SetName("Passkey 2").
		SaveX(context.Background())

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	harness.mainDB.ReadWriteConn.Account.UpdateOneID(accountx.ID).
		SetPasskeyLoginEnabled(true).
		SetPasskeyRecoveryCodeSalt("test-salt").
		SetPasskeyRecoveryCodeHashes([]string{"hash-1", "hash-2"}).
		SaveX(context.Background())

	req := httptest.NewRequest(http.MethodPost, "/-/auth/clear-passkeys-cmd", nil)
	req.Header.Set("HX-Request", "true")
	req.AddCookie(sessionCookie)

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	remainingCredentials := harness.mainDB.ReadWriteConn.PasskeyCredential.Query().
		Where(passkeycredential.AccountID(accountx.ID)).
		CountX(context.Background())
	if remainingCredentials != 0 {
		t.Fatalf("expected 0 passkey credentials, got %d", remainingCredentials)
	}

	updatedAccountx := harness.mainDB.ReadWriteConn.Account.GetX(context.Background(), accountx.ID)
	if updatedAccountx.PasskeyLoginEnabled {
		t.Fatal("expected passkey login to be disabled")
	}
	if updatedAccountx.PasskeyRecoveryCodeSalt != "" {
		t.Fatalf("expected empty passkey recovery code salt, got %q", updatedAccountx.PasskeyRecoveryCodeSalt)
	}
	if len(updatedAccountx.PasskeyRecoveryCodeHashes) != 0 {
		t.Fatalf("expected no passkey recovery code hashes, got %d", len(updatedAccountx.PasskeyRecoveryCodeHashes))
	}
}

func TestPasskeyRecoverySignInCmdConsumesRecoveryCode(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "recovery-user@example.com"
	password := "supersecret"
	recoveryCode := "ABCD-2345"
	createAccount(t, harness.mainDB, email, password)

	salt, ok := accountutil.RandomSalt()
	if !ok {
		t.Fatal("could not generate salt")
	}

	normalizedCode := strings.ToUpper(strings.ReplaceAll(recoveryCode, "-", ""))
	recoveryCodeHash := accountutil.PasswordHash(normalizedCode, salt)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	accountx.Update().
		SetPasskeyLoginEnabled(true).
		SetPasskeyRecoveryCodeSalt(salt).
		SetPasskeyRecoveryCodeHashes([]string{recoveryCodeHash}).
		SaveX(context.Background())

	form := url.Values{}
	form.Set("Email", email)
	form.Set("BackupCode", recoveryCode)

	req := httptest.NewRequest(http.MethodPost, "/-/auth/passkey-recovery-sign-in-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if redirect := rr.Header().Get("HX-Redirect"); redirect != route.Dashboard() {
		t.Fatalf("expected redirect %q, got %q", route.Dashboard(), redirect)
	}

	updatedAccountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	if len(updatedAccountx.PasskeyRecoveryCodeHashes) != 0 {
		t.Fatalf("expected recovery code to be consumed, hashes left: %d", len(updatedAccountx.PasskeyRecoveryCodeHashes))
	}

	form2 := url.Values{}
	form2.Set("Email", email)
	form2.Set("BackupCode", recoveryCode)

	req2 := httptest.NewRequest(http.MethodPost, "/-/auth/passkey-recovery-sign-in-cmd", strings.NewReader(form2.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("HX-Request", "true")

	rr2 := httptest.NewRecorder()
	harness.router.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d on second use, got %d", http.StatusTooManyRequests, rr2.Code)
	}
}

func TestPasskeyRecoverySignInCmdRejectsInvalidRecoveryCode(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "invalid-recovery@example.com"
	password := "supersecret"
	recoveryCode := "ABCD-2345"
	createAccount(t, harness.mainDB, email, password)

	salt, ok := accountutil.RandomSalt()
	if !ok {
		t.Fatal("could not generate salt")
	}

	normalizedCode := strings.ToUpper(strings.ReplaceAll(recoveryCode, "-", ""))
	recoveryCodeHash := accountutil.PasswordHash(normalizedCode, salt)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	accountx.Update().
		SetPasskeyLoginEnabled(true).
		SetPasskeyRecoveryCodeSalt(salt).
		SetPasskeyRecoveryCodeHashes([]string{recoveryCodeHash}).
		SaveX(context.Background())

	form := url.Values{}
	form.Set("Email", email)
	form.Set("BackupCode", "WRNG-0000")

	req := httptest.NewRequest(http.MethodPost, "/-/auth/passkey-recovery-sign-in-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}

	sessionCount := harness.mainDB.ReadWriteConn.Session.Query().CountX(context.Background())
	if sessionCount != 0 {
		t.Fatalf("expected 0 sessions, got %d", sessionCount)
	}
}

func TestSignInCmdAllowsBootstrapWhenTenantRequiresPasskeyAndNoPasskeyExists(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "tenant-passkey-user@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())

	tenantRequired := createTenantWithPasskeyPolicy(t, harness.mainDB, "Required Tenant", true, false)
	tenantOptional := createTenantWithPasskeyPolicy(t, harness.mainDB, "Optional Tenant", false, false)

	assignAccountToTenant(t, harness.mainDB, tenantRequired.ID, accountx.ID, tenantrole.Owner, true)
	assignAccountToTenant(t, harness.mainDB, tenantOptional.ID, accountx.ID, tenantrole.Owner, false)

	form := url.Values{}
	form.Set("Email", email)
	form.Set("Password", password)

	req := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if redirect := rr.Header().Get("HX-Redirect"); redirect != route.Dashboard() {
		t.Fatalf("expected redirect %q, got %q", route.Dashboard(), redirect)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == cookiex.SessionCookieName() {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected session cookie %q", cookiex.SessionCookieName())
	}

	sessionx := harness.mainDB.ReadWriteConn.Session.Query().Where(
		session.Value(sessionCookie.Value),
	).OnlyX(context.Background())
	if !sessionx.IsTemporarySession {
		t.Fatal("expected bootstrap sign-in to create temporary setup session")
	}
}

func TestTemporarySetupSessionExpiresAtDeletableAt(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "expired-setup-session@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())

	tenantRequired := createTenantWithPasskeyPolicy(t, harness.mainDB, "Required Tenant", true, false)
	assignAccountToTenant(t, harness.mainDB, tenantRequired.ID, accountx.ID, tenantrole.Owner, true)

	form := url.Values{}
	form.Set("Email", email)
	form.Set("Password", password)

	signInReq := httptest.NewRequest(http.MethodPost, "/-/auth/sign-in-cmd", strings.NewReader(form.Encode()))
	signInReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	signInReq.Header.Set("HX-Request", "true")

	signInRR := httptest.NewRecorder()
	harness.router.ServeHTTP(signInRR, signInReq)

	if signInRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, signInRR.Code)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range signInRR.Result().Cookies() {
		if cookie.Name == cookiex.SessionCookieName() {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected session cookie %q", cookiex.SessionCookieName())
	}

	harness.mainDB.ReadWriteConn.Session.Update().
		Where(session.Value(sessionCookie.Value)).
		SetDeletableAt(time.Now().Add(-time.Minute)).
		ExecX(context.Background())

	req := httptest.NewRequest(http.MethodPost, "/-/dashboard/dashboard-cards-partial", nil)
	req.AddCookie(sessionCookie)
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}
	if location := rr.Header().Get("Location"); location != "/" {
		t.Fatalf("expected redirect location %q, got %q", "/", location)
	}
}

func TestSetupSessionBlocksNonPasskeyPathsUntilEnrollment(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "tenant-passkey-setup-restricted@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())

	tenantRequired := createTenantWithPasskeyPolicy(t, harness.mainDB, "Required Tenant", true, false)
	assignAccountToTenant(t, harness.mainDB, tenantRequired.ID, accountx.ID, tenantrole.Owner, true)

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	blockedReq := httptest.NewRequest(http.MethodPost, "/-/dashboard/toggle-tenant-passkey-enforcement-cmd", nil)
	blockedReq.AddCookie(sessionCookie)
	blockedReq.Header.Set("HX-Request", "true")

	blockedRR := httptest.NewRecorder()
	harness.router.ServeHTTP(blockedRR, blockedReq)

	if blockedRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, blockedRR.Code)
	}
	if redirect := blockedRR.Header().Get("HX-Redirect"); redirect != route.Dashboard() {
		t.Fatalf("expected redirect %q, got %q", route.Dashboard(), redirect)
	}

	allowedReq := httptest.NewRequest(http.MethodPost, "/-/auth/passkey-register-begin-cmd", nil)
	allowedReq.AddCookie(sessionCookie)
	allowedReq.Header.Set("HX-Request", "true")

	allowedRR := httptest.NewRecorder()
	harness.router.ServeHTTP(allowedRR, allowedReq)

	if allowedRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, allowedRR.Code)
	}

	dashboardCardsReq := httptest.NewRequest(http.MethodPost, "/-/dashboard/dashboard-cards-partial", nil)
	dashboardCardsReq.AddCookie(sessionCookie)
	dashboardCardsReq.Header.Set("HX-Request", "true")

	dashboardCardsRR := httptest.NewRecorder()
	harness.router.ServeHTTP(dashboardCardsRR, dashboardCardsReq)

	if dashboardCardsRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, dashboardCardsRR.Code)
	}

	body := dashboardCardsRR.Body.String()
	if !strings.Contains(body, "Passkey setup required") {
		t.Fatalf("expected passkey setup card headline, body was: %s", body)
	}
	if !strings.Contains(body, "Register passkey") {
		t.Fatalf("expected register passkey action on limited dashboard, body was: %s", body)
	}
	if !strings.Contains(body, "Your organization requires passkey sign-in. Register a passkey to continue.") {
		t.Fatalf("expected setup explanation on passkey card, body was: %s", body)
	}
	if strings.Contains(body, "Delete account") {
		t.Fatalf("expected limited dashboard to hide account management actions, body was: %s", body)
	}
}

func TestToggleTenantPasskeyEnforcementCmdUpdatesTenantPolicyForOwner(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "tenant-owner@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())

	tenantx := createTenantWithPasskeyPolicy(t, harness.mainDB, "Owner Tenant", false, true)
	assignAccountToTenant(t, harness.mainDB, tenantx.ID, accountx.ID, tenantrole.Owner, true)

	initTenantDB(t, harness, tenantx)

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	form := url.Values{}
	form.Set("TenantID", tenantx.PublicID.String())
	form.Set("EnforcePasskeys", "true")

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/dashboard/toggle-tenant-passkey-enforcement-cmd",
		strings.NewReader(form.Encode()),
	)
	req.AddCookie(sessionCookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if trigger := rr.Header().Get("HX-Trigger"); !strings.Contains(trigger, "accountUpdated") {
		t.Fatalf("expected HX-Trigger to include accountUpdated, got %q", trigger)
	}

	updatedTenantx := harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)
	if !updatedTenantx.PasskeyAuthEnforced {
		t.Fatal("expected tenant passkey enforcement to be enabled")
	}
}

func TestToggleTenantPasskeyEnforcementCmdRejectsNonOwner(t *testing.T) {
	harness := newActionTestHarness(t)

	ownerEmail := "owner@example.com"
	memberEmail := "member@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, ownerEmail, password)
	createAccount(t, harness.mainDB, memberEmail, password)

	ownerAccountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(ownerEmail))).OnlyX(context.Background())
	memberAccountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(memberEmail))).OnlyX(context.Background())

	tenantx := createTenantWithPasskeyPolicy(t, harness.mainDB, "Shared Tenant", false, false)

	assignAccountToTenant(t, harness.mainDB, tenantx.ID, ownerAccountx.ID, tenantrole.Owner, true)
	assignAccountToTenant(t, harness.mainDB, tenantx.ID, memberAccountx.ID, tenantrole.User, false)

	sessionCookie := signInAndGetSessionCookie(t, harness, memberEmail, password)

	form := url.Values{}
	form.Set("TenantID", tenantx.PublicID.String())
	form.Set("EnforcePasskeys", "true")

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/dashboard/toggle-tenant-passkey-enforcement-cmd",
		strings.NewReader(form.Encode()),
	)
	req.AddCookie(sessionCookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	updatedTenantx := harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)
	if updatedTenantx.PasskeyAuthEnforced {
		t.Fatal("expected tenant passkey enforcement to stay disabled")
	}
}

func TestDashboardCardsPartialShowsPasskeyEnforcementButtonWithConfirm(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "tenant-owner-passkey-button@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())

	tenantx := createTenantWithPasskeyPolicy(t, harness.mainDB, "Owner Tenant", false, true)
	assignAccountToTenant(t, harness.mainDB, tenantx.ID, accountx.ID, tenantrole.Owner, true)

	initTenantDB(t, harness, tenantx)

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	fetchCards := func() string {
		req := httptest.NewRequest(http.MethodPost, "/-/dashboard/dashboard-cards-partial", nil)
		req.AddCookie(sessionCookie)
		req.Header.Set("HX-Request", "true")

		rr := httptest.NewRecorder()
		harness.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		return rr.Body.String()
	}

	bodyDisabled := fetchCards()
	if !strings.Contains(bodyDisabled, "Enable passkey enforcement") {
		t.Fatalf("expected enable passkey enforcement button, body was: %s", bodyDisabled)
	}
	if !strings.Contains(
		bodyDisabled,
		"Enable passkey enforcement for this organization? Members will need passkeys to sign in.",
	) {
		t.Fatalf("expected enable confirm text, body was: %s", bodyDisabled)
	}

	form := url.Values{}
	form.Set("TenantID", tenantx.PublicID.String())
	form.Set("EnforcePasskeys", "true")

	updateReq := httptest.NewRequest(
		http.MethodPost,
		"/-/dashboard/toggle-tenant-passkey-enforcement-cmd",
		strings.NewReader(form.Encode()),
	)
	updateReq.AddCookie(sessionCookie)
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateReq.Header.Set("HX-Request", "true")

	updateRR := httptest.NewRecorder()
	harness.router.ServeHTTP(updateRR, updateReq)

	if updateRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, updateRR.Code)
	}

	harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(accountx.ID).
		SetCredentialID([]byte("test-credential-id-dashboard-toggle")).
		SetCredentialJSON([]byte(`{"id":"dashboard-toggle"}`)).
		SetName("Browser Passkey").
		SaveX(context.Background())

	bodyEnabled := fetchCards()
	if !strings.Contains(bodyEnabled, "Disable passkey enforcement") {
		t.Fatalf("expected disable passkey enforcement button, body was: %s", bodyEnabled)
	}
	if !strings.Contains(
		bodyEnabled,
		"Disable passkey enforcement for this organization? Members can use passwords again if allowed.",
	) {
		t.Fatalf("expected disable confirm text, body was: %s", bodyEnabled)
	}
}

func TestPasskeyRegisterBeginCmdRedirectsWhenUnauthenticated(t *testing.T) {
	harness := newActionTestHarness(t)

	unauthenticatedReq := httptest.NewRequest(http.MethodPost, "/-/auth/passkey-register-begin-cmd", nil)
	unauthenticatedReq.Header.Set("HX-Request", "true")

	unauthenticatedRR := httptest.NewRecorder()
	harness.router.ServeHTTP(unauthenticatedRR, unauthenticatedReq)

	if unauthenticatedRR.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, unauthenticatedRR.Code)
	}
}

func TestPasskeyRegisterBeginCmdAllowsAuthenticatedSession(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "register-passkey-user@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	authenticatedReq := httptest.NewRequest(http.MethodPost, "/-/auth/passkey-register-begin-cmd", nil)
	authenticatedReq.AddCookie(sessionCookie)
	authenticatedReq.Header.Set("HX-Request", "true")

	authenticatedRR := httptest.NewRecorder()
	harness.router.ServeHTTP(authenticatedRR, authenticatedReq)

	if authenticatedRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, authenticatedRR.Code)
	}

	var payload struct {
		ChallengeID string `json:"challengeId"`
		Options     struct {
			Response struct {
				User struct {
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
				} `json:"user"`
			} `json:"publicKey"`
		} `json:"options"`
	}

	err := json.Unmarshal(authenticatedRR.Body.Bytes(), &payload)
	if err != nil {
		t.Fatalf("expected valid response json: %v", err)
	}

	if payload.ChallengeID == "" {
		t.Fatal("expected challenge id in response")
	}

	if payload.Options.Response.User.Name != "Test User" {
		t.Fatalf("expected passkey user name %q, got %q", "Test User", payload.Options.Response.User.Name)
	}

	if payload.Options.Response.User.DisplayName != "Test User" {
		t.Fatalf(
			"expected passkey user display name %q, got %q",
			"Test User",
			payload.Options.Response.User.DisplayName,
		)
	}
}

func TestAdminPasskeyRecoveryCmdRequiresPrivilegedRole(t *testing.T) {
	harness := newActionTestHarness(t)

	createAccountWithRole(t, harness.mainDB, "user@example.com", "supersecret", mainrole.User)
	createAccount(t, harness.mainDB, "target@example.com", "supersecret")

	targetAccountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText("target@example.com"))).
		OnlyX(context.Background())

	harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(targetAccountx.ID).
		SetCredentialID([]byte("test-credential-id-1")).
		SetCredentialJSON([]byte(`{"id":"credential-1"}`)).
		SetName("Browser Passkey").
		SaveX(context.Background())

	sessionCookie := signInAndGetSessionCookie(t, harness, "user@example.com", "supersecret")

	form := url.Values{}
	form.Set("Email", "target@example.com")

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/admin-passkey-recovery-cmd",
		strings.NewReader(form.Encode()),
	)
	req.AddCookie(sessionCookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestAdminPasskeyRecoveryCmdReturnsCodesForAdmin(t *testing.T) {
	harness := newActionTestHarness(t)

	createAccountWithRole(t, harness.mainDB, "admin@example.com", "supersecret", mainrole.Admin)
	createAccount(t, harness.mainDB, "target-admin-recovery@example.com", "supersecret")

	targetAccountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText("target-admin-recovery@example.com"))).
		OnlyX(context.Background())

	adminAccountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText("admin@example.com"))).
		OnlyX(context.Background())

	sharedTenantx := createTenantWithPasskeyPolicy(t, harness.mainDB, "Admin Recovery Tenant", false, false)
	assignAccountToTenant(t, harness.mainDB, sharedTenantx.ID, adminAccountx.ID, tenantrole.Owner, true)
	harness.mainDB.ReadWriteConn.TenantAccountAssignment.Create().
		SetTenantID(sharedTenantx.ID).
		SetAccountID(targetAccountx.ID).
		SetRole(tenantrole.Owner).
		SetIsOwningTenant(true).
		SaveX(context.Background())

	harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(targetAccountx.ID).
		SetCredentialID([]byte("test-credential-id-2")).
		SetCredentialJSON([]byte(`{"id":"credential-2"}`)).
		SetName("Phone Passkey").
		SaveX(context.Background())

	sessionCookie := signInAndGetSessionCookie(t, harness, "admin@example.com", "supersecret")

	form := url.Values{}
	form.Set("Email", "target-admin-recovery@example.com")

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/admin-passkey-recovery-cmd",
		strings.NewReader(form.Encode()),
	)
	req.AddCookie(sessionCookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var payload struct {
		RecoveryCodes []string `json:"recoveryCodes"`
	}

	err := json.Unmarshal(rr.Body.Bytes(), &payload)
	if err != nil {
		t.Fatalf("expected valid json response: %v", err)
	}

	if len(payload.RecoveryCodes) != 10 {
		t.Fatalf("expected 10 recovery codes, got %d", len(payload.RecoveryCodes))
	}

	recoveryCodeRegex := regexp.MustCompile(
		`^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{5}(-[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{5}){3}$`,
	)
	for _, recoveryCode := range payload.RecoveryCodes {
		if !recoveryCodeRegex.MatchString(recoveryCode) {
			t.Fatalf("expected strong recovery code format, got %q", recoveryCode)
		}
	}

	updatedAccountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText("target-admin-recovery@example.com"))).
		OnlyX(context.Background())

	if len(updatedAccountx.PasskeyRecoveryCodeHashes) != 10 {
		t.Fatalf("expected 10 stored recovery code hashes, got %d", len(updatedAccountx.PasskeyRecoveryCodeHashes))
	}
}

func TestPasskeySignInBeginCmdRateLimited(t *testing.T) {
	harness := newActionTestHarness(t)

	now := time.Now()
	for qi := 0; qi < 40; qi++ {
		harness.mainDB.ReadWriteConn.WebAuthnChallenge.Create().
			SetChallengeID(fmt.Sprintf("challenge-%d", qi)).
			SetClientKey("192.0.2.1").
			SetCeremony("authentication").
			SetSessionDataJSON([]byte(`{"challenge":"x"}`)).
			SetExpiresAt(now.Add(5 * time.Minute)).
			SetCreatedAt(now).
			SaveX(context.Background())
	}

	req := httptest.NewRequest(http.MethodPost, "/-/auth/passkey-sign-in-begin-cmd", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, rr.Code)
	}
}

func TestPasskeySignInBeginCmdRateLimitScopedByClient(t *testing.T) {
	t.Setenv("SIMPLEDMS_PUBLIC_ORIGIN", "http://localhost")
	t.Setenv("SIMPLEDMS_WEBAUTHN_RP_ID", "localhost")

	harness := newActionTestHarness(t)

	for qi := 0; qi < 40; qi++ {
		attackerReq := httptest.NewRequest(http.MethodPost, "http://localhost/-/auth/passkey-sign-in-begin-cmd", nil)
		attackerReq.Host = "localhost"
		attackerReq.RemoteAddr = "198.51.100.10:1234"
		attackerReq.Header.Set("HX-Request", "true")

		attackerRR := httptest.NewRecorder()
		harness.router.ServeHTTP(attackerRR, attackerReq)

		if attackerRR.Code != http.StatusOK {
			t.Fatalf("warm-up request %d expected status %d, got %d", qi, http.StatusOK, attackerRR.Code)
		}
	}

	victimReq := httptest.NewRequest(http.MethodPost, "http://localhost/-/auth/passkey-sign-in-begin-cmd", nil)
	victimReq.Host = "localhost"
	victimReq.RemoteAddr = "203.0.113.10:5678"
	victimReq.Header.Set("HX-Request", "true")

	victimRR := httptest.NewRecorder()
	harness.router.ServeHTTP(victimRR, victimReq)

	if victimRR.Code != http.StatusOK {
		t.Fatalf("expected status %d for a different client, got %d", http.StatusOK, victimRR.Code)
	}
}

func TestPasskeySignInBeginCmdRateLimitedGlobally(t *testing.T) {
	t.Setenv("SIMPLEDMS_PUBLIC_ORIGIN", "http://localhost")
	t.Setenv("SIMPLEDMS_WEBAUTHN_RP_ID", "localhost")

	harness := newActionTestHarness(t)

	for qi := 0; qi < 120; qi++ {
		req := httptest.NewRequest(http.MethodPost, "http://localhost/-/auth/passkey-sign-in-begin-cmd", nil)
		req.Host = "localhost"
		req.Header.Set("HX-Request", "true")
		req.RemoteAddr = fmt.Sprintf("198.51.100.%d:4321", qi+1)

		rr := httptest.NewRecorder()
		harness.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("warm-up request %d expected status %d, got %d", qi, http.StatusOK, rr.Code)
		}
	}

	blockedReq := httptest.NewRequest(http.MethodPost, "http://localhost/-/auth/passkey-sign-in-begin-cmd", nil)
	blockedReq.Host = "localhost"
	blockedReq.Header.Set("HX-Request", "true")
	blockedReq.RemoteAddr = "203.0.113.10:5432"

	blockedRR := httptest.NewRecorder()
	harness.router.ServeHTTP(blockedRR, blockedReq)

	if blockedRR.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, blockedRR.Code)
	}
}

func TestPasskeyRecoverySignInCmdRateLimited(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "rate-limited-recovery@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	accountx.Update().SetPasskeyLoginEnabled(true).SaveX(context.Background())

	firstReqForm := url.Values{}
	firstReqForm.Set("Email", email)
	firstReqForm.Set("BackupCode", "WRNG-0000")

	firstReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/passkey-recovery-sign-in-cmd",
		strings.NewReader(firstReqForm.Encode()),
	)
	firstReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	firstReq.Header.Set("HX-Request", "true")

	firstRR := httptest.NewRecorder()
	harness.router.ServeHTTP(firstRR, firstReq)

	if firstRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected first attempt status %d, got %d", http.StatusUnauthorized, firstRR.Code)
	}

	secondReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/passkey-recovery-sign-in-cmd",
		strings.NewReader(firstReqForm.Encode()),
	)
	secondReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	secondReq.Header.Set("HX-Request", "true")

	secondRR := httptest.NewRecorder()
	harness.router.ServeHTTP(secondRR, secondReq)

	if secondRR.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second attempt status %d, got %d", http.StatusTooManyRequests, secondRR.Code)
	}
}

func TestPasskeyRecoverySignInCmdDoesNotEnumerateAccounts(t *testing.T) {
	harness := newActionTestHarness(t)

	createAccount(t, harness.mainDB, "regular-user@example.com", "supersecret")
	createAccount(t, harness.mainDB, "passkey-user@example.com", "supersecret")

	passkeyAccountx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.Email(entx.NewCIText("passkey-user@example.com"))).
		OnlyX(context.Background())
	passkeyAccountx.Update().SetPasskeyLoginEnabled(true).SaveX(context.Background())

	testCases := []struct {
		name  string
		email string
	}{
		{name: "missing-account", email: "missing@example.com"},
		{name: "password-only-account", email: "regular-user@example.com"},
		{name: "wrong-code", email: "passkey-user@example.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("Email", tc.email)
			form.Set("BackupCode", "WRNG-0000")

			req := httptest.NewRequest(
				http.MethodPost,
				"/-/auth/passkey-recovery-sign-in-cmd",
				strings.NewReader(form.Encode()),
			)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")

			rr := httptest.NewRecorder()
			harness.router.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
			}

			body := rr.Body.String()
			if strings.Contains(body, "Account not found") {
				t.Fatal("response leaked account existence")
			}
			if strings.Contains(body, "only available for passkey-enabled accounts") {
				t.Fatal("response leaked passkey enrollment state")
			}
		})
	}
}

/* commented because Redirect was disabled
func TestPasskeySignInBeginCmdRedirectsToCanonicalHost(t *testing.T) {
	t.Setenv("SIMPLEDMS_PUBLIC_ORIGIN", "https://app.simpledms.eu")

	harness := newActionTestHarness(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"http://app.simpledms.ch/-/auth/passkey-sign-in-begin-cmd",
		nil,
	)
	req.Host = "app.simpledms.ch"

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "https://app.simpledms.eu/-/auth/passkey-sign-in-begin-cmd" {
		t.Fatalf("expected location %q, got %q", "https://app.simpledms.eu/-/auth/passkey-sign-in-begin-cmd", location)
	}
}
*/

func TestPasskeySignInFinishCmdRejectsUsedChallenge(t *testing.T) {
	harness := newActionTestHarness(t)

	now := time.Now()
	harness.mainDB.ReadWriteConn.WebAuthnChallenge.Create().
		SetChallengeID("used-challenge").
		SetCeremony("authentication").
		SetSessionDataJSON([]byte(`{}`)).
		SetExpiresAt(now.Add(5 * time.Minute)).
		SetUsedAt(now).
		SaveX(context.Background())

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/passkey-sign-in-finish-cmd",
		strings.NewReader(`{"challengeId":"used-challenge","credential":{}}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestRegeneratePasskeyCodesCmdSetsNoStoreCacheHeader(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "regen-cache@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(accountx.ID).
		SetCredentialID([]byte("test-credential-id-cache")).
		SetCredentialJSON([]byte(`{"id":"credential-cache"}`)).
		SetName("Laptop Passkey").
		SaveX(context.Background())

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	req := httptest.NewRequest(http.MethodPost, "/-/auth/regenerate-passkey-codes-cmd", nil)
	req.AddCookie(sessionCookie)
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	cacheControl := rr.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-store") {
		t.Fatalf("expected Cache-Control header to contain %q, got %q", "no-store", cacheControl)
	}
}

func TestRegeneratePasskeyCodesCmdReplacesStoredCodes(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "regen-replace@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(accountx.ID).
		SetCredentialID([]byte("test-credential-id-regen-replace")).
		SetCredentialJSON([]byte(`{"id":"credential-regen-replace"}`)).
		SetName("Laptop Passkey").
		SaveX(context.Background())

	harness.mainDB.ReadWriteConn.Account.UpdateOneID(accountx.ID).
		SetPasskeyRecoveryCodeSalt("old-salt").
		SetPasskeyRecoveryCodeHashes([]string{"old-hash-1", "old-hash-2"}).
		SaveX(context.Background())

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	req := httptest.NewRequest(http.MethodPost, "/-/auth/regenerate-passkey-codes-cmd", nil)
	req.AddCookie(sessionCookie)
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var payload struct {
		RecoveryCodesToken string `json:"recoveryCodesToken"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response: %v", err)
	}
	if payload.RecoveryCodesToken == "" {
		t.Fatal("expected recovery codes token")
	}

	updatedAccountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	if updatedAccountx.PasskeyRecoveryCodeSalt == "" || updatedAccountx.PasskeyRecoveryCodeSalt == "old-salt" {
		t.Fatalf("expected regenerated recovery code salt, got %q", updatedAccountx.PasskeyRecoveryCodeSalt)
	}
	if len(updatedAccountx.PasskeyRecoveryCodeHashes) != 10 {
		t.Fatalf("expected 10 stored recovery code hashes, got %d", len(updatedAccountx.PasskeyRecoveryCodeHashes))
	}
	if strings.Join(updatedAccountx.PasskeyRecoveryCodeHashes, ",") == "old-hash-1,old-hash-2" {
		t.Fatal("expected regenerated recovery code hashes to differ from previous hashes")
	}
}

func TestPasskeyRecoverySignInCmdAcceptsRegeneratedRecoveryCode(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "regen-signin@example.com"
	password := "supersecret"
	oldRecoveryCode := "OLD1-2345"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(accountx.ID).
		SetCredentialID([]byte("test-credential-id-regen-signin")).
		SetCredentialJSON([]byte(`{"id":"credential-regen-signin"}`)).
		SetName("Phone Passkey").
		SaveX(context.Background())

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	oldSalt, ok := accountutil.RandomSalt()
	if !ok {
		t.Fatal("could not generate salt")
	}
	oldHash := accountutil.PasswordHash(strings.ReplaceAll(oldRecoveryCode, "-", ""), oldSalt)
	harness.mainDB.ReadWriteConn.Account.UpdateOneID(accountx.ID).
		SetPasskeyLoginEnabled(true).
		SetPasskeyRecoveryCodeSalt(oldSalt).
		SetPasskeyRecoveryCodeHashes([]string{oldHash}).
		SaveX(context.Background())

	regenReq := httptest.NewRequest(http.MethodPost, "/-/auth/regenerate-passkey-codes-cmd", nil)
	regenReq.AddCookie(sessionCookie)
	regenReq.Header.Set("HX-Request", "true")

	regenRR := httptest.NewRecorder()
	harness.router.ServeHTTP(regenRR, regenReq)

	if regenRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, regenRR.Code)
	}

	var regenPayload struct {
		RecoveryCodesToken string `json:"recoveryCodesToken"`
	}
	if err := json.Unmarshal(regenRR.Body.Bytes(), &regenPayload); err != nil {
		t.Fatalf("expected valid json response: %v", err)
	}
	if regenPayload.RecoveryCodesToken == "" {
		t.Fatal("expected recovery codes token")
	}

	dialogForm := url.Values{}
	dialogForm.Set("Token", regenPayload.RecoveryCodesToken)

	dialogReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/passkey-recovery-codes-dialog",
		strings.NewReader(dialogForm.Encode()),
	)
	dialogReq.AddCookie(sessionCookie)
	dialogReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dialogReq.Header.Set("HX-Request", "true")

	dialogRR := httptest.NewRecorder()
	harness.router.ServeHTTP(dialogRR, dialogReq)

	if dialogRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, dialogRR.Code)
	}

	textareaRegex := regexp.MustCompile(`(?s)<textarea[^>]*>(.*?)</textarea>`)
	textareaMatches := textareaRegex.FindStringSubmatch(dialogRR.Body.String())
	if len(textareaMatches) < 2 {
		t.Fatal("expected recovery codes textarea in dialog response")
	}

	recoveryCodesText := htmlstd.UnescapeString(textareaMatches[1])
	lines := strings.Split(recoveryCodesText, "\n")
	recoveryCodeRegex := regexp.MustCompile(
		`^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{5}(-[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{5}){3}$`,
	)
	regeneratedRecoveryCode := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if recoveryCodeRegex.MatchString(trimmed) {
			regeneratedRecoveryCode = trimmed
			break
		}
	}
	if regeneratedRecoveryCode == "" {
		t.Fatal("expected at least one regenerated recovery code")
	}

	form := url.Values{}
	form.Set("Email", email)
	form.Set("BackupCode", regeneratedRecoveryCode)

	signInReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/passkey-recovery-sign-in-cmd",
		strings.NewReader(form.Encode()),
	)
	signInReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	signInReq.Header.Set("HX-Request", "true")

	signInRR := httptest.NewRecorder()
	harness.router.ServeHTTP(signInRR, signInReq)

	if signInRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, signInRR.Code)
	}

	if redirect := signInRR.Header().Get("HX-Redirect"); redirect != route.Dashboard() {
		t.Fatalf("expected redirect %q, got %q", route.Dashboard(), redirect)
	}
}

func TestDashboardCardsPartialShowsRemainingRecoveryCodesCount(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "recovery-count@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(accountx.ID).
		SetCredentialID([]byte("test-credential-id-recovery-count")).
		SetCredentialJSON([]byte(`{"id":"credential-recovery-count"}`)).
		SetName("Laptop Passkey").
		SaveX(context.Background())

	harness.mainDB.ReadWriteConn.Account.UpdateOneID(accountx.ID).
		SetPasskeyRecoveryCodeSalt("test-salt").
		SetPasskeyRecoveryCodeHashes([]string{"a", "b", "c", "d", "e", "f", "g"}).
		SaveX(context.Background())

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	req := httptest.NewRequest(http.MethodPost, "/-/dashboard/dashboard-cards-partial", nil)
	req.AddCookie(sessionCookie)
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "7 backup codes left") {
		t.Fatalf("expected dashboard to include remaining backup codes count, body was: %s", body)
	}
}

func TestPasskeyRecoveryCodesDialogConsumesToken(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "recovery-dialog@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	accountx := harness.mainDB.ReadWriteConn.Account.Query().Where(account.Email(entx.NewCIText(email))).OnlyX(context.Background())
	harness.mainDB.ReadWriteConn.PasskeyCredential.Create().
		SetAccountID(accountx.ID).
		SetCredentialID([]byte("test-credential-id-dialog")).
		SetCredentialJSON([]byte(`{"id":"credential-dialog"}`)).
		SetName("Tablet Passkey").
		SaveX(context.Background())

	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	req := httptest.NewRequest(http.MethodPost, "/-/auth/regenerate-passkey-codes-cmd", nil)
	req.AddCookie(sessionCookie)
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var payload struct {
		RecoveryCodesToken string `json:"recoveryCodesToken"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid json response: %v", err)
	}
	if payload.RecoveryCodesToken == "" {
		t.Fatal("expected recovery codes token")
	}

	form := url.Values{}
	form.Set("Token", payload.RecoveryCodesToken)

	dialogReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/passkey-recovery-codes-dialog",
		strings.NewReader(form.Encode()),
	)
	dialogReq.AddCookie(sessionCookie)
	dialogReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	dialogReq.Header.Set("HX-Request", "true")

	dialogRR := httptest.NewRecorder()
	harness.router.ServeHTTP(dialogRR, dialogReq)

	if dialogRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, dialogRR.Code)
	}

	secondDialogReq := httptest.NewRequest(
		http.MethodPost,
		"/-/auth/passkey-recovery-codes-dialog",
		strings.NewReader(form.Encode()),
	)
	secondDialogReq.AddCookie(sessionCookie)
	secondDialogReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	secondDialogReq.Header.Set("HX-Request", "true")

	secondDialogRR := httptest.NewRecorder()
	harness.router.ServeHTTP(secondDialogRR, secondDialogReq)

	if secondDialogRR.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, secondDialogRR.Code)
	}
}

/* commented because Redirect was disabled
func TestCanonicalHostRedirectForAuthRoutes(t *testing.T) {
	t.Setenv("SIMPLEDMS_PUBLIC_ORIGIN", "https://app.simpledms.eu")

	harness := newActionTestHarness(t)

	form := url.Values{}
	form.Set("Email", "user@example.com")
	form.Set("Password", "supersecret")

	req := httptest.NewRequest(
		http.MethodPost,
		"http://app.simpledms.ch/-/auth/sign-in-cmd",
		strings.NewReader(form.Encode()),
	)
	req.Host = "app.simpledms.ch"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "https://app.simpledms.eu/-/auth/sign-in-cmd" {
		t.Fatalf("expected location %q, got %q", "https://app.simpledms.eu/-/auth/sign-in-cmd", location)
	}
}
*/
