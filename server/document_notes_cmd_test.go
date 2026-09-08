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

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documentnote"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
)

// Every request goes through the registered route, including wrapTx's commit/rollback.
// The setup closure owns separate transactions so assertions observe committed state.
func newDocumentNotesRequestTest(t *testing.T) (
	*actionTestHarness,
	func(func(*ctxx.SpaceContext) error) error,
	func(string, url.Values) *httptest.ResponseRecorder,
	*enttenant.File,
) {
	t.Helper()
	h := newActionTestHarness(t)
	account, tenant := signUpAccount(t, h, "notes-http@example.com")
	db := initTenantDB(t, h, tenant)
	tenant = h.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenant.ID)
	run := func(fn func(*ctxx.SpaceContext) error) error {
		return withTenantContext(t, h, account, tenant, db, func(
			_ *entmain.Tx, _ *enttenant.Tx, ctx *ctxx.TenantContext,
		) error {
			spacex, err := ctx.TTx.Space.Query().Where(space.Name("HTTP notes")).Only(ctx)
			if enttenant.IsNotFound(err) {
				createSpaceViaCmd(t, h.actions, ctx, "HTTP notes")
				spacex = ctx.TTx.Space.Query().Where(space.Name("HTTP notes")).OnlyX(ctx)
			} else if err != nil {
				return err
			}
			return fn(ctxx.NewSpaceContext(ctx, spacex))
		})
	}
	var doc *enttenant.File
	var currentURL string
	if err := run(func(ctx *ctxx.SpaceContext) error {
		doc = createDocumentForNotesTest(ctx, "http-notes.pdf", "legacy note")
		ctx.TTx.User.UpdateOneID(ctx.User.ID).SetFirstName("<script>Author</script>").
			SetLastName("& Writer").SaveX(ctx)
		currentURL = route.BrowseFile(tenant.PublicID.String(), ctx.SpaceID,
			ctx.SpaceRootDir().PublicID.String(), doc.PublicID.String())
		return seedStoredFilesForBenchmarkRows(ctx, []*enttenant.File{doc})
	}); err != nil {
		t.Fatal(err)
	}
	session := createSessionForAccountForRulesTest(t, h, account.ID)
	request := func(endpoint string, form url.Values) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Current-URL", currentURL)
		req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session})
		rr := httptest.NewRecorder()
		h.router.ServeHTTP(rr, req)
		return rr
	}
	return h, run, request, doc
}

func documentNoteForm(doc *enttenant.File, operation, noteID, body string) url.Values {
	return url.Values{
		"FileID": {doc.PublicID.String()}, "Operation": {operation},
		"NoteID": {noteID}, "Body": {body}, "Title": {"Document annotation"},
	}
}

func assertDocumentNoteResponse(t *testing.T, rr *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("status = %d, want %d: %s", rr.Code, status, rr.Body.String())
	}
	trigger := rr.Header().Get("HX-Trigger")
	if strings.Contains(trigger, "documentNotesChanged") != (status == http.StatusOK) ||
		strings.Contains(trigger, "closeDialog") {
		t.Fatalf("unexpected success events for status %d: %q", status, trigger)
	}
}

func countDocumentNoteMutationControls(body, endpoint string) int {
	return strings.Count(body, endpoint) -
		strings.Count(body, `&#34;Operation&#34;:&#34;view&#34;`)
}

func TestDocumentNotesDetailsDialog(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	endpoint := h.actions.Browse.DocumentNoteDialog.Endpoint()
	for _, trashed := range []bool{false, true} {
		if trashed {
			if err := run(func(ctx *ctxx.SpaceContext) error {
				_, err := ctx.TTx.File.UpdateOneID(doc.ID).SetDeletedAt(time.Now()).Save(ctx)
				return err
			}); err != nil {
				t.Fatal(err)
			}
		}
		rr := request(endpoint, documentNoteForm(doc, "view", "legacy", ""))
		if rr.Code != http.StatusOK {
			t.Fatalf("details: %d %s", rr.Code, rr.Body.String())
		}
		for _, want := range []string{"legacy note", "Author: Unknown", "Created: Unknown"} {
			if !strings.Contains(rr.Body.String(), want) {
				t.Fatalf("missing %q from details", want)
			}
		}
		if strings.Contains(rr.Body.String(), `<textarea`) ||
			strings.Contains(rr.Body.String(), h.actions.Browse.DocumentNoteCmd.Endpoint()) {
			t.Fatal("read-only details exposed mutation form")
		}
	}
	rr := request(endpoint, documentNoteForm(doc, "view", "missing", ""))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("unknown note details status: %d", rr.Code)
	}
}

