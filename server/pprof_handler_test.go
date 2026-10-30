package server

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/db/entmain/session"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	sessionmodel "github.com/simpledms/simpledms/model/main/session"
)

const pprofPath = "adminreal1-sec"
const pprofIndexPath = "/" + pprofPath + "/debug/pprof/"

func TestPprofHandlerIsDisabledByDefault(t *testing.T) {
	t.Setenv("SIMPLEDMS_PPROF_PATH", "")
	harness := newActionTestHarness(t)
	email, password := "pprof-disabled@example.com", "supersecret"
	createAccountWithRole(t, harness.mainDB, email, password, mainrole.Admin)
	cookie := signInAndGetSessionCookie(t, harness, email, password)

	rr := servePprofRequest(harness.router, cookie)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("disabled GET status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestPprofHandlerAuthorizesAdminAndServesProfiles(t *testing.T) {
	t.Setenv("SIMPLEDMS_PPROF_PATH", pprofPath)
	harness := newActionTestHarness(t)
	email, password := "pprof-admin@example.com", "supersecret"
	createAccountWithRole(t, harness.mainDB, email, password, mainrole.Admin)
	cookie := signInAndGetSessionCookie(t, harness, email, password)

	for _, path := range []string{pprofIndexPath, pprofIndexPath + "goroutine?debug=1"} {
		rr := servePprofRequest(harness.router, cookie, path)
		if rr.Code != http.StatusOK || rr.Body.Len() == 0 {
			t.Errorf("GET %s = status %d with %d bytes, want nonempty 200 response", path, rr.Code, rr.Body.Len())
		}
		if got := rr.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("GET %s Cache-Control = %q, want no-store", path, got)
		}
		if got := rr.Header().Get("Referrer-Policy"); got != "no-referrer" {
			t.Errorf("GET %s Referrer-Policy = %q, want no-referrer", path, got)
		}
	}

	index := servePprofRequest(harness.router, cookie)
	href := regexp.MustCompile(`href=['\"]([^'\"]*goroutine\?debug=1)['\"]`).FindSubmatch(index.Body.Bytes())
	if href == nil {
		t.Fatal("pprof index does not contain a goroutine?debug=1 link")
	}
	indexURL, err := url.Parse(pprofIndexPath)
	if err != nil {
		t.Fatalf("parse pprof index path: %v", err)
	}
	linkURL, err := url.Parse(string(href[1]))
	if err != nil {
		t.Fatalf("parse pprof goroutine link: %v", err)
	}
	resolvedURL := indexURL.ResolveReference(linkURL)
	if !strings.HasPrefix(resolvedURL.Path, pprofIndexPath) {
		t.Fatalf("resolved goroutine link path = %q, want prefix %q", resolvedURL.Path, pprofIndexPath)
	}
	goroutine := servePprofRequest(harness.router, cookie, resolvedURL.String())
	if goroutine.Code != http.StatusOK {
		t.Errorf("GET resolved goroutine link status = %d, want %d", goroutine.Code, http.StatusOK)
	}
	for _, path := range []string{"/debug/pprof/", "/wrong/debug/pprof/"} {
		if rr := servePprofRequest(harness.router, cookie, path); rr.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want %d", path, rr.Code, http.StatusNotFound)
		}
	}

	req := httptest.NewRequest(http.MethodGet, pprofIndexPath+"profile?seconds=1", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("CPU profile status = %d, want %d", rr.Code, http.StatusOK)
	}
	reader, err := gzip.NewReader(bytes.NewReader(rr.Body.Bytes()))
	if err != nil {
		t.Fatalf("open gzip CPU profile: %v", err)
	}
	defer reader.Close()
	profile, err := io.ReadAll(reader)
	if err != nil || len(profile) == 0 {
		t.Fatalf("read CPU profile: %v; payload length = %d", err, len(profile))
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("CPU profile Cache-Control = %q, want no-store", got)
	}

	post := httptest.NewRequest(http.MethodPost, pprofIndexPath, nil)
	post.AddCookie(cookie)
	postRR := httptest.NewRecorder()
	harness.router.ServeHTTP(postRR, post)
	if postRR.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST pprof status = %d, want %d", postRR.Code, http.StatusMethodNotAllowed)
	}
}

