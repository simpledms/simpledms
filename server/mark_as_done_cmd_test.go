package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestMarkAsDoneCmdRendersEmptyInboxAfterLastFile(t *testing.T) {
	body := markAsDoneResponse(t, []string{"last-file.txt"}, 0)
	if !strings.Contains(body, "No files available yet.") {
		t.Fatalf("expected empty inbox state, got: %s", body)
	}
}

func TestMarkAsDoneCommitFailureDoesNotReleaseSuccessResponse(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "mark-done-commit-failure")
	cmd := h.actions.Inbox.MarkAsDoneCmd
	wrapped := h.router.wrapTxResponse(h.router.wrapCommand(func(
		rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
	) error {
		ctx.TenantCtx().TTx.OnCommit(func(enttenant.Committer) enttenant.Committer {
			return enttenant.CommitFunc(func(context.Context, *enttenant.Tx) error {
				return errors.New("injected commit failure")
			})
		})
		return cmd.Handler(rw, req, ctx)
	}), false, cmd.CommitBeforeResponse())
	request := httptest.NewRequest(http.MethodPost, cmd.Endpoint(), strings.NewReader(
		url.Values{"FileID": {fixture.fileID}}.Encode(),
	))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Current-URL", route.Inbox(
		fixture.tenant.PublicID.String(), fixture.spaceID, fixture.fileID,
	))
	request.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: fixture.session})
	response := httptest.NewRecorder()
	wrapped(response, request)
	if response.Code != http.StatusInternalServerError || response.Header().Get("HX-Trigger") != "" ||
		response.Header().Get("HX-Location") != "" ||
		strings.Contains(response.Body.String(), "Marked file") {
		t.Fatalf("commit failure released success response: status=%d headers=%v body=%s",
			response.Code, response.Header(), response.Body.String())
	}
	filex := fixture.db.ReadOnlyConn.File.Query().Where(
		file.PublicID(entx.NewCIText(fixture.fileID)),
	).OnlyX(
		privacy.DecisionContext(context.Background(), privacy.Allow),
	)
	if !filex.IsInInbox {
		t.Fatalf("commit failure persisted mark-as-done state: %+v", filex)
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
	req.Header.Set("HX-Current-URL", route.Inbox(
		tenantx.PublicID.String(), spacePublicID, filePublicID,
	))
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: createSessionForAccountForRulesTest(t, harness, accountx.ID),
	})

	rr := httptest.NewRecorder()
	harness.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "fileListRadioGroup") ||
		strings.Contains(rr.Body.String(), "No files available yet.") {
		t.Fatalf("mark-as-done command rendered inbox content: %s", rr.Body.String())
	}
	if rr.Header().Get("HX-Trigger") == "" {
		t.Fatalf("mark-as-done command did not emit an invalidation event")
	}

	query := httptest.NewRequest(
		http.MethodPost, harness.actions.Inbox.InboxPage.Endpoint(), strings.NewReader(""),
	)
	query.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	query.Header.Set("HX-Request", "true")
	query.Header.Set("HX-Current-URL", route.Inbox(
		tenantx.PublicID.String(), spacePublicID, filePublicID,
	))
	query.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: createSessionForAccountForRulesTest(t, harness, accountx.ID),
	})
	queryResponse := httptest.NewRecorder()
	harness.router.ServeHTTP(queryResponse, query)
	if queryResponse.Code != http.StatusOK {
		t.Fatalf("inbox query: expected status %d, got %d", http.StatusOK, queryResponse.Code)
	}
	return queryResponse.Body.String()
}
