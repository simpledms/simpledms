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

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/enttest"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/i18n"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/common/mainrole"
	"github.com/simpledms/simpledms/ui"
	"github.com/simpledms/simpledms/util/accountutil"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
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

func TestStaticPageHandlerReturnsNotFoundForUnknownSlug(t *testing.T) {
	page, ctx := newStaticPageTestSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/pages/unknown/", nil)
	req.SetPathValue("slug", "unknown")
	req.Header.Set("HX-Request", "true")

	rr := httptest.NewRecorder()
	rw := httpx.NewResponseWriter(rr)

	err := page.Handler(rw, httpx.NewRequest(req), ctx)
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
	rw := httpx.NewResponseWriter(rr)

	err := page.Handler(rw, httpx.NewRequest(req), ctx)
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

	templates := template.New("app")
	templates.Funcs(ui.TemplateFuncMap(templates))

	parsedTemplates, err := templates.ParseFS(ui.WidgetFS, "widget/*.gohtml")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	renderer := ui.NewRenderer(parsedTemplates)
	infra := common.NewInfra(
		renderer,
		t.TempDir(),
		nil,
		common.NewFactory(),
		common.NewFileRepository(),
		nil,
	)

	actions := new(Actions)
	page := NewStaticPage(infra, actions)
	actions.StaticPage = page

	client := enttest.Open(t, "sqlite3", "file:staticpage-test?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		_ = client.Close()
	})

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
		SaveX(context.Background())

	i18nx := i18n.NewI18n()
	visitorCtx := ctxx.NewVisitorContext(context.Background(), nil, i18nx, "", "UTC", true, false)
	mainCtx := ctxx.NewMainContext(visitorCtx, account, i18nx, nil, tenantdbs.NewTenantDBs(), true)

	return page, mainCtx
}
