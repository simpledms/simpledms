package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

func TestBrowseFileTypeFilters(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "file-type-filters")
	fileIDs := make(map[string]string)
	var nestedDirID string
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(space.Name("MCP file-type-filters")).OnlyX(tc)
		spacex = spacex.Update().SetIsFolderMode(true).SaveX(tc)
		sc := ctxx.NewSpaceContext(tc, spacex)
		root := sc.SpaceRootDir()
		nested := createDirectoryForTest(sc, root.ID, "nested")
		nestedDirID = nested.PublicID.String()
		createDirectoryForTest(sc, root.ID, "directory.gif")
		var files []*enttenant.File
		for _, name := range []string{
			"archive.ZIP", "invoice.pdf", "notes.txt", "README", "literal.x_y", "other.xay",
		} {
			filex := createRegularFileForTest(sc, nested.ID, name).Data
			fileIDs[name] = filex.PublicID.String()
			files = append(files, filex)
		}
		deleted := createRegularFileForTest(sc, root.ID, "deleted.csv").Data
		deleted.Update().SetDeletedAt(time.Now()).SaveX(sc)
		if err := seedStoredFilesForBenchmarkRows(sc, files); err != nil {
			return err
		}
		createSpaceViaCmd(t, h.actions, tc, "Other file types")
		other := tc.TTx.Space.Query().Where(space.Name("Other file types")).OnlyX(tc)
		otherCtx := ctxx.NewSpaceContext(tc, other)
		createRegularFileForTest(otherCtx, otherCtx.SpaceRootDir().ID, "other.docx")
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	browseURL := route.Browse(f.tenant.PublicID.String(), f.spaceID, f.rootID)
	currentURL := browseURL + "?filter_tab=file-types&side_sheet=filtersDialog&sort_by=name"
	request := func(endpoint string, form url.Values) *httptest.ResponseRecorder {
		t.Helper()
		response := f.browserAt(currentURL, endpoint, form)
		if response.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", endpoint, response.Code, response.Body.String())
		}
		return response
	}
	form := url.Values{"CurrentDirID": {f.rootID}}
	panel := request(h.actions.Browse.FileTypeFilterPartial.Endpoint(), form).Body.String()
	for _, extension := range []string{".zip", ".pdf", ".txt", "none"} {
		if !strings.Contains(panel, `value="`+extension+`"`) {
			t.Errorf("Space-wide choices missing %q", extension)
		}
	}
	for _, extension := range []string{".docx", ".gif", ".csv", ".ZIP"} {
		if strings.Contains(panel, `value="`+extension+`"`) {
			t.Errorf("choices contain foreign, directory, deleted or duplicate type %q", extension)
		}
	}

	tabs := request(h.actions.Browse.FilterTabsPartial.Endpoint(), url.Values{
		"CurrentDirID": {f.rootID}, "ActiveTab": {"file-types"},
	}).Body.String()
	if strings.LastIndex(tabs, `role="tab"`) > strings.Index(tabs, `id="filterTabBadge-file-types"`) {
		t.Fatal("File types is not the final tab")
	}
	if !strings.Contains(tabs, `id="fileTypeFilter"`) {
		t.Fatal("File types tab does not render its query island")
	}

	toggle := func(extension string) {
		t.Helper()
		response := request(h.actions.Browse.ToggleFileTypeFilterCmd.Endpoint(), url.Values{
			"CurrentDirID": {f.rootID}, "FileType": {extension},
		})
		if response.Header().Get("HX-Trigger-After-Swap") != event.FileTypeFilterChanged.String() {
			t.Fatal("type selection did not notify query islands after updating the URL")
		}
		if strings.Contains(response.Body.String(), `id="fileList"`) {
			t.Fatal("filter command rendered the file list")
		}
		currentURL = response.Header().Get("HX-Replace-Url")
	}
	assertListed := func(expected ...string) {
		t.Helper()
		body := request(h.actions.Browse.ListDirPartial.Endpoint()+"?hx-target=%23fileList", form).
			Body.String()
		for name, id := range fileIDs {
			if strings.Contains(body, id) != slices.Contains(expected, name) {
				t.Errorf("file %q: expected listed=%v", name, slices.Contains(expected, name))
			}
		}
	}

	toggle(".zip")
	assertListed("archive.ZIP")
	toggle(".pdf")
	assertListed("archive.ZIP", "invoice.pdf")
	toggle(".zip")
	assertListed("invoice.pdf")

	// File types intersect text search and survive switching tabs and reloading their island.
	currentURL += "&q=no-match-token"
	assertListed()
	panel = request(h.actions.Browse.FileTypeFilterPartial.Endpoint(), form).Body.String()
	if !strings.Contains(panel, "checked") || !strings.Contains(panel, `value=".zip"`) {
		t.Fatal("filter query lost selection or hid unselected available types")
	}
	reset := request(h.actions.Browse.ResetFiltersCmd.Endpoint(), form)
	currentURL = reset.Header().Get("HX-Replace-Url")
	parsed, err := url.Parse(currentURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Has("file_types") || parsed.Query().Get("q") != "no-match-token" ||
		parsed.Query().Get("filter_tab") != "file-types" ||
		parsed.Query().Get("side_sheet") != "filtersDialog" ||
		parsed.Query().Get("sort_by") != "name" {
		t.Fatalf("reset did not clear types while preserving other state: %s", currentURL)
	}
	if !strings.Contains(reset.Header().Get("HX-Trigger-After-Swap"),
		event.FileTypeFilterChanged.String()) {
		t.Fatal("reset did not refresh the file-type panel and badge")
	}

	currentURL = browseURL
	toggle("none")
	assertListed("README")
	currentURL = browseURL
	toggle(".x_y")
	assertListed("literal.x_y")

	// Filtering preserves an open file's URL rather than closing its detail pane.
	currentURL = route.BrowseFile(
		f.tenant.PublicID.String(), f.spaceID, f.rootID, fileIDs["archive.ZIP"],
	)
	toggle(".zip")
	if !strings.Contains(currentURL, "/file/"+fileIDs["archive.ZIP"]) {
		t.Fatal("filter selection discarded the selected file")
	}

	// No selection restores ordinary folder browsing instead of forcing recursive results.
	currentURL = browseURL
	body := request(h.actions.Browse.ListDirPartial.Endpoint()+"?hx-target=%23fileList", form).
		Body.String()
	if !strings.Contains(body, nestedDirID) || strings.Contains(body, fileIDs["archive.ZIP"]) {
		t.Fatal("unfiltered folder browsing did not restore the current directory scope")
	}
}
