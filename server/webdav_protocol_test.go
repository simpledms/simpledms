package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	mainprivacy "github.com/simpledms/simpledms/db/entmain/privacy"
	entmainwebdavcredential "github.com/simpledms/simpledms/db/entmain/webdavcredential"
	"github.com/simpledms/simpledms/db/enttenant"
	enttenantfile "github.com/simpledms/simpledms/db/enttenant/file"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	enttenantschema "github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	enttenantwebdavresource "github.com/simpledms/simpledms/db/enttenant/webdavresource"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	credentialmodel "github.com/simpledms/simpledms/model/main/webdavcredential"
	webdavresourcemodel "github.com/simpledms/simpledms/model/tenant/webdavresource"
)

func TestWebDAVOptionsRequiresBasicAndAdvertisesNarrowAllow(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVProtocolFixture(t)
	url := webDAVTestURL(tenantx, spacex, "/")

	unauthenticated := httptest.NewRecorder()
	harness.router.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodOptions, url, nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("expected challenge, got %d", unauthenticated.Code)
	}
	if got := unauthenticated.Header().Get("WWW-Authenticate"); got != webDAVRealmExpected {
		t.Fatalf("unexpected challenge %q", got)
	}
	invalidReq := httptest.NewRequest(http.MethodOptions, url, nil)
	invalidReq.SetBasicAuth(username, "wrong-secret")
	invalid := httptest.NewRecorder()
	harness.router.ServeHTTP(invalid, invalidReq)
	if invalid.Code != http.StatusUnauthorized || invalid.Header().Get("Location") != "" {
		t.Fatalf("invalid credential should challenge without redirect, got %d", invalid.Code)
	}

	req := httptest.NewRequest(http.MethodOptions, url, nil)
	req.SetBasicAuth(username, secret)
	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected OK, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Allow"); got != webDAVAllowExpected {
		t.Fatalf("unexpected Allow %q", got)
	}
	if rr.Header().Get("DAV") != "1, 2" || rr.Header().Get("MS-Author-Via") != "DAV" {
		t.Fatalf("missing DAV headers: %#v", rr.Header())
	}
}

func TestWebDAVStructuralPropfindHidesFiles(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVProtocolFixture(t)

	for _, tc := range []struct {
		path       string
		depth      string
		wantInbox  bool
		wantStatus int
	}{
		{path: "/", depth: "0", wantStatus: http.StatusMultiStatus},
		{path: "/", depth: "1", wantInbox: true, wantStatus: http.StatusMultiStatus},
		{path: "/Inbox/", depth: "infinity", wantInbox: true, wantStatus: http.StatusMultiStatus},
		{path: "/Inbox/upload.pdf", depth: "0", wantStatus: http.StatusNotFound},
	} {
		req := httptest.NewRequest("PROPFIND", webDAVTestURL(tenantx, spacex, tc.path), strings.NewReader(webDAVPropfindBody))
		req.Header.Set("Depth", tc.depth)
		req.SetBasicAuth(username, secret)
		rr := httptest.NewRecorder()
		harness.router.ServeHTTP(rr, req)
		if rr.Code != tc.wantStatus {
			t.Fatalf("%s depth %s: expected %d, got %d: %s", tc.path, tc.depth, tc.wantStatus, rr.Code, rr.Body.String())
		}
		if tc.wantInbox && !strings.Contains(rr.Body.String(), "/Inbox/") {
			t.Fatalf("expected Inbox in PROPFIND response: %s", rr.Body.String())
		}
		if strings.Contains(rr.Body.String(), "upload.pdf") {
			t.Fatalf("file leaked in PROPFIND response: %s", rr.Body.String())
		}
	}
}

