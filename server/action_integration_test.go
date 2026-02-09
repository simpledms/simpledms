package server

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/simpledms/simpledms/action"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	migratemain "github.com/simpledms/simpledms/db/entmain/migrate"
	_ "github.com/simpledms/simpledms/db/entmain/runtime"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/i18n"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/common/mainrole"
	"github.com/simpledms/simpledms/model/filesystem"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/pathx"
	"github.com/simpledms/simpledms/pluginx"
	"github.com/simpledms/simpledms/ui"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/accountutil"
	"github.com/simpledms/simpledms/util/cookiex"
)

type actionTestHarness struct {
	t         *testing.T
	mainDB    *sqlx.MainDB
	tenantDBs *tenantdbs.TenantDBs
	infra     *common.Infra
	actions   *action.Actions
	router    *Router
	metaPath  string
	i18n      *i18n.I18n
}

type testS3Config struct {
	client            *minio.Client
	bucketName        string
	disableEncryption bool
}

func newActionTestHarness(t *testing.T) *actionTestHarness {
	return newActionTestHarnessWithSaaS(t, true) // saas mode required for registration
}

func newActionTestHarnessWithSaaS(t *testing.T, isSaaSModeEnabled bool) *actionTestHarness {
	return newActionTestHarnessWithSaaSAndS3Config(t, isSaaSModeEnabled, nil)
}

func newActionTestHarnessWithS3(t *testing.T) *actionTestHarness {
	return newActionTestHarnessWithSaaSAndS3(t, true) // saas mode required for registration
}

func newActionTestHarnessWithS3AndEncryption(t *testing.T, disableEncryption bool) *actionTestHarness {
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

func newActionTestHarnessWithSaaSAndS3(t *testing.T, isSaaSModeEnabled bool) *actionTestHarness {
	s3Config := newTestS3Config(t)
	return newActionTestHarnessWithSaaSAndS3Config(t, isSaaSModeEnabled, s3Config)
}

func newActionTestHarnessWithSaaSAndS3Config(t *testing.T, isSaaSModeEnabled bool, s3Config *testS3Config) *actionTestHarness {
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

	systemConfig := initSystemConfig(t, mainDB, isSaaSModeEnabled)

	templates := template.New("app")
	templates.Funcs(ui.TemplateFuncMap(templates))
	templates, err = templates.ParseFS(ui.WidgetFS, "widget/*.gohtml")
	if err != nil {
		t.Fatal(err)
	}

	renderer := ui.NewRenderer(templates)
	i18nx := i18n.NewI18n()

	fileSystem := filesystem.NewFileSystem(metaPath)
	s3FileSystem := filesystem.NewS3FileSystem(nil, "", fileSystem, false)
	if s3Config != nil {
		s3FileSystem = filesystem.NewS3FileSystem(s3Config.client, s3Config.bucketName, fileSystem, s3Config.disableEncryption)
	}

	infra := common.NewInfra(
		renderer,
		metaPath,
		s3FileSystem,
		common.NewFactory(),
		common.NewFileRepository(),
		pluginx.NewRegistry(),
		systemConfig,
	)

	tenantDBs := tenantdbs.NewTenantDBs()
	router := NewRouter(mainDB, tenantDBs, infra, true, metaPath, i18nx)
	actions := action.NewActions(infra, tenantDBs)
	router.RegisterActions(actions)

	err = infra.PluginRegistry().RegisterActions(router)
	if err != nil {
		t.Fatal(err)
	}

	return &actionTestHarness{
		t:         t,
		mainDB:    mainDB,
		tenantDBs: tenantDBs,
		infra:     infra,
		actions:   actions,
		router:    router,
		metaPath:  metaPath,
		i18n:      i18nx,
	}
}

func cleanupS3TestObjects(t *testing.T, mainDB *sqlx.MainDB, s3Config *testS3Config) {
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

func cleanupS3Prefix(t *testing.T, client *minio.Client, bucketName, prefix string) {
	t.Helper()

	ctx := context.Background()
	for object := range client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if object.Err != nil {
			t.Errorf("list objects %s: %v", prefix, object.Err)
			continue
		}

		if err := client.RemoveObject(ctx, bucketName, object.Key, minio.RemoveObjectOptions{}); err != nil {
			t.Errorf("remove object %s: %v", object.Key, err)
		}
	}
}

func newTestS3Config(t *testing.T) *testS3Config {
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

func initSystemConfig(t *testing.T, mainDB *sqlx.MainDB, isSaaSModeEnabled bool) *modelmain.SystemConfig {
	t.Helper()

	ctx := context.Background()
	tx, err := mainDB.ReadWriteConn.Tx(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = modelmain.InitAppWithoutCustomContext(
		ctx,
		tx,
		"",
		true,
		modelmain.S3Config{},
		modelmain.TLSConfig{},
		modelmain.MailerConfig{},
		modelmain.OCRConfig{},
	)
	if err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	systemConfigx := mainDB.ReadWriteConn.SystemConfig.Query().FirstX(ctx)
	return modelmain.NewSystemConfig(systemConfigx, isSaaSModeEnabled, false, true)
}

func createAccount(t *testing.T, mainDB *sqlx.MainDB, email, password string) {
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
		SetRole(mainrole.User).
		SetPasswordSalt(salt).
		SetPasswordHash(passwordHash).
		SaveX(context.Background())
}

func TestSignInCmdSetsSessionAndRedirect(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "user@example.com"
	password := "supersecret"
	createAccount(t, harness.mainDB, email, password)

	form := url.Values{}
	form.Set("Email", email)
	form.Set("Password", password)
	form.Set("TwoFactorAuthenticationCode", "")

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
