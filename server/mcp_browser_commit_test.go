package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestMCPBrowserMarkAsDoneDoesNotReleaseResponseBeforeCommit(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "browser-commit-done")
	cmd := h.actions.Inbox.MarkAsDoneCmd
	wrapped := h.router.wrapTxResponse(h.router.wrapCommand(func(
		rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
	) error {
		ctx.TenantCtx().TTx.OnCommit(func(enttenant.Committer) enttenant.Committer {
			return enttenant.CommitFunc(func(context.Context, *enttenant.Tx) error {
				return errors.New("injected browser commit failure")
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

	if response.Code != http.StatusInternalServerError ||
		strings.Contains(response.Body.String(), "Marked file") ||
		response.Header().Get("HX-Trigger") != "" {
		t.Fatalf("commit failure released filing response: %d %s", response.Code, response.Body.String())
	}
	filex := fixture.db.ReadOnlyConn.File.Query().Where(
		file.PublicID(entx.NewCIText(fixture.fileID)),
	).OnlyX(privacy.DecisionContext(context.Background(), privacy.Allow))
	if !filex.IsInInbox {
		t.Fatalf("failed filing changed persisted Inbox state: %v", filex)
	}
}

func TestMCPBrowserSetTypedPropertyDoesNotReleaseResponseBeforeCommit(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "browser-commit-property")
	var propertyID int64
	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spaceCtx := ctxx.NewSpaceContext(tc, tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(fixture.spaceID)),
		).OnlyX(tc))
		propertyx, err := propertymodel.NewPropertyService().Create(
			spaceCtx, spaceCtx.Space.ID, "Reference", fieldtype.Text, "",
		)
		if err != nil {
			return err
		}
		propertyID = propertyx.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	cmd := h.actions.Browse.SetFilePropertyCmd
	wrapped := h.router.wrapTxResponse(h.router.wrapCommand(func(
		rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
	) error {
		ctx.TenantCtx().TTx.OnCommit(func(enttenant.Committer) enttenant.Committer {
			return enttenant.CommitFunc(func(context.Context, *enttenant.Tx) error {
				return errors.New("injected browser commit failure")
			})
		})
		return cmd.Handler(rw, req, ctx)
	}), false, cmd.CommitBeforeResponse())

	form := url.Values{
		"FileID":     {fixture.fileID},
		"PropertyID": {fmt.Sprint(propertyID)},
		"TextValue":  {"must roll back"},
	}
	request := httptest.NewRequest(http.MethodPost, cmd.Endpoint(), strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Current-URL", route.Inbox(
		fixture.tenant.PublicID.String(), fixture.spaceID, fixture.fileID,
	))
	request.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: fixture.session})
	response := httptest.NewRecorder()
	wrapped(response, request)

	if response.Code != http.StatusInternalServerError ||
		strings.Contains(response.Body.String(), "saved.") ||
		response.Header().Get("HX-Trigger") != "" {
		t.Fatalf("commit failure released metadata response: %d %s", response.Code, response.Body.String())
	}
	if count := fixture.db.ReadOnlyConn.FilePropertyAssignment.Query().CountX(
		privacy.DecisionContext(context.Background(), privacy.Allow),
	); count != 0 {
		t.Fatalf("failed metadata update persisted: %d assignments", count)
	}
}