func TestPprofHandlerRejectsUnauthorizedSessions(t *testing.T) {
	t.Setenv("SIMPLEDMS_PPROF_PATH", pprofPath)
	harness := newActionTestHarness(t)
	for _, path := range []string{pprofIndexPath, pprofIndexPath + "profile?seconds=1"} {
		rr := servePprofRequest(harness.router, nil, path)
		if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/" {
			t.Errorf("anonymous GET %s status/location = %d/%q, want 303 /", path, rr.Code, rr.Header().Get("Location"))
		}
		if got := rr.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("anonymous GET %s Cache-Control = %q, want no-store", path, got)
		}
	}

	email, password := "pprof-non-admin@example.com", "supersecret"
	createAccountWithRole(t, harness.mainDB, email, password, mainrole.User)
	cookie := signInAndGetSessionCookie(t, harness, email, password)
	nonAdmin := servePprofRequest(harness.router, cookie)
	if nonAdmin.Code != http.StatusForbidden {
		t.Fatalf("non-admin GET status = %d, want %d", nonAdmin.Code, http.StatusForbidden)
	}

	email, password = "pprof-expired@example.com", "supersecret"
	createAccountWithRole(t, harness.mainDB, email, password, mainrole.Admin)
	cookie = signInAndGetSessionCookie(t, harness, email, password)
	harness.mainDB.ReadWriteConn.Session.Update().
		Where(session.Value(sessionmodel.HashValue(cookie.Value))).
		SetExpiresAt(time.Now().Add(-time.Minute)).
		SaveX(t.Context())
	expired := servePprofRequest(harness.router, cookie)
	if expired.Code != http.StatusSeeOther || expired.Header().Get("Location") != "/" {
		t.Fatalf("expired session status/location = %d/%q, want 303 /", expired.Code, expired.Header().Get("Location"))
	}
}

func TestPprofHandlerRejectsTemporaryAdminSession(t *testing.T) {
	t.Setenv("SIMPLEDMS_PPROF_PATH", pprofPath)
	harness := newActionTestHarness(t)
	email, password := "pprof-temporary@example.com", "supersecret"
	createAccountWithRole(t, harness.mainDB, email, password, mainrole.Admin)
	cookie := signInAndGetSessionCookie(t, harness, email, password)
	harness.mainDB.ReadWriteConn.Session.Update().
		Where(session.Value(sessionmodel.HashValue(cookie.Value))).
		SetIsTemporarySession(true).
		SetDeletableAt(time.Now().Add(time.Hour)).
		SaveX(t.Context())

	rr := servePprofRequest(harness.router, cookie)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("temporary admin status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestPprofHandlerRejectsInvalidConfiguredPaths(t *testing.T) {
	for _, path := range []string{"bad/path", "bad*path", "bad.path", "bad path"} {
		t.Run(path, func(t *testing.T) {
			t.Setenv("SIMPLEDMS_PPROF_PATH", path)
			harness := newActionTestHarness(t)
			email, password := "pprof-invalid@example.com", "supersecret"
			createAccountWithRole(t, harness.mainDB, email, password, mainrole.Admin)
			cookie := signInAndGetSessionCookie(t, harness, email, password)
			for _, requestPath := range []string{pprofIndexPath, "/" + url.PathEscape(path) + "/debug/pprof/"} {
				if rr := servePprofRequest(harness.router, cookie, requestPath); rr.Code != http.StatusNotFound {
					t.Errorf("GET %s status = %d, want %d", requestPath, rr.Code, http.StatusNotFound)
				}
			}
		})
	}
}

func servePprofRequest(router http.Handler, cookie *http.Cookie, paths ...string) *httptest.ResponseRecorder {
	path := pprofIndexPath
	if len(paths) > 0 {
		path = paths[0]
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}