func TestDocumentNotesCmdHTTPHistoryAndEscaping(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	cmd := h.actions.Browse.DocumentNoteCmd.Endpoint()
	form := documentNoteForm(doc, "create", "", "first line\n<script>note</script> & text")
	form.Set("Title", "<script>Review title</script>")
	form.Set("AuthorID", "999999")
	form.Set("AuthoredAt", "2000-01-01T00:00:00Z")
	assertDocumentNoteResponse(t, request(cmd, form), http.StatusOK)
	var first *enttenant.DocumentNote
	if err := run(func(ctx *ctxx.SpaceContext) error {
		first = ctx.TTx.DocumentNote.Query().OnlyX(ctx)
		if first.AuthorID != ctx.User.ID || first.AuthoredAt == nil ||
			first.AuthoredAt.Year() == 2000 || first.Body != form.Get("Body") ||
			first.Title != form.Get("Title") {
			t.Fatalf("spoofed attribution or changed text: %v", first)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	partial := h.actions.Browse.DocumentNotesPartial.Endpoint()
	read := func(history bool) string {
		rr := request(partial, url.Values{
			"FileID": {doc.PublicID.String()}, "ShowHistory": {fmt.Sprint(history)},
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("list: %d %s", rr.Code, rr.Body.String())
		}
		return rr.Body.String()
	}
	body := read(false)
	for _, want := range []string{
		`class="js-list-item`, `role="menu"`, `popover="manual"`,
		`popovertarget="documentNoteMenu-`,
		`tabindex="0"`,
		"first line\n&lt;script&gt;note&lt;/script&gt; &amp; text",
		"&lt;script&gt;Review title&lt;/script&gt;", "legacy note",
		`id="documentNotesHistory-button"`, `aria-pressed="false"`,
		`aria-label="Show deleted and replaced notes"`, `aria-label="Add note"`,
		`aria-label="Actions"`,
		"documentNotesChanged from:body",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "<script>note</script>") ||
		strings.Contains(body, "<script>Review title</script>") ||
		strings.Contains(body, "<script>Author</script>") {
		t.Fatal("unescaped note or author")
	}
	if strings.Contains(body, `role="switch"`) || strings.Contains(body, `type="checkbox"`) {
		t.Fatal("notes toolbar still renders a switch")
	}
	id := first.PublicID.String()
	dialog := request(h.actions.Browse.DocumentNoteDialog.Endpoint(),
		documentNoteForm(doc, "edit", id, ""))
	if dialog.Code != http.StatusOK || !strings.Contains(dialog.Body.String(), `hx-swap="none"`) {
		t.Fatalf("dialog does not preserve typed text on failure: %s", dialog.Body.String())
	}
	if !strings.Contains(dialog.Body.String(), `value="&lt;script&gt;Review title&lt;/script&gt;"`) {
		t.Fatalf("edit title not prefilled: %s", dialog.Body.String())
	}
	editForm := documentNoteForm(doc, "edit", id, "corrected note")
	editForm.Set("Title", "Corrected title")
	assertDocumentNoteResponse(t, request(cmd, editForm), http.StatusOK)
	if err := run(func(ctx *ctxx.SpaceContext) error {
		edited := ctx.TTx.DocumentNote.GetX(ctx, first.ID)
		if edited.Title != "Corrected title" || edited.Body != "corrected note" ||
			edited.AuthorID != first.AuthorID ||
			!edited.AuthoredAt.Equal(*first.AuthoredAt) || edited.EditedAt == nil ||
			edited.EditorID != ctx.User.ID || ctx.TTx.DocumentNote.Query().CountX(ctx) != 1 {
			t.Fatalf("edit did not preserve identity/attribution: %v", edited)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	replaceForm := documentNoteForm(doc, "replace", id, "successor note")
	replaceForm.Set("Title", "Successor title")
	assertDocumentNoteResponse(t, request(cmd, replaceForm), http.StatusOK)
	var successor *enttenant.DocumentNote
	if err := run(func(ctx *ctxx.SpaceContext) error {
		old := ctx.TTx.DocumentNote.GetX(ctx, first.ID)
		successor = ctx.TTx.DocumentNote.GetX(ctx, old.ReplacedByID)
		if old.Body != "corrected note" || old.AuthorID != first.AuthorID ||
			successor.AuthorID != ctx.User.ID || old.Title != "Corrected title" ||
			successor.Title != "Successor title" {
			t.Fatalf("replacement attribution: %v / %v", old, successor)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if body := read(false); strings.Contains(body, "corrected note") ||
		!strings.Contains(body, "successor note") {
		t.Fatalf("current filter: %s", body)
	}
	assertDocumentNoteResponse(t, request(cmd,
		documentNoteForm(doc, "delete", successor.PublicID.String(), "")), http.StatusOK)
	for _, historicalID := range []string{id, successor.PublicID.String()} {
		for _, operation := range []string{"edit", "replace", "delete"} {
			assertDocumentNoteResponse(t, request(cmd,
				documentNoteForm(doc, operation, historicalID, "stale")), http.StatusConflict)
		}
	}
	if body := read(false); strings.Contains(body, "corrected note") ||
		strings.Contains(body, "successor note") {
		t.Fatalf("deletion reactivated history: %s", body)
	}
	body = read(true)
	if strings.Contains(body, "corrected note") || strings.Contains(body, "successor note") {
		t.Fatal("historical summary exposed retained body")
	}
	for _, want := range []string{"Corrected title", "Successor title", "<em>Replaced by: Successor title</em>", "<em>Deleted</em>",
		"Created:", "Author:"} {
		if !strings.Contains(body, want) {
			t.Fatalf("history missing %q: %s", want, body)
		}
	}
	// Only the legacy entry remains changeable: one add and three legacy actions.
	deletedDetails := request(h.actions.Browse.DocumentNoteDialog.Endpoint(),
		documentNoteForm(doc, "view", successor.PublicID.String(), ""))
	if deletedDetails.Code != http.StatusOK ||
		!strings.Contains(deletedDetails.Body.String(), "successor note") {
		t.Fatal("deleted note details lost retained body")
	}
	details := request(h.actions.Browse.DocumentNoteDialog.Endpoint(),
		documentNoteForm(doc, "view", id, ""))
	for _, want := range []string{"corrected note", "Edited by", "Replaced",
		"Corrected title", "Author: &lt;script&gt;Author&lt;/script&gt;", "Created:",
		"#documentNote-" + successor.PublicID.String()} {
		if details.Code != http.StatusOK || !strings.Contains(details.Body.String(), want) {
			t.Fatalf("historical details missing %q: %s", want, details.Body.String())
		}
	}
	if count := countDocumentNoteMutationControls(body, h.actions.Browse.DocumentNoteDialog.Endpoint()); count != 4 {
		t.Fatalf("historical entries exposed mutation controls: %d", count)
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		if ctx.TTx.DocumentNote.Query().CountX(ctx) != 2 ||
			ctx.TTx.DocumentNote.GetX(ctx, successor.ID).DeletedAt == nil ||
			ctx.TTx.File.GetX(ctx, doc.ID).Notes != "legacy note" {
			t.Fatal("reads materialized legacy or history was physically deleted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentNotesCmdHTTPErrorsAndPermissions(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	cmd := h.actions.Browse.DocumentNoteCmd.Endpoint()
	assertDocumentNoteResponse(t, request(cmd,
		documentNoteForm(doc, "create", "", "own note")), http.StatusOK)
	var own, other *enttenant.DocumentNote
	var otherDoc *enttenant.File
	if err := run(func(ctx *ctxx.SpaceContext) error {
		own = ctx.TTx.DocumentNote.Query().OnlyX(ctx)
		otherDoc = createDocumentForNotesTest(ctx, "other.pdf", "")
		author := ctx.TTx.User.Create().SetAccountID(ctx.User.AccountID + 10000).
			SetEmail(entx.NewCIText("other-notes@example.com")).SetFirstName("Other").
			SetRole(tenantrole.User).SaveX(ctx)
		other = ctx.TTx.DocumentNote.Create().SetSpaceID(ctx.Space.ID).SetFileID(doc.ID).
			SetBody("other note").SetAuthorID(author.ID).SetAuthoredAt(time.Now()).SaveX(ctx)
		ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.User).SaveX(ctx)
		ctx.TTx.SpaceUserAssignment.Update().Where(spaceuserassignment.SpaceID(ctx.Space.ID),
			spaceuserassignment.UserID(ctx.User.ID)).SetRole(spacerole.User).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Compare all persisted note fields, not only row counts, after rejected requests.
	var before []*enttenant.DocumentNote
	if err := run(func(ctx *ctxx.SpaceContext) error {
		before = ctx.TTx.DocumentNote.Query().Order(enttenant.Asc(documentnote.FieldID)).AllX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"create", "edit", "replace"} {
		for _, title := range []string{"", " \n\t"} {
			form := documentNoteForm(doc, operation, own.PublicID.String(), "valid body")
			form.Set("Title", title)
			assertDocumentNoteResponse(t, request(cmd, form), http.StatusBadRequest)
		}
		assertDocumentNoteResponse(t, request(cmd,
			documentNoteForm(doc, operation, own.PublicID.String(), " \n\t")), http.StatusBadRequest)
	}
	assertDocumentNoteResponse(t, request(cmd,
		documentNoteForm(doc, "unsupported", "", "text")), http.StatusBadRequest)
	for _, operation := range []string{"edit", "replace", "delete"} {
		assertDocumentNoteResponse(t, request(cmd,
			documentNoteForm(doc, operation, other.PublicID.String(), "forbidden")), http.StatusForbidden)
		assertDocumentNoteResponse(t, request(cmd,
			documentNoteForm(doc, operation, "legacy", "forbidden")), http.StatusForbidden)
		assertDocumentNoteResponse(t, request(cmd,
			documentNoteForm(otherDoc, operation, own.PublicID.String(), "mismatch")), http.StatusNotFound)
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		after := ctx.TTx.DocumentNote.Query().Order(enttenant.Asc(documentnote.FieldID)).AllX(ctx)
		// String excludes the transaction/client internals while including persisted fields.
		if fmt.Sprint(before) != fmt.Sprint(after) {
			t.Fatalf("failed command mutated notes: %v -> %v", before, after)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	partial := request(h.actions.Browse.DocumentNotesPartial.Endpoint(),
		url.Values{"FileID": {doc.PublicID.String()}, "ShowHistory": {"true"}})
	if partial.Code != http.StatusOK ||
		countDocumentNoteMutationControls(partial.Body.String(), h.actions.Browse.DocumentNoteDialog.Endpoint()) != 4 {
		t.Fatalf("member should see add plus own actions only: %s", partial.Body.String())
	}
	assertDocumentNoteResponse(t, request(cmd,
		documentNoteForm(doc, "edit", own.PublicID.String(), "author edit")), http.StatusOK)
	view := request(h.actions.Browse.DocumentNoteDialog.Endpoint(),
		documentNoteForm(doc, "view", other.PublicID.String(), ""))
	if view.Code != http.StatusOK || !strings.Contains(view.Body.String(), "other note") {
		t.Fatal("member cannot inspect another author's note")
	}
	view = request(h.actions.Browse.DocumentNoteDialog.Endpoint(),
		documentNoteForm(otherDoc, "view", own.PublicID.String(), ""))
	if view.Code != http.StatusNotFound {
		t.Fatal("details accepted a mismatched document/note pair")
	}
	assertDocumentNoteResponse(t, request(cmd,
		documentNoteForm(doc, "create", "", "member added")), http.StatusOK)
	for _, owner := range []string{"Space", "tenant"} {
		if err := run(func(ctx *ctxx.SpaceContext) error {
			if owner == "Space" {
				ctx.TTx.SpaceUserAssignment.Update().Where(
					spaceuserassignment.SpaceID(ctx.Space.ID), spaceuserassignment.UserID(ctx.User.ID),
				).SetRole(spacerole.Owner).SaveX(ctx)
			} else {
				ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.Owner).SaveX(ctx)
				ctx.TTx.SpaceUserAssignment.Delete().Where(
					spaceuserassignment.SpaceID(ctx.Space.ID), spaceuserassignment.UserID(ctx.User.ID),
				).ExecX(ctx)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		assertDocumentNoteResponse(t, request(cmd,
			documentNoteForm(doc, "edit", other.PublicID.String(), owner+" edit")), http.StatusOK)
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		edited := ctx.TTx.DocumentNote.GetX(ctx, other.ID)
		if edited.AuthorID != other.AuthorID || edited.EditorID != ctx.User.ID {
			t.Fatalf("owner overwrote original author: %v", edited)
		}
		ctx.TTx.File.UpdateOneID(doc.ID).SetDeletedAt(time.Now()).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"create", "edit", "replace", "delete"} {
		assertDocumentNoteResponse(t, request(cmd,
			documentNoteForm(doc, operation, own.PublicID.String(), "Trash")), http.StatusForbidden)
	}
	partial = request(h.actions.Browse.DocumentNotesPartial.Endpoint(),
		url.Values{"FileID": {doc.PublicID.String()}, "ShowHistory": {"true"}})
	if partial.Code != http.StatusOK ||
		countDocumentNoteMutationControls(partial.Body.String(), h.actions.Browse.DocumentNoteDialog.Endpoint()) != 0 ||
		!strings.Contains(partial.Body.String(), "Show deleted and replaced notes") {
		t.Fatalf("Trash controls: %d %s", partial.Code, partial.Body.String())
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		if ctx.TTx.DocumentNote.Query().CountX(ctx) != 3 ||
			ctx.TTx.DocumentNote.GetX(ctx, own.ID).Body != "author edit" ||
			ctx.TTx.DocumentNote.GetX(ctx, own.ID).DeletedAt != nil {
			t.Fatal("Trash request changed note state")
		}
		ctx.TTx.File.UpdateOneID(doc.ID).ClearDeletedAt().SaveX(schema.SkipSoftDelete(ctx))
		// The tenant owner has no Space assignment. Demotion must revoke even their
		// own notes, including history reads through a previously valid URL.
		ctx.TTx.User.UpdateOneID(ctx.User.ID).SetRole(tenantrole.User).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"create", "edit", "replace", "delete"} {
		assertDocumentNoteResponse(t, request(cmd,
			documentNoteForm(doc, operation, own.PublicID.String(), "revoked")), http.StatusForbidden)
	}
	for _, history := range []string{"false", "true"} {
		rr := request(h.actions.Browse.DocumentNotesPartial.Endpoint(),
			url.Values{"FileID": {doc.PublicID.String()}, "ShowHistory": {history}})
		assertDocumentNoteResponse(t, rr, http.StatusForbidden)
		if strings.Contains(rr.Body.String(), "author edit") {
			t.Fatal("revoked author can read retained text")
		}
	}
	view = request(h.actions.Browse.DocumentNoteDialog.Endpoint(),
		documentNoteForm(doc, "view", own.PublicID.String(), ""))
	if view.Code != http.StatusForbidden || strings.Contains(view.Body.String(), "author edit") {
		t.Fatal("revoked author can inspect note details")
	}
}

func TestDocumentNotesCmdHTTPConcurrentReplacement(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	cmd := h.actions.Browse.DocumentNoteCmd.Endpoint()
	assertDocumentNoteResponse(t, request(cmd,
		documentNoteForm(doc, "create", "", "predecessor")), http.StatusOK)
	var original *enttenant.DocumentNote
	if err := run(func(ctx *ctxx.SpaceContext) error {
		original = ctx.TTx.DocumentNote.Query().OnlyX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, 2)
	for i := range 2 {
		go func() {
			<-start
			responses <- request(cmd, documentNoteForm(doc, "replace",
				original.PublicID.String(), fmt.Sprintf("contender %d", i)))
		}()
	}
	close(start)
	successes := 0
	for range 2 {
		rr := <-responses
		if rr.Code == http.StatusOK {
			successes++
			assertDocumentNoteResponse(t, rr, http.StatusOK)
		} else {
			// SQLite can reject a stale snapshot with BUSY rather than a domain conflict.
			if rr.Code != http.StatusConflict && rr.Code != http.StatusInternalServerError {
				t.Fatalf("unexpected contender status %d: %s", rr.Code, rr.Body.String())
			}
			assertDocumentNoteResponse(t, rr, rr.Code)
		}
	}
	if successes != 1 {
		t.Fatalf("successful competing replacements = %d, want 1", successes)
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		old := ctx.TTx.DocumentNote.GetX(ctx, original.ID)
		if old.ReplacedByID == 0 || old.Body != original.Body ||
			ctx.TTx.DocumentNote.Query().CountX(ctx) != 2 {
			t.Fatalf("non-atomic competing replacement: %v", old)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentNotesCmdHTTPConstraintRollsBackReplacement(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	for _, operation := range []string{"create", "edit", "replace"} {
		form := documentNoteForm(doc, operation, "legacy", "valid body")
		form.Set("Title", " \t\n")
		assertDocumentNoteResponse(t, request(h.actions.Browse.DocumentNoteCmd.Endpoint(), form),
			http.StatusBadRequest)
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		if ctx.TTx.DocumentNote.Query().CountX(ctx) != 0 ||
			ctx.TTx.File.GetX(ctx, doc.ID).Notes != "legacy note" {
			t.Fatal("invalid title materialized legacy or changed its text")
		}
		// A real SQLite constraint fails the final link, after legacy materialization,
		// clearing File.Notes and inserting the successor. No production hooks are needed.
		_, err := ctx.TTx.ExecContext(ctx, `CREATE TRIGGER reject_note_replacement
			BEFORE UPDATE OF replaced_by_id ON document_notes
			WHEN NEW.replaced_by_id IS NOT NULL
			BEGIN SELECT RAISE(ABORT, 'forced note constraint'); END`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	rr := request(h.actions.Browse.DocumentNoteCmd.Endpoint(),
		documentNoteForm(doc, "replace", "legacy", "must roll back"))
	assertDocumentNoteResponse(t, rr, http.StatusInternalServerError)
	if err := run(func(ctx *ctxx.SpaceContext) error {
		if ctx.TTx.DocumentNote.Query().CountX(ctx) != 0 ||
			ctx.TTx.File.GetX(ctx, doc.ID).Notes != "legacy note" {
			t.Fatal("constraint failure left partial replacement or cleared legacy text")
		}
		_, err := ctx.TTx.ExecContext(ctx, "DROP TRIGGER reject_note_replacement")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	assertDocumentNoteResponse(t, request(h.actions.Browse.DocumentNoteCmd.Endpoint(),
		documentNoteForm(doc, "replace", "legacy", "retry succeeds")), http.StatusOK)
}

func TestDocumentNotesHTTPTitlelessFallback(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	var note *enttenant.DocumentNote
	if err := run(func(ctx *ctxx.SpaceContext) error {
		note = ctx.TTx.DocumentNote.Create().SetSpaceID(ctx.Space.ID).SetFileID(doc.ID).
			SetBody("Existing titleless body").SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	partial := request(h.actions.Browse.DocumentNotesPartial.Endpoint(),
		url.Values{"FileID": {doc.PublicID.String()}})
	if partial.Code != http.StatusOK || strings.Count(partial.Body.String(), `>Note</div>`) != 2 {
		t.Fatalf("legacy and titleless rows need a display-only title: %s", partial.Body.String())
	}
	for _, id := range []string{"legacy", note.PublicID.String()} {
		details := request(h.actions.Browse.DocumentNoteDialog.Endpoint(),
			documentNoteForm(doc, "view", id, ""))
		for _, want := range []string{`id="documentNoteDetailsDialog-headline"`, "Author: Unknown", "Created: Unknown"} {
			if details.Code != http.StatusOK || !strings.Contains(details.Body.String(), want) {
				t.Fatalf("titleless details missing %q: %s", want, details.Body.String())
			}
		}
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		loaded := ctx.TTx.DocumentNote.GetX(ctx, note.ID)
		if loaded.Title != "" || loaded.Body != note.Body || loaded.AuthorID != 0 ||
			loaded.AuthoredAt != nil || ctx.TTx.File.GetX(ctx, doc.ID).Notes != "legacy note" {
			t.Fatal("display fallback invented stored title or attribution")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentNotesHTTPReplacementTitle(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	var predecessor, successor *enttenant.DocumentNote
	if err := run(func(ctx *ctxx.SpaceContext) error {
		successor = ctx.TTx.DocumentNote.Create().SetSpaceID(ctx.Space.ID).SetFileID(doc.ID).
			SetBody("Successor body").SaveX(ctx)
		predecessor = ctx.TTx.DocumentNote.Create().SetSpaceID(ctx.Space.ID).SetFileID(doc.ID).
			SetTitle("Old title").SetBody("Historical body only in details").
			SetReplacedByID(successor.ID).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"", "<strong>Replacement & title</strong>"} {
		if err := run(func(ctx *ctxx.SpaceContext) error {
			_, err := ctx.TTx.DocumentNote.UpdateOneID(successor.ID).SetTitle(title).Save(ctx)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		want := "Replaced by: Note"
		if title != "" {
			want = "Replaced by: &lt;strong&gt;Replacement &amp; title&lt;/strong&gt;"
		}
		partial := request(h.actions.Browse.DocumentNotesPartial.Endpoint(), url.Values{
			"FileID":      {doc.PublicID.String()},
			"ShowHistory": {"true"},
		})
		body := partial.Body.String()
		if partial.Code != http.StatusOK || !strings.Contains(body, "<em>"+want+"</em>") ||
			strings.Contains(body, predecessor.Body) || strings.Contains(body, "<strong>") {
			t.Fatalf("invalid replacement summary: %s", body)
		}
		details := request(h.actions.Browse.DocumentNoteDialog.Endpoint(),
			documentNoteForm(doc, "view", predecessor.PublicID.String(), ""))
		for _, text := range []string{predecessor.Body, want,
			`hx-target="#documentNoteDetailsContent"`, `hx-select="#documentNoteDetailsContent"`} {
			if details.Code != http.StatusOK || !strings.Contains(details.Body.String(), text) {
				t.Fatalf("replacement details missing %q: %s", text, details.Body.String())
			}
		}
	}
}

func TestDocumentNotesHTTPEmptyState(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	for _, view := range []string{"Browse", "Inbox", "Trash"} {
		t.Run(view, func(t *testing.T) {
			if err := run(func(ctx *ctxx.SpaceContext) error {
				update := ctx.TTx.File.UpdateOneID(doc.ID).ClearNotes().SetIsInInbox(view == "Inbox")
				if view == "Trash" {
					update.SetDeletedAt(time.Now())
				}
				_, err := update.Save(ctx)
				return err
			}); err != nil {
				t.Fatal(err)
			}
			for _, history := range []string{"false", "true"} {
				rr := request(h.actions.Browse.DocumentNotesPartial.Endpoint(), url.Values{
					"FileID":      {doc.PublicID.String()},
					"ShowHistory": {history},
				})
				body := rr.Body.String()
				if rr.Code != http.StatusOK ||
					!strings.Contains(body, `<h3 class="title-lg">No notes available.</h3>`) {
					t.Fatalf("missing empty state: %d %s", rr.Code, body)
				}
				wantActions := 2 // Toolbar and empty-state creation actions.
				if view == "Trash" {
					wantActions = 0
				}
				if strings.Count(body, h.actions.Browse.DocumentNoteDialog.Endpoint()) != wantActions {
					t.Fatalf("incorrect empty-state creation controls in %s: %s", view, body)
				}
				if !strings.Contains(body, `aria-pressed="`+history+`"`) {
					t.Fatal("empty state lost history control or state")
				}
			}
		})
	}
}

func TestDocumentNotesHTTPAllViewTabsAndFolderModes(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	for _, folderMode := range []bool{false, true} {
		for _, view := range []string{"Browse", "Inbox", "Trash"} {
			t.Run(fmt.Sprintf("%s/folders=%t", view, folderMode), func(t *testing.T) {
				var rootID string
				if err := run(func(ctx *ctxx.SpaceContext) error {
					ctx.TTx.Space.UpdateOneID(ctx.Space.ID).SetIsFolderMode(folderMode).SaveX(ctx)
					update := ctx.TTx.File.UpdateOneID(doc.ID).SetIsInInbox(view == "Inbox")
					if view == "Trash" {
						update.SetDeletedAt(time.Now())
					} else {
						update.ClearDeletedAt()
					}
					update.SaveX(schema.SkipSoftDelete(ctx))
					rootID = ctx.SpaceRootDir().PublicID.String()
					return nil
				}); err != nil {
					t.Fatal(err)
				}
				endpoint := h.actions.Browse.FileTabsPartial.Endpoint()
				if view == "Inbox" {
					endpoint = h.actions.Inbox.FileTabsPartial.Endpoint()
				} else if view == "Trash" {
					endpoint = h.actions.Trash.FileTabsPartial.Endpoint()
				}
				rr := request(endpoint, url.Values{"FileID": {doc.PublicID.String()},
					"CurrentDirID": {rootID}, "ActiveTab": {"notes"}})
				if rr.Code != http.StatusOK {
					t.Fatalf("tabs: %d %s", rr.Code, rr.Body.String())
				}
				for _, want := range []string{`id="documentNotes"`, "legacy note",
					"Show deleted and replaced notes", "Notes"} {
					if !strings.Contains(rr.Body.String(), want) {
						t.Fatalf("missing %q: %s", want, rr.Body.String())
					}
				}
				if (countDocumentNoteMutationControls(rr.Body.String(), h.actions.Browse.DocumentNoteDialog.Endpoint()) > 0) !=
					(view != "Trash") {
					t.Fatal("incorrect read-only controls")
				}
				if rr.Header().Get("HX-Push-Url") == "" {
					t.Fatal("tab selection did not preserve navigation state")
				}
			})
		}
	}
}

func TestDocumentNotesHTTPUnavailableAuthorRetainsAttribution(t *testing.T) {
	h, run, request, doc := newDocumentNotesRequestTest(t)
	var authorID, noteID int64
	if err := run(func(ctx *ctxx.SpaceContext) error {
		author := ctx.TTx.User.Create().SetAccountID(ctx.User.AccountID + 10000).
			SetEmail(entx.NewCIText("removed-author@example.com")).SetFirstName("Removed author").
			SetRole(tenantrole.User).SaveX(ctx)
		authorID = author.ID
		note := ctx.TTx.DocumentNote.Create().SetSpaceID(ctx.Space.ID).SetFileID(doc.ID).
			SetBody("Retained text after author removal").SetAuthorID(authorID).
			SetEditorID(authorID).SetAuthoredAt(time.Now()).SetEditedAt(time.Now()).SaveX(ctx)
		noteID = note.ID
		ctx.TTx.User.UpdateOneID(authorID).SetDeletedAt(time.Now()).SaveX(ctx)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rr := request(h.actions.Browse.DocumentNotesPartial.Endpoint(),
		url.Values{"FileID": {doc.PublicID.String()}, "ShowHistory": {"true"}})
	if rr.Code != http.StatusOK {
		t.Fatalf("read removed author's note: %d %s", rr.Code, rr.Body.String())
	}
	for _, text := range []string{
		"Retained text after author removal", "Author: Unknown", `data-tooltip="Created:`,
	} {
		if !strings.Contains(rr.Body.String(), text) {
			t.Fatalf("missing %q: %s", text, rr.Body.String())
		}
	}
	if strings.Contains(rr.Body.String(), "Removed author") ||
		strings.Contains(rr.Body.String(), "removed-author@example.com") {
		t.Fatal("unavailable author was loaded through the deleted-user filter")
	}
	if err := run(func(ctx *ctxx.SpaceContext) error {
		note := ctx.TTx.DocumentNote.GetX(ctx, noteID)
		if note.AuthorID != authorID || note.EditorID != authorID ||
			ctx.TTx.File.GetX(ctx, doc.ID).Notes != "legacy note" {
			t.Fatal("reading an unavailable author changed attribution or materialized legacy text")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
