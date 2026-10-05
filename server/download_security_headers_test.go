package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/action/download"
	trashaction "github.com/simpledms/simpledms/action/trash"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
)

// Uploaded files are served from the application origin. Inline active content, such as
// HTML or SVG, must not run scripts with the viewer's session.
func TestDownloadRoutesSandboxUploadedActiveContent(t *testing.T) {
	harness := newActionTestHarnessWithS3(t)
	accountx, tenantx := signUpAccount(t, harness, "download-headers@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	var spaceID, liveFileID, trashedFileID string
	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Download headers")
		spacex := tenantCtx.TTx.Space.Query().Where(space.Name("Download headers")).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		spaceID = spaceCtx.SpaceID
		upload := func(filename string) *enttenant.File {
			filex := uploadSpaceFile(
				t,
				harness,
				spaceCtx,
				spaceCtx.SpaceRootDir().ID,
				filename,
				[]byte("<html><script>alert(document.cookie)</script></html>"),
				false,
			)
			filex.QueryFileVersions().QueryStoredFile().OnlyX(spaceCtx).Update().
				SetMimeType("text/html; charset=utf-8").
				SaveX(spaceCtx)
			return filex
		}
		liveFileID = upload("evil.html").PublicID.String()
		trashedFile := upload("trashed-evil.html")
		trashedFile.Update().
			SetDeletedAt(trashedFile.CreatedAt).
			SetDeleter(spaceCtx.User).
			SaveX(spaceCtx)
		trashedFileID = trashedFile.PublicID.String()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	harness.router.RegisterPage(route.DownloadRoute(), download.NewDownload(harness.infra).Handler)
	harness.router.RegisterPage(
		route.TrashDownloadRoute(),
		trashaction.NewDownload(harness.infra).Handler,
	)
	harness.router.RegisterPage(
		route.OriginalSourceRoute(),
		download.NewPreview(harness.infra).OriginalSourceHandler,
	)
	session := createSessionForAccountForRulesTest(t, harness, accountx.ID)
	tenantID := tenantx.PublicID.String()
	for name, url := range map[string]string{
		"inline download":       route.DownloadInline(tenantID, spaceID, liveFileID),
		"attachment download":   route.Download(tenantID, spaceID, liveFileID),
		"inline trash download": route.TrashDownloadInline(tenantID, spaceID, trashedFileID),
		"HTML source preview":   route.OriginalSource(tenantID, spaceID, liveFileID),
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, url, nil)
			req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session})
			rr := httptest.NewRecorder()
			harness.router.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d: %s", rr.Code, rr.Body.String())
			}
			if !strings.Contains(rr.Body.String(), "<script>") {
				t.Fatalf("expected original file bytes, got %q", rr.Body.String())
			}
			if got := rr.Header().Get("Content-Security-Policy"); got != "sandbox" {
				t.Fatalf("Content-Security-Policy = %q", got)
			}
			if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("X-Content-Type-Options = %q", got)
			}
		})
	}
}
