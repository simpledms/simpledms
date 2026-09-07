package server

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documentnote"
	"github.com/simpledms/simpledms/ui/uix/event"
)

func TestDocumentNotesHTTPInboxRootFilingPreservesHistory(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	cmd := h.actions.Browse.DocumentNoteCmd.Endpoint()
	var rootID string
	if err := run(func(ctx *ctxx.SpaceContext) error {
		ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(true).SaveX(ctx)
		rootID = ctx.SpaceRootDir().PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"current", "deleted", "replaced"} {
		assertDocumentNoteResponse(t, request(cmd,
			documentNoteForm(doc, "create", "", text)), http.StatusOK)
		if text == "current" {
			continue
		}
		var id string
		if err := run(func(ctx *ctxx.SpaceContext) error {
			id = ctx.TTx.DocumentNote.Query().Where(documentnote.Body(text)).OnlyX(ctx).
				PublicID.String()
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		operation := "delete"
		if text == "replaced" {
			operation = "replace"
		}
		assertDocumentNoteResponse(t, request(cmd,
			documentNoteForm(doc, operation, id, "successor")), http.StatusOK)
	}
	var before string
	if err := run(func(ctx *ctxx.SpaceContext) error {
		before = fmt.Sprint(ctx.TTx.DocumentNote.Query().
			Order(enttenant.Asc(documentnote.FieldID)).AllX(ctx))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Reproduce the initial dialog: no destination has been explicitly selected.
	rr := request(h.actions.Inbox.MoveFileCmd.FormEndpoint(),
		url.Values{"FileID": {doc.PublicID.String()}})
	if rr.Code != http.StatusOK {
		t.Fatalf("initial Move form: %d %s", rr.Code, rr.Body.String())
	}
	var destination string
	for _, input := range regexp.MustCompile(`<input\b[^>]*>`).FindAllString(rr.Body.String(), -1) {
		if strings.Contains(input, `name="CurrentDirID"`) && strings.Contains(input, `type="hidden"`) {
			match := regexp.MustCompile(`value="([^"]*)"`).FindStringSubmatch(input)
			if len(match) == 2 {
				destination = match[1]
			}
		}
	}
	if destination != rootID {
		t.Fatalf("initial hidden destination = %q, want root %q", destination, rootID)
	}
	// Submit the rendered destination unchanged, as a user filing directly to root does.
	rr = request(h.actions.Inbox.MoveFileCmd.Endpoint(), url.Values{
		"FileID": {doc.PublicID.String()}, "CurrentDirID": {destination}, "Filename": {doc.Name},
	})
	if rr.Code != http.StatusOK || !strings.Contains(rr.Header().Get("HX-Trigger"),
		event.FileMoved.String()) {
		t.Fatalf("file to root: %d %s", rr.Code, rr.Body.String())
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		file := ctx.TTx.File.GetX(ctx, doc.ID)
		if file.IsInInbox || file.ParentID != ctx.SpaceRootDir().ID || file.Notes != "legacy note" {
			t.Fatalf("unexpected filed document state: %v", file)
		}
		// Generated String includes identity, text, authorship, timestamps and links.
		after := fmt.Sprint(ctx.TTx.DocumentNote.Query().
			Order(enttenant.Asc(documentnote.FieldID)).AllX(ctx))
		if before != after {
			t.Fatalf("filing changed retained history: %s -> %s", before, after)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentNotesHTTPMissingParentRestoreAndNonFolderFiling(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	cmd := h.actions.Browse.DocumentNoteCmd.Endpoint()
	assertDocumentNoteResponse(t, request(cmd,
		documentNoteForm(doc, "replace", "legacy", "current replacement")), http.StatusOK)
	assertDocumentNoteResponse(t, request(cmd,
		documentNoteForm(doc, "create", "", "deleted note")), http.StatusOK)
	var deletedID, before string
	if err := run(func(ctx *ctxx.SpaceContext) error {
		deletedID = ctx.TTx.DocumentNote.Query().Where(documentnote.Body("deleted note")).
			OnlyX(ctx).PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	assertDocumentNoteResponse(t, request(cmd,
		documentNoteForm(doc, "delete", deletedID, "")), http.StatusOK)
	if err := run(func(ctx *ctxx.SpaceContext) error {
		before = fmt.Sprint(ctx.TTx.DocumentNote.Query().
			Order(enttenant.Asc(documentnote.FieldID)).AllX(ctx))
		parent := ctx.TTx.File.Create().SetSpaceID(ctx.Space.ID).
			SetName("Removed parent").SetIsDirectory(true).SetIndexedAt(time.Now()).
			SetParentID(ctx.SpaceRootDir().ID).SaveX(ctx)
		ctx.TTx.File.UpdateOneID(doc.ID).SetParentID(parent.ID).SetDeletedAt(time.Now()).SaveX(ctx)
		ctx.TTx.File.UpdateOneID(parent.ID).SetDeletedAt(time.Now()).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rr := request(h.actions.Trash.RestoreFileCmd.Endpoint(),
		url.Values{"FileID": {doc.PublicID.String()}})
	if rr.Code != http.StatusOK ||
		!strings.Contains(rr.Body.String(), "Restored to Inbox") {
		t.Fatalf("restore with removed parent: %d %s", rr.Code, rr.Body.String())
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		restored := ctx.TTx.File.GetX(ctx, doc.ID)
		if !restored.DeletedAt.IsZero() || !restored.IsInInbox ||
			restored.ParentID != ctx.SpaceRootDir().ID {
			t.Fatalf("unexpected restored document: %v", restored)
		}
		ctx.TTx.Space.UpdateOneID(ctx.Space.ID).SetIsFolderMode(false).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rr = request(h.actions.Inbox.MarkAsDoneCmd.Endpoint(),
		url.Values{"FileID": {doc.PublicID.String()}})
	if rr.Code != http.StatusOK {
		t.Fatalf("non-folder filing: %d %s", rr.Code, rr.Body.String())
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		file := ctx.TTx.File.GetX(ctx, doc.ID)
		if file.IsInInbox || file.ParentID != ctx.SpaceRootDir().ID {
			t.Fatalf("non-folder filing did not complete: %v", file)
		}
		after := fmt.Sprint(ctx.TTx.DocumentNote.Query().
			Order(enttenant.Asc(documentnote.FieldID)).AllX(ctx))
		if before != after {
			t.Fatalf("restore/filing changed note history: %s -> %s", before, after)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