func TestWebDAVRejectsUnsafePathsAndMethods(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVProtocolFixture(t)

	for _, tc := range []struct {
		method string
		path   string
		want   int
	}{
		{method: "MKCOL", path: "/Inbox/new-folder", want: http.StatusMethodNotAllowed},
		{method: http.MethodDelete, path: "/Inbox/a.pdf", want: http.StatusMethodNotAllowed},
		{method: http.MethodHead, path: "/", want: http.StatusMethodNotAllowed},
		{method: http.MethodGet, path: "/Inbox/a.pdf", want: http.StatusNotFound},
		{method: http.MethodHead, path: "/Inbox/a.pdf", want: http.StatusNotFound},
		{method: "PUT", path: "/inbox/a.pdf", want: http.StatusConflict},
		{method: "PUT", path: "/Inbox/nested/a.pdf", want: http.StatusConflict},
		{method: "PUT", path: "/Inbox/a%2fb.pdf", want: http.StatusConflict},
		{method: "PUT", path: "/Inbox/", want: http.StatusMethodNotAllowed},
	} {
		req := httptest.NewRequest(tc.method, webDAVTestURL(tenantx, spacex, tc.path), strings.NewReader("x"))
		req.SetBasicAuth(username, secret)
		rr := httptest.NewRecorder()
		harness.router.ServeHTTP(rr, req)
		if rr.Code != tc.want {
			t.Fatalf("%s %s: expected %d, got %d", tc.method, tc.path, tc.want, rr.Code)
		}
		if tc.want == http.StatusMethodNotAllowed && rr.Header().Get("Allow") != webDAVAllowExpected {
			t.Fatalf("%s %s: missing Allow on 405: %#v", tc.method, tc.path, rr.Header())
		}
	}
}

func TestWebDAVRejectsCleartextOutsideDevBeforeAuth(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVProtocolFixture(t)
	router := NewRouter(
		harness.mainDB,
		harness.tenantDBs,
		harness.infra,
		false,
		harness.metaPath,
		harness.i18n,
		[]netip.Prefix{netip.MustParsePrefix("192.0.2.10/32")},
	)
	cleartextReq := httptest.NewRequest(http.MethodOptions, webDAVTestURL(tenantx, spacex, "/"), nil)
	cleartextReq.SetBasicAuth(username, secret)
	cleartext := httptest.NewRecorder()
	router.ServeHTTP(cleartext, cleartextReq)
	if cleartext.Code != http.StatusForbidden {
		t.Fatalf("expected cleartext forbidden, got %d", cleartext.Code)
	}
	if cleartext.Header().Get("WWW-Authenticate") != "" {
		t.Fatal("cleartext request should not get Basic challenge")
	}

	forwardedReq := httptest.NewRequest(http.MethodOptions, webDAVTestURL(tenantx, spacex, "/"), nil)
	forwardedReq.Header.Set("X-Forwarded-Proto", "https")
	forwardedReq.SetBasicAuth(username, secret)
	forwardedReq.RemoteAddr = "198.51.100.10:1234"
	forwarded := httptest.NewRecorder()
	router.ServeHTTP(forwarded, forwardedReq)
	if forwarded.Code != http.StatusForbidden {
		t.Fatalf("expected untrusted forwarded request forbidden, got %d", forwarded.Code)
	}

	forwardedReq.Header.Set("X-Forwarded-Proto", "http")
	forwardedReq.RemoteAddr = "192.0.2.10:1234"
	forwarded = httptest.NewRecorder()
	router.ServeHTTP(forwarded, forwardedReq)
	if forwarded.Code != http.StatusForbidden {
		t.Fatalf("expected trusted cleartext request forbidden, got %d", forwarded.Code)
	}

	forwardedReq.Header.Set("X-Forwarded-Proto", "https")
	forwarded = httptest.NewRecorder()
	router.ServeHTTP(forwarded, forwardedReq)
	if forwarded.Code != http.StatusOK {
		t.Fatalf(
			"expected forwarded HTTPS request OK, got %d: %s",
			forwarded.Code,
			forwarded.Body.String(),
		)
	}
}

func TestWebDAVEndpointScopeMismatchIsNotFound(t *testing.T) {
	harness, tenantx, _, username, secret := newWebDAVProtocolFixture(t)
	req := httptest.NewRequest(http.MethodOptions, "/webdav/"+tenantx.PublicID.String()+"/wrongspace/", nil)
	req.SetBasicAuth(username, secret)
	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected scope mismatch not found, got %d", rr.Code)
	}
}

