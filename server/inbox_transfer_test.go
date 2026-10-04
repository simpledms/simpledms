package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documentnote"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/db/enttenant/webdavresource"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

func TestInboxTransferHTTPPreservesDocumentAndHistoryClearsClassification(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	var destination *enttenant.Space
	var before *enttenant.File
	var history []*enttenant.DocumentNote
	var versions string
	if err := run(func(ctx *ctxx.SpaceContext) error {
		createSpaceViaCmd(t, h.actions, ctx.TenantContext, "Receiving Inbox")
		destination = ctx.TTx.Space.Query().Where(space.Name("Receiving Inbox")).OnlyX(ctx)
		destination = ctx.TTx.Space.UpdateOne(destination).SetAcceptsInboxTransfers(true).SaveX(ctx)
		docType := ctx.TTx.DocumentType.Create().SetSpaceID(ctx.Space.ID).
			SetName("Invoice").SaveX(ctx)
		tag := ctx.TTx.Tag.Create().SetSpaceID(ctx.Space.ID).SetName("Urgent").
			SetType(tagtype.Simple).SaveX(ctx)
		property := ctx.TTx.Property.Create().SetSpaceID(ctx.Space.ID).SetName("Reference").
			SetType(fieldtype.Text).SaveX(ctx)
		ctx.TTx.TagAssignment.Create().SetSpaceID(ctx.Space.ID).
			SetFileID(doc.ID).SetTagID(tag.ID).SaveX(ctx)
		ctx.TTx.FilePropertyAssignment.Create().SetSpaceID(ctx.Space.ID).
			SetFileID(doc.ID).SetPropertyID(property.ID).SetTextValue("A-123").SaveX(ctx)
		ctx.TTx.WebDAVResource.Create().SetSpaceID(ctx.Space.ID).SetFileID(doc.ID).
			SetCredentialPublicID(entx.NewCIText("transfer-credential")).
			SetDavPath("/original.pdf").SaveX(ctx)
		before = ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).
			SetDocumentTypeID(docType.ID).SetOcrContent("Retained OCR").
			SetOcrSuccessAt(time.Now()).SaveX(ctx)
		notes := filemodel.NewDocumentNotes()
		current, err := notes.Create(ctx, doc.PublicID.String(), "Current", "original")
		if err != nil {
			return err
		}
		if _, err := notes.Edit(ctx, doc.PublicID.String(), current.PublicID.String(),
			"Edited", "corrected"); err != nil {
			return err
		}
		deleted, err := notes.Create(ctx, doc.PublicID.String(), "Deleted", "retained deletion")
		if err != nil {
			return err
		}
		if _, err := notes.Delete(ctx, doc.PublicID.String(), deleted.PublicID.String()); err != nil {
			return err
		}
		predecessor, err := notes.Create(ctx, doc.PublicID.String(), "Old", "retained predecessor")
		if err != nil {
			return err
		}
		if _, err := notes.Replace(ctx, doc.PublicID.String(), predecessor.PublicID.String(),
			"Replacement", "retained successor"); err != nil {
			return err
		}
		history = ctx.TTx.DocumentNote.Query().Where(documentnote.FileID(doc.ID)).AllX(ctx)
		versions = fmt.Sprint(ctx.TTx.FileVersion.Query().Where(fileversion.FileID(doc.ID)).AllX(ctx))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	const message = "Please review\n<script>message</script> & follow up"
	form := url.Values{
		"FileID":             {doc.PublicID.String()},
		"DestinationSpaceID": {destination.PublicID.String()},
		"Message":            {message},
		"AuthorID":           {"999999"},
	}
	rr := request(h.actions.Inbox.TransferFileCmd.Endpoint(), form)
	// The command only emits FileMoved; the Inbox query it invalidates selects the next file and
	// replaces the URL.
	if rr.Code != http.StatusOK ||
		!strings.Contains(rr.Header().Get("HX-Trigger"), event.FileMoved.String()) ||
		rr.Header().Get("HX-Replace-Url") != "" || rr.Header().Get("HX-Redirect") != "" {
		t.Fatalf("transfer response: %d %v %s", rr.Code, rr.Header(), rr.Body.String())
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		destination = ctx.TTx.Space.GetX(ctx, destination.ID)
		destinationCtx := ctxx.NewSpaceContext(ctx.TenantContext, destination)
		got := ctx.TTx.File.GetX(destinationCtx, doc.ID)
		if got.PublicID != before.PublicID || got.Name != before.Name ||
			got.SpaceID != destination.ID || got.ParentID != destinationCtx.SpaceRootDir().ID ||
			!got.IsInInbox || got.Source != before.Source || got.Notes != before.Notes ||
			got.OcrContent != before.OcrContent || got.OcrSuccessAt == nil ||
			before.OcrSuccessAt == nil || !got.OcrSuccessAt.Equal(*before.OcrSuccessAt) ||
			got.DocumentTypeID != 0 {
			t.Fatalf("transferred document changed unexpectedly: %v", got)
		}
		if after := fmt.Sprint(ctx.TTx.FileVersion.Query().
			Where(fileversion.FileID(doc.ID)).AllX(destinationCtx)); after != versions {
			t.Fatalf("versions changed: %s -> %s", versions, after)
		}
		for _, old := range history {
			got := ctx.TTx.DocumentNote.GetX(destinationCtx, old.ID)
			expected := *old
			expected.SpaceID = destination.ID
			if got.String() != expected.String() {
				t.Fatalf("note history changed: %v -> %v", old, got)
			}
		}
		messageNote := ctx.TTx.DocumentNote.Query().Where(
			documentnote.FileID(doc.ID), documentnote.Title("Inbox transfer"),
		).OnlyX(destinationCtx)
		if messageNote.Body != message || messageNote.AuthorID != ctx.User.ID ||
			messageNote.AuthoredAt == nil || messageNote.FileID != doc.ID ||
			ctx.TTx.DocumentNote.Query().Where(documentnote.FileID(doc.ID)).
				CountX(destinationCtx) != len(history)+1 {
			t.Fatalf("message attribution or history count: %v", messageNote)
		}
		for _, count := range []int{
			ctx.TTx.TagAssignment.Query().Where(tagassignment.FileID(doc.ID)).CountX(ctx),
			ctx.TTx.FilePropertyAssignment.Query().
				Where(filepropertyassignment.FileID(doc.ID)).CountX(ctx),
			ctx.TTx.WebDAVResource.Query().Where(enttenantwebdavresource.FileID(doc.ID)).CountX(ctx),
			ctx.TTx.DocumentNote.Query().Where(documentnote.FileID(doc.ID)).CountX(ctx),
		} {
			if count != 0 {
				t.Fatal("old Space retained classification, upload alias, or note visibility")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// The source URL can remain open in another tab, but resubmission cannot move twice.
	rr = request(h.actions.Inbox.TransferFileCmd.Endpoint(), form)
	if rr.Code < 400 || strings.Contains(rr.Header().Get("HX-Trigger"), event.FileMoved.String()) {
		t.Fatalf("stale transfer succeeded: %d %s", rr.Code, rr.Body.String())
	}
}

func TestInboxTransferDialogListsOnlyOptedInActiveOtherInboxesForNonmember(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	var allowed, disabled, deleted *enttenant.Space
	if err := run(func(ctx *ctxx.SpaceContext) error {
		ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).SaveX(ctx)
		ctx.TTx.Space.UpdateOneID(ctx.Space.ID).SetAcceptsInboxTransfers(true).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"FileID": {doc.PublicID.String()}}
	rr := request(h.actions.Inbox.TransferFileDialog.Endpoint(), form)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "No other Inbox is available.") ||
		strings.Contains(rr.Body.String(), `name="DestinationSpaceID"`) {
		t.Fatalf("empty destination dialog: %d %s", rr.Code, rr.Body.String())
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		for _, name := range []string{"<script>Receiving</script>", "Disabled", "Deleted"} {
			createSpaceViaCmd(t, h.actions, ctx.TenantContext, name)
		}
		allowed = ctx.TTx.Space.Query().Where(space.Name("<script>Receiving</script>")).OnlyX(ctx)
		allowed = ctx.TTx.Space.UpdateOne(allowed).SetAcceptsInboxTransfers(true).SaveX(ctx)
		disabled = ctx.TTx.Space.Query().Where(space.Name("Disabled")).OnlyX(ctx)
		if disabled.AcceptsInboxTransfers {
			t.Fatal("new Space accepts transfers without opting in")
		}
		deleted = ctx.TTx.Space.Query().Where(space.Name("Deleted")).OnlyX(ctx)
		deleted = ctx.TTx.Space.UpdateOne(deleted).SetAcceptsInboxTransfers(true).
			SetDeletedAt(time.Now()).SaveX(ctx)
		for _, destination := range []*enttenant.Space{allowed, disabled, deleted} {
			ctx.TTx.SpaceUserAssignment.Delete().Where(
				spaceuserassignment.SpaceID(destination.ID),
				spaceuserassignment.UserID(ctx.User.ID),
			).ExecX(ctxx.NewSpaceContext(ctx.TenantContext, destination))
		}
		ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.User).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rr = request(h.actions.Inbox.TransferFileDialog.Endpoint(), form)
	body := rr.Body.String()
	if rr.Code != http.StatusOK {
		t.Fatalf("destination dialog: %d %s", rr.Code, body)
	}
	for _, want := range []string{
		allowed.PublicID.String(), "&lt;script&gt;Receiving&lt;/script&gt;",
		`name="DestinationSpaceID"`, `name="Message"`, "Message (optional)",
		"Moving clears the document type, tags, and custom fields.",
		"You can choose other Spaces in this organization where you have write access, " +
			"or whose Inboxes accept transfers.",
		"Versions and notes are kept.", "you will lose access to this file", `hx-swap="none"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("dialog missing %q: %s", want, body)
		}
	}
	for _, forbidden := range []string{
		disabled.PublicID.String(), deleted.PublicID.String(), "<script>Receiving</script>",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("dialog exposed %q", forbidden)
		}
	}
}

func TestInboxTransferRejectsUnavailableDestinationAndInvalidSource(t *testing.T) {
	for _, scenario := range []string{
		"disabled destination", "deleted destination", "same Space", "unknown destination",
		"filed source", "deleted source", "directory source", "foreign Space source",
	} {
		t.Run(scenario, func(t *testing.T) {
			h, run, request, doc := newDocumentNotesRequestTest(t)
			var destination *enttenant.Space
			var sourceCtxSpace *enttenant.Space
			var sourceID, destinationID, before string
			if err := run(func(ctx *ctxx.SpaceContext) error {
				createSpaceViaCmd(t, h.actions, ctx.TenantContext, "Receiver")
				destination = ctx.TTx.Space.Query().Where(space.Name("Receiver")).OnlyX(ctx)
				destination = ctx.TTx.Space.UpdateOne(destination).
					SetAcceptsInboxTransfers(true).SaveX(ctx)
				ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).SaveX(ctx)
				sourceCtxSpace = ctx.Space
				sourceID, destinationID = doc.PublicID.String(), destination.PublicID.String()
				switch scenario {
				case "disabled destination":
					ctx.TTx.Space.UpdateOne(destination).SetAcceptsInboxTransfers(false).SaveX(ctx)
					ctx.TTx.SpaceUserAssignment.Delete().Where(
						spaceuserassignment.SpaceID(destination.ID),
						spaceuserassignment.UserID(ctx.User.ID),
					).ExecX(ctxx.NewSpaceContext(ctx.TenantContext, destination))
					ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.User).SaveX(ctx)
				case "deleted destination":
					ctx.TTx.Space.UpdateOne(destination).SetDeletedAt(time.Now()).SaveX(ctx)
				case "same Space":
					ctx.TTx.Space.UpdateOneID(ctx.Space.ID).SetAcceptsInboxTransfers(true).SaveX(ctx)
					destinationID = ctx.SpaceID
				case "unknown destination":
					destinationID = "unknown-inbox"
				case "filed source":
					ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(false).SaveX(ctx)
				case "deleted source":
					ctx.TTx.File.UpdateOneID(doc.ID).SetDeletedAt(time.Now()).SaveX(ctx)
				case "directory source":
					sourceID = ctx.SpaceRootDir().PublicID.String()
				case "foreign Space source":
					foreignCtx := ctxx.NewSpaceContext(ctx.TenantContext, destination)
					foreign := createDocumentForNotesTest(foreignCtx, "foreign.pdf", "private")
					doc = ctx.TTx.File.UpdateOne(foreign).SetIsInInbox(true).SaveX(foreignCtx)
					sourceID, sourceCtxSpace = doc.PublicID.String(), destination
				}
				readCtx := ctxx.NewSpaceContext(ctx.TenantContext, sourceCtxSpace)
				before = ctx.TTx.File.GetX(schema.SkipSoftDelete(readCtx), doc.ID).String()
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			rr := request(h.actions.Inbox.TransferFileCmd.Endpoint(), url.Values{
				"FileID": {sourceID}, "DestinationSpaceID": {destinationID},
				"Message": {"Must not be persisted"},
			})
			if rr.Code < 400 || rr.Header().Get("HX-Replace-Url") != "" ||
				strings.Contains(rr.Header().Get("HX-Trigger"), event.FileMoved.String()) {
				t.Fatalf("rejected transfer response: %d %v %s", rr.Code, rr.Header(), rr.Body.String())
			}
			if err := run(func(ctx *ctxx.SpaceContext) error {
				readCtx := ctxx.NewSpaceContext(ctx.TenantContext, sourceCtxSpace)
				got := ctx.TTx.File.GetX(schema.SkipSoftDelete(readCtx), doc.ID)
				if got.String() != before || ctx.TTx.DocumentNote.Query().
					Where(documentnote.FileID(doc.ID)).CountX(readCtx) != 0 {
					t.Fatalf("rejected transfer modified document or notes: %v", got)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInboxTransferOptInDoesNotGrantDestinationReadAccess(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	var destination *enttenant.Space
	if err := run(func(ctx *ctxx.SpaceContext) error {
		createSpaceViaCmd(t, h.actions, ctx.TenantContext, "Private receiver")
		destination = ctx.TTx.Space.Query().Where(space.Name("Private receiver")).OnlyX(ctx)
		destination = ctx.TTx.Space.UpdateOne(destination).SetAcceptsInboxTransfers(true).SaveX(ctx)
		ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).SaveX(ctx)
		ctx.TTx.SpaceUserAssignment.Delete().Where(
			spaceuserassignment.SpaceID(destination.ID), spaceuserassignment.UserID(ctx.User.ID),
		).ExecX(ctxx.NewSpaceContext(ctx.TenantContext, destination))
		ctx.TTx.SpaceUserAssignment.Update().Where(
			spaceuserassignment.SpaceID(ctx.Space.ID), spaceuserassignment.UserID(ctx.User.ID),
		).SetRole(spacerole.User).SaveX(ctx)
		ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.User).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	form := url.Values{
		"FileID": {doc.PublicID.String()}, "DestinationSpaceID": {destination.PublicID.String()},
		"Message": {" \n\t"},
	}
	rr := request(h.actions.Inbox.TransferFileDialog.Endpoint(), form)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), destination.PublicID.String()) {
		t.Fatalf("opt-in destination hidden from sender: %d %s", rr.Code, rr.Body.String())
	}
	rr = request(h.actions.Inbox.TransferFileCmd.Endpoint(), form)
	if rr.Code != http.StatusOK {
		t.Fatalf("transfer without destination membership: %d %s", rr.Code, rr.Body.String())
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		destinationURL := route.Inbox(ctx.TenantID, destination.PublicID.String(), doc.PublicID.String())
		if strings.Contains(rr.Body.String(), destinationURL) ||
			strings.Contains(rr.Body.String(), "Open file") {
			t.Fatalf("nonmember transfer exposed destination link: %s", rr.Body.String())
		}
		if _, err := ctx.TTx.Space.Get(ctx, destination.ID); !enttenant.IsNotFound(err) {
			t.Fatalf("transfer granted Space visibility: %v", err)
		}
		if ctx.TTx.SpaceUserAssignment.Query().Where(
			spaceuserassignment.SpaceID(destination.ID), spaceuserassignment.UserID(ctx.User.ID),
		).CountX(ctxx.NewSpaceContext(ctx.TenantContext, destination)) != 0 {
			t.Fatal("transfer created destination membership")
		}
		if _, _, err := filemodel.NewDocumentNotes().List(
			ctxx.NewSpaceContext(ctx.TenantContext, destination), doc.PublicID.String(), true,
		); err == nil {
			t.Fatal("sender can read destination notes through a forged context")
		}
		destinationCtx := ctxx.NewSpaceContext(ctx.TenantContext, destination)
		if ctx.TTx.DocumentNote.Query().Where(documentnote.FileID(doc.ID)).
			CountX(destinationCtx) != 0 {
			t.Fatal("whitespace-only message created a note")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rr = request(h.actions.Browse.DocumentNotesPartial.Endpoint(), form)
	if rr.Code < 400 || strings.Contains(rr.Body.String(), "legacy note") {
		t.Fatalf("sender retained source document access: %d %s", rr.Code, rr.Body.String())
	}
}

func TestInboxTransferCallerRollbackRestoresDocumentNotesAndUploadAlias(t *testing.T) {
	h, run, _, doc := newDocumentNotesRequestTest(t)
	var destination *enttenant.Space
	var before string
	if err := run(func(ctx *ctxx.SpaceContext) error {
		createSpaceViaCmd(t, h.actions, ctx.TenantContext, "Rollback receiver")
		destination = ctx.TTx.Space.Query().Where(space.Name("Rollback receiver")).OnlyX(ctx)
		destination = ctx.TTx.Space.UpdateOne(destination).SetAcceptsInboxTransfers(true).SaveX(ctx)
		before = ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).SaveX(ctx).String()
		ctx.TTx.WebDAVResource.Create().SetSpaceID(ctx.Space.ID).SetFileID(doc.ID).
			SetCredentialPublicID(entx.NewCIText("rollback-credential")).
			SetDavPath("/original.pdf").SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("later transaction work failed")
	err := run(func(ctx *ctxx.SpaceContext) error {
		if _, err := filemodel.NewInboxTransferService().Transfer(
			ctx, doc.PublicID.String(), destination.PublicID.String(), "Rolled back message",
		); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("transfer before forced rollback: %v", err)
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		if got := ctx.TTx.File.GetX(ctx, doc.ID); got.String() != before {
			t.Fatalf("rollback retained moved document: %v", got)
		}
		if ctx.TTx.DocumentNote.Query().Where(documentnote.FileID(doc.ID)).CountX(ctx) != 0 ||
			ctx.TTx.WebDAVResource.Query().Where(enttenantwebdavresource.FileID(doc.ID)).CountX(ctx) != 1 {
			t.Fatal("rollback retained message or removed upload alias")
		}
		destinationCtx := ctxx.NewSpaceContext(ctx.TenantContext, destination)
		if ctx.TTx.DocumentNote.Query().Where(documentnote.FileID(doc.ID)).
			CountX(destinationCtx) != 0 {
			t.Fatal("rollback retained destination message")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInboxTransferOptInSettingRequiresTenantOwner(t *testing.T) {
	h, run, request, _ := newDocumentNotesRequestTest(t)
	var source *enttenant.Space
	if err := run(func(ctx *ctxx.SpaceContext) error {
		source = ctx.Space
		if source.AcceptsInboxTransfers {
			t.Fatal("new Space opted in automatically")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	form := url.Values{
		"SpaceID": {source.PublicID.String()}, "Name": {source.Name},
		"Description": {source.Description},
	}
	for _, enabled := range []bool{true, false} {
		form.Set("AcceptsInboxTransfers", fmt.Sprint(enabled))
		rr := request(h.actions.Spaces.EditSpaceCmd.Endpoint(), form)
		if rr.Code != http.StatusOK {
			t.Fatalf("owner setting update: %d %s", rr.Code, rr.Body.String())
		}
		if err := run(func(ctx *ctxx.SpaceContext) error {
			if ctx.Space.AcceptsInboxTransfers != enabled {
				t.Fatalf("accepts transfers = %v, want %v", ctx.Space.AcceptsInboxTransfers, enabled)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		// Retain ownership of the Space: only tenant ownership authorizes this setting.
		ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.User).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	form.Set("AcceptsInboxTransfers", "true")
	rr := request(h.actions.Spaces.EditSpaceCmd.Endpoint(), form)
	if rr.Code < 400 || strings.Contains(rr.Header().Get("HX-Trigger"), event.SpaceUpdated.String()) {
		t.Fatalf("Space owner forged opt-in: %d %v %s", rr.Code, rr.Header(), rr.Body.String())
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		if ctx.Space.AcceptsInboxTransfers {
			t.Fatal("forged request enabled transfers")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInboxTransferRejectsReferencesFromAnotherTenant(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	_, runOtherTenant, _, otherDoc := newDocumentNotesRequestTest(t)
	var otherSpaceID string
	if err := runOtherTenant(func(ctx *ctxx.SpaceContext) error {
		ctx.TTx.File.UpdateOneID(otherDoc.ID).SetIsInInbox(true).SaveX(ctx)
		ctx.TTx.Space.UpdateOneID(ctx.Space.ID).SetAcceptsInboxTransfers(true).SaveX(ctx)
		otherSpaceID = ctx.SpaceID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var localDestinationID, before string
	if err := run(func(ctx *ctxx.SpaceContext) error {
		before = ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).SaveX(ctx).String()
		createSpaceViaCmd(t, h.actions, ctx.TenantContext, "Local receiver")
		destination := ctx.TTx.Space.Query().Where(space.Name("Local receiver")).OnlyX(ctx)
		ctx.TTx.Space.UpdateOne(destination).SetAcceptsInboxTransfers(true).SaveX(ctx)
		localDestinationID = destination.PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, form := range []url.Values{
		{"FileID": {doc.PublicID.String()}, "DestinationSpaceID": {otherSpaceID}},
		{"FileID": {otherDoc.PublicID.String()}, "DestinationSpaceID": {localDestinationID}},
	} {
		rr := request(h.actions.Inbox.TransferFileCmd.Endpoint(), form)
		if rr.Code < 400 || strings.Contains(rr.Header().Get("HX-Trigger"), event.FileMoved.String()) {
			t.Fatalf("cross-tenant transfer accepted: %d %s", rr.Code, rr.Body.String())
		}
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		if got := ctx.TTx.File.GetX(ctx, doc.ID); got.String() != before {
			t.Fatalf("cross-tenant request changed source: %v", got)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := runOtherTenant(func(ctx *ctxx.SpaceContext) error {
		got := ctx.TTx.File.GetX(ctx, otherDoc.ID)
		if got.PublicID != otherDoc.PublicID || got.SpaceID != ctx.Space.ID || !got.IsInInbox {
			t.Fatalf("cross-tenant request changed foreign document: %v", got)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInboxTransferWritableDestinationsIgnoreOptIn(t *testing.T) {
	for _, role := range []string{"Space user", "Space owner", "tenant owner"} {
		for _, disableAfterDialog := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/disable-after-dialog=%t", role, disableAfterDialog), func(t *testing.T) {
				h, run, request, doc := newDocumentNotesRequestTest(t)
				var destination *enttenant.Space
				if err := run(func(ctx *ctxx.SpaceContext) error {
					createSpaceViaCmd(t, h.actions, ctx.TenantContext, "Writable receiver")
					destination = ctx.TTx.Space.Query().
						Where(space.Name("Writable receiver")).OnlyX(ctx)
					destination = ctx.TTx.Space.UpdateOne(destination).
						SetAcceptsInboxTransfers(disableAfterDialog).SaveX(ctx)
					ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).SaveX(ctx)
					destinationCtx := ctxx.NewSpaceContext(ctx.TenantContext, destination)
					if role == "tenant owner" {
						// Tenant ownership permits writing without an explicit Space assignment.
						ctx.TTx.SpaceUserAssignment.Delete().Where(
							spaceuserassignment.SpaceID(destination.ID),
							spaceuserassignment.UserID(ctx.User.ID),
						).ExecX(destinationCtx)
					} else {
						spaceRole := spacerole.Owner
						if role == "Space user" {
							spaceRole = spacerole.User
						}
						ctx.TTx.SpaceUserAssignment.Update().Where(
							spaceuserassignment.SpaceID(destination.ID),
							spaceuserassignment.UserID(ctx.User.ID),
						).SetRole(spaceRole).SaveX(destinationCtx)
						ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.User).SaveX(ctx)
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				form := url.Values{
					"FileID":             {doc.PublicID.String()},
					"DestinationSpaceID": {destination.PublicID.String()},
				}
				dialog := request(h.actions.Inbox.TransferFileDialog.Endpoint(), form)
				if dialog.Code != http.StatusOK ||
					!strings.Contains(dialog.Body.String(), destination.PublicID.String()) {
					t.Fatalf("writable destination missing: %d %s", dialog.Code, dialog.Body.String())
				}
				if disableAfterDialog {
					if err := run(func(ctx *ctxx.SpaceContext) error {
						// Simulate a tenant owner disabling public transfers after selection.
						ctx.TTx.Space.UpdateOneID(destination.ID).SetAcceptsInboxTransfers(false).
							SaveX(privacy.DecisionContext(ctx, privacy.Allow))
						return nil
					}); err != nil {
						t.Fatal(err)
					}
				}
				rr := request(h.actions.Inbox.TransferFileCmd.Endpoint(), form)
				if rr.Code != http.StatusOK ||
					!strings.Contains(rr.Header().Get("HX-Trigger"), event.FileMoved.String()) {
					t.Fatalf("writable transfer: %d %s", rr.Code, rr.Body.String())
				}
				if err := run(func(ctx *ctxx.SpaceContext) error {
					destinationURL := route.Inbox(
						ctx.TenantID, destination.PublicID.String(), doc.PublicID.String(),
					)
					if !strings.Contains(rr.Body.String(), `href="`+destinationURL+`"`) ||
						!strings.Contains(rr.Body.String(), "Open file") {
						t.Fatalf("writable transfer missing destination link: %s", rr.Body.String())
					}
					destination = ctx.TTx.Space.GetX(ctx, destination.ID)
					destinationCtx := ctxx.NewSpaceContext(ctx.TenantContext, destination)
					got := ctx.TTx.File.GetX(destinationCtx, doc.ID)
					if destination.AcceptsInboxTransfers || got.SpaceID != destination.ID ||
						!got.IsInInbox || got.ParentID != destinationCtx.SpaceRootDir().ID {
						t.Fatalf("writable transfer did not persist: %v", got)
					}
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
