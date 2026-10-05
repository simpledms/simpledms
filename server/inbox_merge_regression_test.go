package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

func TestInboxMergeRegressionKeepsTargetNameWhenIncomingNameCollides(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "inbox-merge-regression")
	fixture := newInboxMergeCollisionFixture(t, h, f)

	form := url.Values{
		"TargetFileID":   {fixture.targetPublicID},
		"SourceFileID":   {fixture.sourcePublicID},
		"ConfirmWarning": {"true"},
	}
	rr := f.browserAt(route.Inbox(f.tenant.PublicID.String(), f.spaceID, fixture.sourcePublicID),
		h.actions.Browse.FileVersionFromInboxCmd.Endpoint(), form)
	if rr.Code != http.StatusOK {
		t.Fatalf("merge status %d", rr.Code)
	}
	wantTrigger := fmt.Sprintf("%s, %s", event.FileVersionMerged.String(), event.CloseDialog.String())
	if rr.Header().Get("HX-Trigger") != wantTrigger || rr.Header().Get("HX-Reswap") != "none" {
		t.Fatalf("unexpected merge response headers: trigger=%q reswap=%q",
			rr.Header().Get("HX-Trigger"), rr.Header().Get("HX-Reswap"))
	}
	fixture.assertPersisted(t, h, f)
	list := f.browserAt(
		route.BrowseFile(f.tenant.PublicID.String(), f.spaceID, f.rootID, fixture.targetPublicID),
		h.actions.Browse.ListDirPartial.Endpoint(), url.Values{
			"CurrentDirID": {f.rootID}, "SelectedFileID": {fixture.targetPublicID},
		})
	if list.Code != http.StatusOK {
		t.Fatalf("directory query status %d", list.Code)
	}
	if !strings.Contains(list.Body.String(), `id="listDirWrapper"`) ||
		!strings.Contains(list.Body.String(), fixture.targetPublicID) {
		t.Fatal("directory refresh did not render the selected target in its wrapper")
	}
}
