package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

func TestConsumeUploadsCmdWithUnknownTokenRedirectsToInboxWithoutBusinessContent(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "consume-uploads-empty")
	fileCount := f.db.ReadOnlyConn.File.Query().CountX(privacy.DecisionContext(context.Background(), privacy.Allow))
	currentURL := route.Dashboard()
	endpoint := "/-/org/" + f.tenant.PublicID.String() + "/space/" + f.spaceID +
		"/inbox/consume-uploads"
	response := f.browserAt(currentURL, endpoint, url.Values{"UploadToken": {"unknown-staged-upload"}})
	if response.Code != http.StatusOK {
		t.Fatalf("consume uploads returned status %d", response.Code)
	}
	wantLocation := route.InboxRoot(f.tenant.PublicID.String(), f.spaceID)
	if got := response.Header().Get("HX-Location"); got != wantLocation {
		t.Fatalf("expected Inbox location %q, got %q", wantLocation, got)
	}
	if got := response.Header().Get("HX-Trigger"); got != event.InboxChanged.String() {
		t.Fatalf("expected inbox invalidation event, got %q", got)
	}
	if strings.Contains(response.Body.String(), "fileListRadioGroup") ||
		strings.Contains(response.Body.String(), "No files available yet.") {
		t.Fatal("consume uploads returned replacement inbox content")
	}
	afterCount := f.db.ReadOnlyConn.File.Query().CountX(
		privacy.DecisionContext(context.Background(), privacy.Allow),
	)
	if afterCount != fileCount {
		t.Fatalf("replayed upload changed file count: before=%d after=%d", fileCount, afterCount)
	}
}

func TestConsumeUploadsCmdRequiresUploadToken(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "consume-uploads-missing-token")
	response := f.browserAt(
		route.Dashboard(),
		"/-/org/"+f.tenant.PublicID.String()+"/space/"+f.spaceID+"/inbox/consume-uploads",
		url.Values{},
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing upload token returned status %d", response.Code)
	}
	if response.Header().Get("HX-Trigger") != "" || response.Header().Get("HX-Location") != "" {
		t.Fatal("missing upload token released success feedback")
	}
}

func TestConsumeUploadsCmdUsesExplicitTenantAndSpaceRoute(t *testing.T) {
	h := newActionTestHarness(t)
	authorized := newMCPFixtureWithWrites(t, h, "consume-uploads-authorized")
	other := newMCPFixtureWithWrites(t, h, "consume-uploads-other")
	currentURL := route.InboxRoot(authorized.tenant.PublicID.String(), authorized.spaceID)
	endpoint := "/-/org/" + other.tenant.PublicID.String() + "/space/" + other.spaceID +
		"/inbox/consume-uploads"
	response := authorized.browserAt(
		currentURL, endpoint, url.Values{"UploadToken": {"unknown-staged-upload"}},
	)
	if response.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant consume uploads returned status %d", response.Code)
	}
}
