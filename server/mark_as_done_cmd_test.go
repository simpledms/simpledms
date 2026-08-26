package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
)

func TestMarkAsDoneCmdRendersEmptyInboxAfterLastFile(t *testing.T) {
	body := markAsDoneResponse(t, []string{"last-file.txt"}, 0)
	if !strings.Contains(body, "No files available yet.") {
		t.Fatalf("expected empty inbox state, got: %s", body)
	}
}

func TestMarkAsDoneCmdSelectsRemainingFile(t *testing.T) {
	body := markAsDoneResponse(t, []string{"remaining-file.txt", "done-file.txt"}, 1)
	if count := strings.Count(body, `name="fileListRadioGroup"`); count != 1 {
		t.Fatalf("expected one file in inbox list, got %d: %s", count, body)
	}
	if !regexp.MustCompile(`name="fileListRadioGroup"[^>]*checked`).MatchString(body) {
		t.Fatalf("expected remaining inbox file to be selected: %s", body)
	}
	if !strings.Contains(body, "remaining-file.txt") {
		t.Fatalf("expected remaining file in inbox list: %s", body)
	}
}

func markAsDoneResponse(t *testing.T, filenames []string, markedIndex int) string {
	t.Helper()
	harness := newActionTestHarness(t)
	accountx, tenantx := signUpAccount(t, harness, "mark-as-done@example.com")
	tenantDB := initTenantDB(t, harness, tenantx)
	tenantx = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenantx.ID)

	var spacePublicID string
	var filePublicID string
	err := withTenantContext(t, harness, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx,
		_ *enttenant.Tx,
		tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Mark As Done Space")
		spacex := tenantCtx.TTx.Space.Query().Where(space.Name("Mark As Done Space")).OnlyX(tenantCtx)
		spaceCtx := ctxx.NewSpaceContext(tenantCtx, spacex)
		spacePublicID = spacex.PublicID.String()
		for qi, filename := range filenames {
			filex := createRegularFileForTest(spaceCtx, spaceCtx.SpaceRootDir().ID, filename).
				Data.Update().SetIsInInbox(true).SaveX(spaceCtx)
			if err := seedStoredFilesForBenchmarkRows(spaceCtx, []*enttenant.File{filex}); err != nil {
				return err
			}
			if qi == markedIndex {
				filePublicID = filex.PublicID.String()
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{"FileID": {filePublicID}}
	req := httptest.NewRequest(
		http.MethodPost,
		harness.actions.Inbox.MarkAsDoneCmd.Endpoint(),
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", route.Inbox(tenantx.PublicID.String(), spacePublicID, filePublicID))
	req.Header.Set("X-Query-Endpoint", harness.actions.Inbox.InboxPage.Endpoint())
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: createSessionForAccountForRulesTest(t, harness, accountx.ID),
	})

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	return rr.Body.String()
}
