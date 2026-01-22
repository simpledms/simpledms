package server

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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
	router    *Router
}

func newActionTestHarness(t *testing.T) *actionTestHarness {
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

	systemConfig := initSystemConfig(t, mainDB)

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

	infra := common.NewInfra(
		renderer,
		metaPath,
		s3FileSystem,
		common.NewFactory(),
		common.NewFileRepository(),
		systemConfig,
	)

	tenantDBs := tenantdbs.NewTenantDBs()
	router := NewRouter(mainDB, tenantDBs, infra, true, metaPath, i18nx)
	actions := action.NewActions(infra, tenantDBs)
	router.RegisterActions(actions)

	return &actionTestHarness{
		t:         t,
		mainDB:    mainDB,
		tenantDBs: tenantDBs,
		infra:     infra,
		router:    router,
	}
}

func initSystemConfig(t *testing.T, mainDB *sqlx.MainDB) *modelmain.SystemConfig {
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
	return modelmain.NewSystemConfig(systemConfigx, false, false, true)
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
