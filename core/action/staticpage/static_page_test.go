package staticpage

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/core/db/entmain/enttest"
	"github.com/simpledms/simpledms/core/db/entx"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	common2 "github.com/simpledms/simpledms/core/common"
	appmodel "github.com/simpledms/simpledms/core/model/app"
	"github.com/simpledms/simpledms/core/model/common/language"
	"github.com/simpledms/simpledms/core/model/common/mainrole"
	systemconfigmodel "github.com/simpledms/simpledms/core/model/systemconfig"
	"github.com/simpledms/simpledms/core/pluginx"
	ui2 "github.com/simpledms/simpledms/core/ui"
	"github.com/simpledms/simpledms/core/util/accountutil"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	ctxx2 "github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/i18n"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
)

func TestRenderMarkdownAbout(t *testing.T) {
	htmlContent, err := RenderMarkdown("content/about.md")
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	rendered := string(htmlContent)

	if !strings.Contains(rendered, `class="headline-sm text-on-surface mt-6 mb-3"`) {
		t.Fatalf("expected rendered markdown to contain styled heading class, got: %s", rendered)
	}

	if !strings.Contains(rendered, `class="text-on-surface list-disc ml-6 my-3"`) {
		t.Fatalf("expected rendered markdown to contain styled unordered list class, got: %s", rendered)
	}

	if !strings.Contains(rendered, `target="_blank"`) {
		t.Fatalf("expected rendered markdown to contain target _blank for external links, got: %s", rendered)
	}

	if !strings.Contains(rendered, `rel="noopener noreferrer"`) {
		t.Fatalf("expected rendered markdown to contain rel noopener noreferrer for external links, got: %s", rendered)
	}

	if strings.Contains(rendered, `style=`) {
		t.Fatalf("expected rendered markdown to avoid inline styles, got: %s", rendered)
	}
}

func TestRenderMarkdownMissingFile(t *testing.T) {
	_, err := RenderMarkdown("content/does-not-exist.md")
	if err == nil {
		t.Fatal("expected error for missing markdown file")
	}
}

func TestRenderMarkdownAllowsInlineHTML(t *testing.T) {
	htmlContent, err := RenderMarkdown("content/html_support_test.md")
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	rendered := string(htmlContent)

	if !strings.Contains(rendered, `<span class="inline-html">inline html</span>`) {
		t.Fatalf("expected rendered markdown to include inline html span, got: %s", rendered)
	}

	if !strings.Contains(rendered, `<div class="custom-block">`) {
		t.Fatalf("expected rendered markdown to include html div block, got: %s", rendered)
	}
}

func TestStaticPageHandlerReturnsNotFoundForUnknownSlug(t *testing.T) {
	page, ctx := newStaticPageTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/pages/unknown/", nil)
	req.SetPathValue("slug", "unknown")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	rw := httpx2.NewResponseWriter(rr)

	err := page.Handler(rw, httpx2.NewRequest(req), ctx)
	if err == nil {
		t.Fatal("expected handler error for unknown static page slug")
	}

	var httpErr *e.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTPError, got %T", err)
	}

	if httpErr.StatusCode() != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, httpErr.StatusCode())
	}
}

func TestStaticPageHandlerRendersAbout(t *testing.T) {
	page, ctx := newStaticPageTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/pages/about/", nil)
	req.SetPathValue("slug", "about")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	rw := httpx2.NewResponseWriter(rr)

	err := page.Handler(rw, httpx2.NewRequest(req), ctx)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}

	body := rr.Body.String()

	if !strings.Contains(body, "About SimpleDMS") {
		t.Fatalf("expected response to contain app bar title, got: %s", body)
	}

	if !strings.Contains(body, "Copyright (c) 2023") {
		t.Fatalf("expected response to contain about markdown content, got: %s", body)
	}

	if !strings.Contains(body, "list-disc ml-6 my-3") {
		t.Fatalf("expected response to contain rendered markdown classes, got: %s", body)
	}
}

func newStaticPageTestSetup(t *testing.T) (*StaticPage, *ctxx.MainContext) {
	t.Helper()
	metaPath := t.TempDir()
	ctx := context.Background()

	client := enttest.Open(t, "sqlite3", "file:staticpage-test?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		_ = client.Close()
	})

	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
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
		t.Fatalf("init app: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	systemConfigx := client.SystemConfig.Query().FirstX(ctx)
	systemConfig := systemconfigmodel.NewSystemConfig(systemConfigx, false, false, true, "", "", "")

	templates := template.New("app")
	templates.Funcs(ui2.TemplateFuncMap(templates))

	parsedTemplates, err := templates.ParseFS(ui2.WidgetFS, "widget/*.gohtml")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	renderer := ui2.NewRenderer(parsedTemplates)
	fileSystem := filesystem.NewS3FileSystem(
		nil,
		"",
		filesystem.NewFileSystem(metaPath),
		false,
		filesystem.NewStorageQuota(false),
	)
	infra := common2.NewInfra(
		renderer,
		metaPath,
		fileSystem,
		common.NewFileRepository(),
		pluginx.NewRegistry(),
		systemConfig,
	)

	actions := new(Actions)
	page := NewStaticPage(infra, actions)
	actions.StaticPage = page

	salt, ok := accountutil.RandomSalt()
	if !ok {
		t.Fatal("could not generate account salt")
	}

	passwordHash := accountutil.PasswordHash("test-password", salt)

	account := client.Account.Create().
		SetEmail(entx.NewCIText("test@example.com")).
		SetFirstName("Test").
		SetLastName("User").
		SetLanguage(language.Unknown).
		SetRole(mainrole.User).
		SetPasswordSalt(salt).
		SetPasswordHash(passwordHash).
		SaveX(ctx)

	i18nx := i18n.NewI18n()
	visitorCtx := ctxx2.NewVisitorContext(ctx, nil, i18nx, "", "UTC", true, false, false)
	mainCtx := ctxx.NewMainContext(visitorCtx, account, i18nx, nil, tenantdbs.NewTenantDBs(), true)

	return page, mainCtx
}
