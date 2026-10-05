package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/ui/uix/route"
)

func TestBrowseFileListLoadsRowContextMenuOnOpen(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "browse-lazy-menu")
	var fileID string
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(space.Name("MCP browse-lazy-menu")).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, spacex)
		filex := createRegularFileForTest(sc, sc.SpaceRootDir().ID, "lazy-menu.txt").Data
		fileID = filex.PublicID.String()
		return seedStoredFilesForBenchmarkRows(sc, []*enttenant.File{filex})
	}); err != nil {
		t.Fatal(err)
	}
	browseURL := route.Browse(f.tenant.PublicID.String(), f.spaceID, f.rootID)

	assertFileListLoadsRowContextMenuOnOpen(
		t,
		f,
		browseURL,
		f.browserAt(browseURL, h.actions.Browse.ListDirPartial.Endpoint(), url.Values{
			"CurrentDirID": {f.rootID},
		}),
		h.actions.Browse.FileContextMenuPartial.Endpoint(),
		fileID,
		h.actions.Browse.DeleteFileCmd.Endpoint(),
	)
}

func TestInboxFileListLoadsRowContextMenuOnOpen(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "inbox-lazy-menu")
	inboxURL := route.InboxRoot(f.tenant.PublicID.String(), f.spaceID)

	assertFileListLoadsRowContextMenuOnOpen(
		t,
		f,
		inboxURL,
		f.browserAt(inboxURL, h.actions.Inbox.InboxPage.Endpoint(), url.Values{}),
		h.actions.Inbox.FileContextMenuPartial.Endpoint(),
		f.fileID,
		h.actions.Inbox.TransferFileDialog.Endpoint(),
	)
}

func assertFileListLoadsRowContextMenuOnOpen(
	t *testing.T,
	f *mcpFixture,
	currentURL string,
	list *httptest.ResponseRecorder,
	menuEndpoint string,
	fileID string,
	menuActionEndpoint string,
) {
	t.Helper()
	if list.Code != http.StatusOK {
		t.Fatalf("file list returned status %d: %s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), menuEndpoint) {
		t.Fatalf("file list does not load row context menus from %s", menuEndpoint)
	}
	if strings.Contains(list.Body.String(), menuActionEndpoint) {
		t.Fatalf("file list renders row context menu items eagerly (%s)", menuActionEndpoint)
	}

	menu := f.browserAt(currentURL, menuEndpoint, url.Values{"FileID": {fileID}})
	if menu.Code != http.StatusOK {
		t.Fatalf("context menu returned status %d: %s", menu.Code, menu.Body.String())
	}
	if !strings.Contains(menu.Body.String(), menuActionEndpoint) ||
		!strings.Contains(menu.Body.String(), fileID) {
		t.Fatalf("context menu misses the file's actions: %s", menu.Body.String())
	}

	unknown := f.browserAt(currentURL, menuEndpoint, url.Values{"FileID": {"unknown-file"}})
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("context menu of an unknown file returned status %d", unknown.Code)
	}
}
