package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/model/main/common/spacerole"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
)

func TestMCPNotesHistory(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "notes")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")

	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(space.Name("MCP notes")).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, spacex)
		filex := tc.TTx.File.Query().Where(file.Name("mcp-notes-0.txt")).OnlyX(sc)
		tc.TTx.File.UpdateOne(filex).SetNotes("legacy UTF-8 🙂").SaveX(sc)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	legacy := callMCP(t, client, "list_document_notes", map[string]any{"file_id": fixture.fileID})
	if legacy["file_id"] != fixture.fileID || len(legacy["notes"].([]any)) != 0 ||
		legacy["legacy_note"] == nil {
		t.Fatalf("legacy list = %v", legacy)
	}
	legacyNote := legacy["legacy_note"].(map[string]any)
	if legacyNote["note_id"] != "legacy" || legacyNote["is_legacy"] != true ||
		legacyNote["author"] != nil || legacyNote["authored_at"] != nil {
		t.Fatalf("legacy projection = %v", legacyNote)
	}
	legacyRead := callMCP(t, client, "get_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": "legacy",
	})
	if legacyRead["body"] != "legacy UTF-8 🙂" || legacyRead["is_legacy"] != true {
		t.Fatalf("legacy body = %v", legacyRead)
	}

	created := callMCP(t, client, "create_document_note", map[string]any{
		"file_id": fixture.fileID, "title": "First", "body": "first body",
	})
	noteID := created["note_id"].(string)
	author := created["author"].(map[string]any)
	if created["file_id"] != fixture.fileID || author["user_id"] == "" ||
		created["can_change"] != true {
		t.Fatalf("created note = %v", created)
	}
	edited := callMCP(t, client, "edit_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": noteID, "title": "Edited", "body": "edited body",
	})
	if edited["note_id"] != noteID || edited["author"].(map[string]any)["user_id"] != author["user_id"] ||
		edited["body"] != "edited body" || edited["editor"].(map[string]any)["user_id"] != author["user_id"] {
		t.Fatalf("edited note = %v", edited)
	}

	replaced := callMCP(t, client, "replace_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": noteID, "title": "Replacement", "body": "replacement",
	})
	successorID := replaced["note_id"].(string)
	if successorID == noteID || replaced["author"].(map[string]any)["user_id"] != author["user_id"] {
		t.Fatalf("replacement = %v", replaced)
	}
	old := callMCP(t, client, "get_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": noteID,
	})
	if old["replaced_by_note_id"] != successorID || old["body"] != "edited body" ||
		old["can_change"] != false {
		t.Fatalf("replacement history = %v", old)
	}
	deleted := callMCP(t, client, "delete_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": successorID,
	})
	if deleted["deleted"] != true {
		t.Fatalf("delete = %v", deleted)
	}
	deletedData := callMCP(t, client, "get_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": successorID,
	})
	if deletedData["deleted_at"] == nil || deletedData["can_change"] != false {
		t.Fatalf("deleted history = %v", deletedData)
	}
	assertMCPToolErrorCode(t, client, "edit_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": successorID, "title": "stale", "body": "stale",
	}, "conflict", "Historical notes")

	for _, body := range []string{"page one", "page two"} {
		callMCP(t, client, "create_document_note", map[string]any{
			"file_id": fixture.fileID, "title": body, "body": body,
		})
	}
	page := callMCP(t, client, "list_document_notes", map[string]any{
		"file_id": fixture.fileID, "limit": 1,
	})
	if len(page["notes"].([]any)) != 1 || page["has_more"] != true || page["next_offset"] == nil {
		t.Fatalf("current pagination = %v", page)
	}
	for _, item := range page["notes"].([]any) {
		if item.(map[string]any)["note_id"] == noteID || item.(map[string]any)["note_id"] == successorID {
			t.Fatalf("history leaked into current page: %v", page)
		}
	}
	history := callMCP(t, client, "list_document_notes", map[string]any{
		"file_id": fixture.fileID, "show_history": true, "limit": 50,
	})
	if len(history["notes"].([]any)) < 4 || history["legacy_note"] == nil {
		t.Fatalf("history list = %v", history)
	}

	utf8Body := strings.Repeat("🙂", 4) + " tail"
	utf8Note := callMCP(t, client, "create_document_note", map[string]any{
		"file_id": fixture.fileID, "title": "UTF-8", "body": utf8Body,
	})
	window := callMCP(t, client, "get_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": utf8Note["note_id"], "offset": 1, "length": 2,
	})
	if window["body"] != "🙂🙂" || window["body_offset"] != float64(1) || window["has_more_body"] != true {
		t.Fatalf("UTF-8 window = %v", window)
	}

	readOnly := *fixture
	data := url.Values{
		"Label":       {"MCP notes read-only"},
		"Destination": {fixture.tenant.PublicID.String() + ":" + fixture.spaceID},
	}
	response := fixture.browser(h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(), data)
	if response.Code != http.StatusOK {
		t.Fatalf("create read-only credential: %d %s", response.Code, response.Body.String())
	}
	readOnly.token = regexp.MustCompile(`sdmcp_[a-z0-9]+\.[A-Za-z0-9_-]{43}`).FindString(response.Body.String())
	readOnlyClient := readOnly.connect(t, server.URL+"/mcp")
	if got := callMCP(t, readOnlyClient, "get_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": utf8Note["note_id"],
	})["note_id"]; got != utf8Note["note_id"] {
		t.Fatalf("read-only note = %v", got)
	}
	assertMCPToolErrorCode(t, readOnlyClient, "create_document_note", map[string]any{
		"file_id": fixture.fileID, "title": "denied", "body": "denied",
	}, "forbidden", "cannot write")
	for _, name := range []string{"edit_document_note", "replace_document_note", "delete_document_note"} {
		args := map[string]any{"file_id": fixture.fileID, "note_id": utf8Note["note_id"]}
		if name != "delete_document_note" {
			args["title"], args["body"] = "denied", "denied"
		}
		assertMCPToolErrorCode(t, readOnlyClient, name, args, "forbidden", "cannot write")
	}

	var otherNoteID, foreignFileID, foreignNoteID string
	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(space.Name("MCP notes")).OnlyX(tc)
		ctx := ctxx.NewSpaceContext(tc, spacex)
		otherAuthor := tc.TTx.User.Create().SetAccountID(fixture.account.ID + 10000).
			SetEmail("mcp-other-author@example.com").SetFirstName("Other").
			SetRole(tenantrole.User).SaveX(ctx)
		note := tc.TTx.DocumentNote.Create().SetSpaceID(spacex.ID).
			SetFileID(tc.TTx.File.Query().Where(file.Name("mcp-notes-0.txt"), file.SpaceID(spacex.ID)).OnlyX(ctx).ID).
			SetTitle("Other").SetBody("other").SetAuthorID(otherAuthor.ID).
			SetAuthoredAt(time.Now()).SaveX(ctx)
		otherNoteID = note.PublicID.String()
		assignment := tc.TTx.SpaceUserAssignment.Query().Where(
			spaceuserassignment.SpaceID(spacex.ID), spaceuserassignment.UserID(tc.User.ID),
		).OnlyX(ctx)
		tc.TTx.User.UpdateOneID(tc.User.ID).SetRole(tenantrole.User).SaveX(ctx)
		tc.TTx.SpaceUserAssignment.UpdateOne(assignment).SetRole(spacerole.User).SaveX(ctx)
		createSpaceViaCmd(t, h.actions, tc, "MCP notes foreign")
		foreignSpace := tc.TTx.Space.Query().Where(space.Name("MCP notes foreign")).OnlyX(ctx)
		foreignCtx := ctxx.NewSpaceContext(tc, foreignSpace)
		foreign := createDocumentForNotesTest(foreignCtx, "foreign.pdf", "")
		foreignFileID = foreign.PublicID.String()
		foreignNote, err := filemodel.NewDocumentNotes().Create(foreignCtx, foreignFileID, "Foreign", "foreign")
		if err != nil {
			return err
		}
		foreignNoteID = foreignNote.PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	assertMCPToolErrorCode(t, client, "edit_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": otherNoteID, "title": "no", "body": "no",
	}, "forbidden", "cannot change")
	assertMCPToolErrorCode(t, client, "edit_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": foreignNoteID, "title": "no", "body": "no",
	}, "not_found", "not found")
	assertMCPToolErrorCode(t, client, "get_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": foreignNoteID,
	}, "not_found", "not found")
	assertMCPToolErrorCode(t, client, "get_document_note", map[string]any{
		"file_id": foreignFileID, "note_id": foreignNoteID,
	}, "not_found", "not found")
	assertMCPToolErrorCode(t, client, "list_document_notes", map[string]any{
		"file_id": foreignFileID,
	}, "not_found", "not found")
	assertMCPToolErrorCode(t, client, "create_document_note", map[string]any{
		"file_id": foreignFileID, "title": "denied", "body": "denied",
	}, "not_found", "not found")

	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(space.Name("MCP notes")).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, spacex)
		assignment := tc.TTx.SpaceUserAssignment.Query().Where(
			spaceuserassignment.SpaceID(spacex.ID), spaceuserassignment.UserID(tc.User.ID),
		).OnlyX(sc)
		tc.TTx.SpaceUserAssignment.UpdateOne(assignment).SetRole(spacerole.Owner).SaveX(sc)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ownerEdit := callMCP(t, client, "edit_document_note", map[string]any{
		"file_id": fixture.fileID, "note_id": otherNoteID, "title": "Owner", "body": "owner edit",
	})
	if ownerEdit["note_id"] != otherNoteID || ownerEdit["body"] != "owner edit" {
		t.Fatalf("owner edit = %v", ownerEdit)
	}
}
