package server

import (
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/action/browse"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
)

func TestBrowseFolderNavigationPreservesSheetAndUpdatesItsScope(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "folder-side-sheet")
	var childID string
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(space.Name("MCP folder-side-sheet")).OnlyX(tc)
		spacex = spacex.Update().SetIsFolderMode(true).SaveX(tc)
		sc := ctxx.NewSpaceContext(tc, spacex)
		childID = createDirectoryForTest(sc, sc.SpaceRootDir().ID, "child").PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	h.router.RegisterPage(route.BrowseRoute(true), h.actions.Browse.BrowsePage.Handler)
	rootURL := route.Browse(f.tenant.PublicID.String(), f.spaceID, f.rootID)
	childURL := route.Browse(f.tenant.PublicID.String(), f.spaceID, childID)
	navigate := func(currentURL string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, childURL, nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Boosted", "true")
		req.Header.Set("HX-Current-URL", currentURL)
		req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: f.session})
		response := httptest.NewRecorder()
		h.router.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			t.Fatalf("folder navigation: %d: %s", response.Code, response.Body.String())
		}
		return response
	}

	response := navigate(rootURL +
		"?side_sheet=filtersDialog&filter_tab=file-types&sort_by=name&file_types=.zip")
	location, err := url.Parse(response.Header().Get("HX-Replace-Url"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Path != childURL || location.Query().Get("side_sheet") != "filtersDialog" ||
		location.Query().Get("filter_tab") != "file-types" ||
		location.Query().Get("file_types") != ".zip" || location.Query().Get("sort_by") != "name" {
		t.Fatalf("folder navigation lost filter state: %s", location)
	}
	body := response.Body.String()
	if strings.Count(body, `id="filtersDialog"`) != 1 {
		t.Fatal("folder response must refresh exactly one filters side sheet")
	}
	panel := regexp.MustCompile(`id="fileTypeFilter"[^>]*hx-vals="([^"]+)"`).
		FindStringSubmatch(body)
	if len(panel) != 2 {
		t.Fatal("selected file-types tab is missing its query payload")
	}
	var data browse.FileTypeFilterPartialData
	if err := json.Unmarshal([]byte(html.UnescapeString(panel[1])), &data); err != nil {
		t.Fatal(err)
	}
	if data.CurrentDirID != childID {
		t.Fatalf("side sheet still targets previous folder: %q", data.CurrentDirID)
	}
	reset := f.browserAt(location.String(), h.actions.Browse.ResetFiltersCmd.Endpoint(), url.Values{
		"CurrentDirID": {data.CurrentDirID},
	})
	resetLocation, err := url.Parse(reset.Header().Get("HX-Replace-Url"))
	if err != nil {
		t.Fatal(err)
	}
	if reset.Code != http.StatusOK || resetLocation.Path != childURL ||
		resetLocation.Query().Get("side_sheet") != "filtersDialog" ||
		resetLocation.Query().Has("file_types") {
		t.Fatalf("reset did not retain the destination folder and open sheet: %s", resetLocation)
	}

	closed := navigate(rootURL)
	if strings.Contains(closed.Body.String(), `id="filtersDialog"`) ||
		strings.Contains(closed.Body.String(), `id="browseDefaultSideSheetTrigger"`) {
		t.Fatal("folder navigation reopened a closed filters sheet")
	}
}
