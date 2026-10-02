package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/action/download"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documentnote"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
)

func TestInboxTransferHTTPRevokesSenderDownloadPreviewAndNotesAccess(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	var destination *enttenant.Space
	var accountID int64
	var tenantID, sourceID string
	if err := run(func(ctx *ctxx.SpaceContext) error {
		accountID, tenantID, sourceID = ctx.Account.ID, ctx.TenantID, ctx.SpaceID
		createSpaceViaCmd(t, h.actions, ctx.TenantContext, "Private security receiver")
		destination = ctx.TTx.Space.Query().
			Where(space.Name("Private security receiver")).OnlyX(ctx)
		destination = ctx.TTx.Space.UpdateOne(destination).
			SetAcceptsInboxTransfers(true).SaveX(ctx)
		ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).SaveX(ctx)
		destinationCtx := ctxx.NewSpaceContext(ctx.TenantContext, destination)
		ctx.TTx.SpaceUserAssignment.Delete().Where(
			spaceuserassignment.SpaceID(destination.ID),
			spaceuserassignment.UserID(ctx.User.ID),
		).ExecX(destinationCtx)
		ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.User).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// The base harness registers actions only; use the production page handlers.
	h.router.RegisterPage(route.DownloadRoute(), download.NewDownload(h.infra).Handler)
	preview := download.NewPreview(h.infra)
	h.router.RegisterPage(route.PreviewPDFRoute(), preview.PDFInlineHandler)
	h.router.RegisterPage(route.PreviewPDFDownloadRoute(), preview.PDFDownloadHandler)
	h.router.RegisterPage(route.OriginalSourceRoute(), preview.OriginalSourceHandler)
	form := url.Values{"FileID": {doc.PublicID.String()}}
	before := request(h.actions.Browse.DocumentNotesPartial.Endpoint(), form)
	if before.Code != http.StatusOK || !strings.Contains(before.Body.String(), "legacy note") {
		t.Fatalf("source notes before transfer: %d %s", before.Code, before.Body.String())
	}
	form.Set("DestinationSpaceID", destination.PublicID.String())
	form.Set("Message", "Private transfer message")
	moved := request(h.actions.Inbox.TransferFileCmd.Endpoint(), form)
	if moved.Code != http.StatusOK {
		t.Fatalf("transfer: %d %s", moved.Code, moved.Body.String())
	}
	session := createSessionForAccountForRulesTest(t, h, accountID)
	for _, spaceID := range []string{sourceID, destination.PublicID.String()} {
		currentURL := route.Inbox(tenantID, spaceID, doc.PublicID.String())
		for _, endpoint := range []string{
			route.Download(tenantID, spaceID, doc.PublicID.String()),
			route.DownloadWithVersion(tenantID, spaceID, doc.PublicID.String(), "1"),
			route.DownloadInline(tenantID, spaceID, doc.PublicID.String()),
			route.PreviewPDF(tenantID, spaceID, doc.PublicID.String()),
			route.PreviewPDFDownload(tenantID, spaceID, doc.PublicID.String()),
			route.OriginalSource(tenantID, spaceID, doc.PublicID.String()),
		} {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			req.Header.Set("HX-Request", "true")
			req.Header.Set("HX-Current-URL", currentURL)
			req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session})
			rr := httptest.NewRecorder()
			h.router.ServeHTTP(rr, req)
			if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
				t.Fatalf("sender download/preview %s: %d %s",
					endpoint, rr.Code, rr.Body.String())
			}
			if rr.Header().Get("Content-Disposition") != "" {
				t.Fatalf("denied download exposed file headers: %v", rr.Header())
			}
		}
		for _, endpoint := range []string{
			h.actions.Browse.DocumentNotesPartial.Endpoint(),
			h.actions.Browse.DocumentNoteDialog.Endpoint(),
		} {
			readForm := url.Values{
				"FileID": {doc.PublicID.String()}, "ShowHistory": {"true"},
				"Operation": {"view"}, "NoteID": {"legacy"},
			}
			req := httptest.NewRequest(http.MethodPost,
				endpoint, strings.NewReader(readForm.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			req.Header.Set("HX-Current-URL", currentURL)
			req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session})
			rr := httptest.NewRecorder()
			h.router.ServeHTTP(rr, req)
			if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
				t.Fatalf("sender notes via %s: %d %s",
					currentURL, rr.Code, rr.Body.String())
			}
			if strings.Contains(rr.Body.String(), "legacy note") ||
				strings.Contains(rr.Body.String(), "Private transfer message") {
				t.Fatal("denied request disclosed notes")
			}
		}
	}
}