func TestWebDAVLockCreatesNoRows(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVProtocolFixture(t)
	req := httptest.NewRequest("LOCK", webDAVTestURL(tenantx, spacex, "/Inbox/scan.pdf"), strings.NewReader(webDAVLockBody))
	req.Header.Set("Depth", "0")
	req.SetBasicAuth(username, secret)
	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("expected LOCK success, got %d: %s", rr.Code, rr.Body.String())
	}
	ctx := tenantprivacy.DecisionContext(
		enttenantschema.WithUnfinishedUploads(context.Background()),
		tenantprivacy.Allow,
	)
	tenantDB := mustTenantDB(t, harness, tenantx.ID)
	if count := tenantDB.ReadOnlyConn.WebDAVResource.Query().CountX(ctx); count != 0 {
		t.Fatalf("LOCK created DAV resources: %d", count)
	}
	if count := tenantDB.ReadOnlyConn.StoredFile.Query().CountX(ctx); count != 0 {
		t.Fatalf("LOCK created stored files: %d", count)
	}
}

func TestWebDAVPutCreatesUnreadableInboxFileAndRetryConflictsBeforeRead(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVS3ProtocolFixture(t)

	rr := webDAVRequest(t, harness, username, secret, "PUT", webDAVTestURL(tenantx, spacex, "/Inbox/scan.pdf"), strings.NewReader("hello"), nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected PUT created, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("ETag") != "" {
		t.Fatalf("PUT leaked ETag %q", rr.Header().Get("ETag"))
	}

	propfind := webDAVRequest(t, harness, username, secret, "PROPFIND", webDAVTestURL(tenantx, spacex, "/Inbox/scan.pdf"), strings.NewReader(webDAVPropfindBody), func(req *http.Request) {
		req.Header.Set("Depth", "0")
	})
	if propfind.Code != http.StatusNotFound {
		t.Fatalf("uploaded DAV path should stay unreadable, got %d", propfind.Code)
	}

	body := &countingReadCloser{r: strings.NewReader("second")}
	retry := webDAVRequest(t, harness, username, secret, "PUT", webDAVTestURL(tenantx, spacex, "/Inbox/scan.pdf"), body, nil)
	if retry.Code != http.StatusConflict {
		t.Fatalf("expected retry conflict, got %d: %s", retry.Code, retry.Body.String())
	}
	if body.reads != 0 {
		t.Fatalf("retry conflict read body %d times", body.reads)
	}

	storedFiles := mustTenantDB(t, harness, tenantx.ID).ReadOnlyConn.StoredFile.Query().AllX(
		tenantprivacy.DecisionContext(context.Background(), tenantprivacy.Allow),
	)
	if len(storedFiles) != 1 || storedFiles[0].MimeType == "" {
		t.Fatalf("expected finalized upload MIME type, got %#v", storedFiles)
	}
}

func TestWebDAVChunkedAndTruncatedPutStatuses(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVS3ProtocolFixture(t)
	chunked := &countingReadCloser{r: strings.NewReader("chunked")}
	rr := webDAVRequest(t, harness, username, secret, "PUT", webDAVTestURL(tenantx, spacex, "/Inbox/chunked.pdf"), chunked, func(req *http.Request) {
		req.ContentLength = -1
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected chunked PUT created, got %d: %s", rr.Code, rr.Body.String())
	}

	truncated := &countingReadCloser{r: strings.NewReader("short")}
	rr = webDAVRequest(t, harness, username, secret, "PUT", webDAVTestURL(tenantx, spacex, "/Inbox/truncated.pdf"), truncated, func(req *http.Request) {
		req.ContentLength = int64(len("short") + 1)
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected truncated PUT bad request, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWebDAVMoveAndOverwriteConflict(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVS3ProtocolFixture(t)

	if rr := webDAVRequest(t, harness, username, secret, "PUT", webDAVTestURL(tenantx, spacex, "/Inbox/a.pdf"), strings.NewReader("a"), nil); rr.Code != http.StatusCreated {
		t.Fatalf("put a: %d %s", rr.Code, rr.Body.String())
	}
	if rr := webDAVRequest(t, harness, username, secret, "PUT", webDAVTestURL(tenantx, spacex, "/Inbox/c.pdf"), strings.NewReader("c"), nil); rr.Code != http.StatusCreated {
		t.Fatalf("put c: %d %s", rr.Code, rr.Body.String())
	}

	move := webDAVRequest(t, harness, username, secret, "MOVE", webDAVTestURL(tenantx, spacex, "/Inbox/a.pdf"), nil, func(req *http.Request) {
		req.Header.Set("Destination", webDAVTestURL(tenantx, spacex, "/Inbox/b.pdf"))
	})
	if move.Code != http.StatusCreated {
		t.Fatalf("expected MOVE created, got %d: %s", move.Code, move.Body.String())
	}

	overwrite := webDAVRequest(t, harness, username, secret, "MOVE", webDAVTestURL(tenantx, spacex, "/Inbox/b.pdf"), nil, func(req *http.Request) {
		req.Header.Set("Destination", webDAVTestURL(tenantx, spacex, "/Inbox/c.pdf"))
		req.Header.Set("Overwrite", "T")
	})
	if overwrite.Code != http.StatusConflict {
		t.Fatalf("expected Overwrite:T conflict, got %d: %s", overwrite.Code, overwrite.Body.String())
	}
	ctx := tenantprivacy.DecisionContext(context.Background(), tenantprivacy.Allow)
	tenantDB := mustTenantDB(t, harness, tenantx.ID)
	if count := tenantDB.ReadOnlyConn.File.Query().Where(enttenantfile.Name("c.pdf")).CountX(ctx); count != 1 {
		t.Fatalf("Overwrite:T changed destination file count to %d", count)
	}
}

func TestWebDAVMoveRejectsNetworkPathDestinationHost(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVProtocolFixture(t)

	move := webDAVRequest(t, harness, username, secret, "MOVE", webDAVTestURL(tenantx, spacex, "/Inbox/a.pdf"), nil, func(req *http.Request) {
		req.Header.Set("Destination", "//other-host"+webDAVTestURL(tenantx, spacex, "/Inbox/b.pdf"))
	})
	if move.Code != http.StatusConflict {
		t.Fatalf("expected network-path Destination conflict, got %d: %s", move.Code, move.Body.String())
	}
}

func TestWebDAVMoveDeletesActiveAliasWithNilFileID(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVProtocolFixture(t)
	mainCtx := mainprivacy.DecisionContext(context.Background(), mainprivacy.Allow)
	credentialx := harness.mainDB.ReadOnlyConn.WebDAVCredential.Query().
		Where(entmainwebdavcredential.Username(username)).
		OnlyX(mainCtx)
	tenantCtx := tenantprivacy.DecisionContext(context.Background(), tenantprivacy.Allow)
	tenantDB := mustTenantDB(t, harness, tenantx.ID)
	resource := tenantDB.ReadWriteConn.WebDAVResource.Create().
		SetCredentialPublicID(entx.NewCIText(credentialx.PublicID.String())).
		SetSpaceID(spacex.ID).
		SetDavPath("/Inbox/stale.pdf").
		SetState(webdavresourcemodel.Active).
		SaveX(tenantCtx)

	move := webDAVRequest(t, harness, username, secret, "MOVE", webDAVTestURL(tenantx, spacex, "/Inbox/stale.pdf"), nil, func(req *http.Request) {
		req.Header.Set("Destination", webDAVTestURL(tenantx, spacex, "/Inbox/new.pdf"))
	})
	if move.Code != http.StatusConflict {
		t.Fatalf("expected stale alias conflict, got %d: %s", move.Code, move.Body.String())
	}
	if count := tenantDB.ReadOnlyConn.WebDAVResource.Query().Where(enttenantwebdavresource.ID(resource.ID)).CountX(tenantCtx); count != 0 {
		t.Fatalf("stale alias was not deleted, count %d", count)
	}
}

func TestWebDAVLockBoundsAndXMLLimit(t *testing.T) {
	harness, tenantx, spacex, username, secret := newWebDAVProtocolFixture(t)
	for i := range 32 {
		rr := webDAVRequest(t, harness, username, secret, "LOCK", webDAVTestURL(tenantx, spacex, "/Inbox/lock-"+strconv.Itoa(i)+".pdf"), strings.NewReader(webDAVLockBody), func(req *http.Request) {
			req.Header.Set("Depth", "0")
		})
		if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
			t.Fatalf("lock %d: got %d: %s", i, rr.Code, rr.Body.String())
		}
	}
	rr := webDAVRequest(t, harness, username, secret, "LOCK", webDAVTestURL(tenantx, spacex, "/Inbox/too-many.pdf"), strings.NewReader(webDAVLockBody), func(req *http.Request) {
		req.Header.Set("Depth", "0")
	})
	if rr.Code < 400 {
		t.Fatalf("expected excess LOCK rejection, got %d", rr.Code)
	}

	bigXML := strings.Repeat("x", 64*1024+1)
	rr = webDAVRequest(t, harness, username, secret, "PROPFIND", webDAVTestURL(tenantx, spacex, "/"), strings.NewReader(bigXML), func(req *http.Request) {
		req.Header.Set("Depth", "0")
	})
	if rr.Code < 400 {
		t.Fatalf("expected oversized XML rejection, got %d", rr.Code)
	}
}

func webDAVRequest(
	t *testing.T,
	harness *actionTestHarness,
	username string,
	secret string,
	method string,
	url string,
	body io.Reader,
	mutate func(*http.Request),
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, url, body)
	req.SetBasicAuth(username, secret)
	if mutate != nil {
		mutate(req)
	}
	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)
	return rr
}

type countingReadCloser struct {
	r     io.Reader
	reads int
}

func (qq *countingReadCloser) Read(p []byte) (int, error) {
	qq.reads++
	return qq.r.Read(p)
}

func (qq *countingReadCloser) Close() error { return nil }

func newWebDAVProtocolFixture(t *testing.T) (*actionTestHarness, *entmain.Tenant, *enttenant.Space, string, string) {
	t.Helper()
	harness := newActionTestHarnessWithSaaS(t, true)
	return newWebDAVProtocolFixtureWithHarness(t, harness)
}

func newWebDAVS3ProtocolFixture(t *testing.T) (*actionTestHarness, *entmain.Tenant, *enttenant.Space, string, string) {
	t.Helper()
	endpoint := envOrDefault("SIMPLEDMS_S3_ENDPOINT", "localhost:7070")
	conn, err := net.DialTimeout("tcp", endpoint, time.Second)
	if err != nil {
		t.Skipf("S3 endpoint %s unavailable: %v", endpoint, err)
	}
	_ = conn.Close()
	harness := newActionTestHarnessWithSaaSAndS3(t, true)
	return newWebDAVProtocolFixtureWithHarness(t, harness)
}

func newWebDAVProtocolFixtureWithHarness(t *testing.T, harness *actionTestHarness) (*actionTestHarness, *entmain.Tenant, *enttenant.Space, string, string) {
	t.Helper()
	accountx, tenantx := signUpAccount(t, harness, "webdav-owner@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	mainTx, tenantTx, tenantCtx := newTenantContext(t, harness, accountx, tenantx, tenantDB)
	createSpaceViaCmd(t, harness.actions, tenantCtx, "WebDAV Space")
	spacex := tenantTx.Space.Query().Where(space.Name("WebDAV Space")).OnlyX(tenantCtx)
	spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
	result, err := credentialmodel.NewCredentialService().CreateOwnerCredential(
		spaceCtx,
		"Scanner",
		webDAVTestURL(tenantx, spacex, "/"),
		0,
		false,
	)
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		t.Fatalf("commit main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		t.Fatalf("commit tenant tx: %v", err)
	}
	return harness, tenantx, spacex, result.Username, result.Secret
}

func mustTenantDB(t *testing.T, harness *actionTestHarness, tenantID int64) *sqlx.TenantDB {
	t.Helper()
	tenantDB, ok := harness.tenantDBs.Load(tenantID)
	if !ok {
		t.Fatal("tenant db missing")
	}
	return tenantDB
}

func webDAVTestURL(tenantx *entmain.Tenant, spacex *enttenant.Space, suffix string) string {
	return "/webdav/" + tenantx.PublicID.String() + "/" + spacex.PublicID.String() + suffix
}

const webDAVPropfindBody = `<?xml version="1.0"?><D:propfind xmlns:D="DAV:"><D:allprop/></D:propfind>`

const webDAVLockBody = `<?xml version="1.0"?><D:lockinfo xmlns:D="DAV:"><D:lockscope><D:exclusive/></D:lockscope><D:locktype><D:write/></D:locktype><D:owner>test</D:owner></D:lockinfo>`

const webDAVAllowExpected = "OPTIONS, PROPFIND, PUT, LOCK, UNLOCK, MOVE"

const webDAVRealmExpected = `Basic realm="SimpleDMS WebDAV", charset="UTF-8"`