func TestInboxTransferHTTPMessageRendersEscapedForDestinationReader(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	var destination *enttenant.Space
	var accountID int64
	var tenantID string
	if err := run(func(ctx *ctxx.SpaceContext) error {
		accountID, tenantID = ctx.Account.ID, ctx.TenantID
		createSpaceViaCmd(t, h.actions, ctx.TenantContext, "Message receiver")
		destination = ctx.TTx.Space.Query().Where(space.Name("Message receiver")).OnlyX(ctx)
		destination = ctx.TTx.Space.UpdateOne(destination).
			SetAcceptsInboxTransfers(true).SaveX(ctx)
		ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	const message = "Please review\n<script>alert('transfer')</script> & follow up"
	moved := request(h.actions.Inbox.TransferFileCmd.Endpoint(), url.Values{
		"FileID":             {doc.PublicID.String()},
		"DestinationSpaceID": {destination.PublicID.String()},
		"Message":            {message},
	})
	if moved.Code != http.StatusOK {
		t.Fatalf("transfer: %d %s", moved.Code, moved.Body.String())
	}
	form := url.Values{"FileID": {doc.PublicID.String()}, "ShowHistory": {"true"}}
	req := httptest.NewRequest(http.MethodPost,
		h.actions.Browse.DocumentNotesPartial.Endpoint(), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", route.Inbox(
		tenantID, destination.PublicID.String(), doc.PublicID.String(),
	))
	req.AddCookie(&http.Cookie{
		Name:  cookiex.SessionCookieName(),
		Value: createSessionForAccountForRulesTest(t, h, accountID),
	})
	rr := httptest.NewRecorder()
	h.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("destination notes: %d %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		"Inbox transfer", "Please review", "&lt;script&gt;",
		"&lt;/script&gt; &amp; follow up", "legacy note",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("destination notes missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "<script>alert('transfer')</script>") ||
		strings.Contains(body, "<script>Author</script>") {
		t.Fatal("transfer message or author rendered as HTML")
	}
}

func TestInboxTransferHTTPRechecksAuthorizationAfterDialogSelection(t *testing.T) {
	for _, revocation := range []string{
		"source membership", "tenant membership", "destination opt-in", "destination membership",
	} {
		t.Run(revocation, func(t *testing.T) {
			h, run, request, doc := newDocumentNotesRequestTest(t)
			var destination *enttenant.Space
			var tenantDB *sqlx.TenantDB
			var before, beforeNotes string
			if err := run(func(ctx *ctxx.SpaceContext) error {
				var ok bool
				tenantDB, ok = ctx.UnsafeTenantDB()
				if !ok {
					t.Fatal("missing fixture tenant database")
				}
				createSpaceViaCmd(t, h.actions, ctx.TenantContext, "Revocation receiver")
				destination = ctx.TTx.Space.Query().
					Where(space.Name("Revocation receiver")).OnlyX(ctx)
				destination = ctx.TTx.Space.UpdateOne(destination).
					SetAcceptsInboxTransfers(true).SaveX(ctx)
				ctx.TTx.DocumentNote.Create().SetSpaceID(ctx.Space.ID).SetFileID(doc.ID).
					SetTitle("Existing").SetBody("Existing attributed note").
					SetAuthorID(ctx.User.ID).SetAuthoredAt(time.Now()).SaveX(ctx)
				before = ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).SaveX(ctx).String()
				beforeNotes = fmt.Sprint(ctx.TTx.DocumentNote.Query().
					Where(documentnote.FileID(doc.ID)).AllX(ctx))
				if revocation == "destination opt-in" || revocation == "destination membership" {
					if revocation == "destination opt-in" {
						ctx.TTx.SpaceUserAssignment.Delete().Where(
							spaceuserassignment.SpaceID(destination.ID),
							spaceuserassignment.UserID(ctx.User.ID),
						).ExecX(ctxx.NewSpaceContext(ctx.TenantContext, destination))
					} else {
						ctx.TTx.Space.UpdateOneID(destination.ID).
							SetAcceptsInboxTransfers(false).SaveX(ctx)
					}
					ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.User).SaveX(ctx)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			form := url.Values{
				"FileID":             {doc.PublicID.String()},
				"DestinationSpaceID": {destination.PublicID.String()},
				"Message":            {"This message must not be saved"},
			}
			dialog := request(h.actions.Inbox.TransferFileDialog.Endpoint(), form)
			if dialog.Code != http.StatusOK ||
				!strings.Contains(dialog.Body.String(), destination.PublicID.String()) {
				t.Fatalf("initial dialog: %d %s", dialog.Code, dialog.Body.String())
			}
			if err := run(func(ctx *ctxx.SpaceContext) error {
				switch revocation {
				case "source membership":
					ctx.TTx.SpaceUserAssignment.Delete().Where(
						spaceuserassignment.SpaceID(ctx.Space.ID),
						spaceuserassignment.UserID(ctx.User.ID),
					).ExecX(ctx)
					ctx.TTx.User.UpdateOneID(ctx.User.ID).
						SetRole(tenantrole.User).SaveX(ctx)
				case "tenant membership":
					ctx.MainTx.TenantAccountAssignment.Update().Where(
						tenantaccountassignment.AccountID(ctx.Account.ID),
						tenantaccountassignment.TenantID(ctx.Tenant.ID),
					).SetExpiresAt(time.Now().Add(-time.Minute)).SaveX(ctx)
				case "destination opt-in":
					// Simulate a tenant owner revoking transfers from nonmembers.
					ctx.TTx.Space.UpdateOneID(destination.ID).
						SetAcceptsInboxTransfers(false).SaveX(privacy.DecisionContext(ctx, privacy.Allow))
				case "destination membership":
					ctx.TTx.SpaceUserAssignment.Delete().Where(
						spaceuserassignment.SpaceID(destination.ID),
						spaceuserassignment.UserID(ctx.User.ID),
					).ExecX(ctxx.NewSpaceContext(ctx.TenantContext, destination))
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			rr := request(h.actions.Inbox.TransferFileCmd.Endpoint(), form)
			if revocation == "tenant membership" {
				// Losing the only active tenant membership invalidates the Session.
				if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/" {
					t.Fatalf("revoked tenant Session: %d %v %s",
						rr.Code, rr.Header(), rr.Body.String())
				}
			} else if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
				t.Fatalf("revoked transfer: %d %s", rr.Code, rr.Body.String())
			}
			if strings.Contains(rr.Header().Get("HX-Trigger"), event.FileMoved.String()) ||
				rr.Header().Get("HX-Replace-Url") != "" {
				t.Fatalf("revoked transfer emitted success headers: %v", rr.Header())
			}
			// Inspect committed state after the sender has lost authorization.
			readCtx := privacy.DecisionContext(context.Background(), privacy.Allow)
			got := tenantDB.ReadOnlyConn.File.GetX(readCtx, doc.ID)
			gotNotes := fmt.Sprint(tenantDB.ReadOnlyConn.DocumentNote.Query().
				Where(documentnote.FileID(doc.ID)).AllX(readCtx))
			if got.String() != before || gotNotes != beforeNotes {
				t.Fatalf("revoked transfer mutated document or notes: %v", got)
			}
		})
	}
}
