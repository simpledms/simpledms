package server

import (
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
)

func TestMCPOrganization(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "organization")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")

	assertMCPToolErrorCode(t, client, "rename_file", map[string]any{
		"file_id": fixture.fileID, "new_filename": "inbox-renamed.txt",
	}, "invalid_input", "filed")
	assertMCPToolErrorCode(t, client, "move_file", map[string]any{
		"file_id": fixture.fileID, "destination_directory_id": fixture.rootID,
	}, "invalid_input", "filed")

	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		return tc.TTx.Space.Update().Where(space.PublicID(entx.NewCIText(fixture.spaceID))).
			SetIsFolderMode(true).Exec(tc)
	}); err != nil {
		t.Fatal(err)
	}

	tag := callMCP(t, client, "create_tag", map[string]any{"name": "Organization tag", "type": "Simple"})
	property := callMCP(t, client, "create_property", map[string]any{
		"name": "Organization field", "type": "Text",
	})
	documentType := callMCP(t, client, "create_document_type", map[string]any{"name": "Organization type"})
	tagID := stringField(t, tag, "tag_id")
	propertyID := stringField(t, property, "property_id")
	typeID := stringField(t, documentType, "document_type_id")
	callMCP(t, client, "assign_tag", map[string]any{"file_id": fixture.fileID, "tag_id": tagID})
	callMCP(t, client, "set_file_property", map[string]any{
		"file_id": fixture.fileID, "property_id": propertyID, "text_value": "unchanged",
	})
	callMCP(t, client, "set_document_type", map[string]any{
		"file_id": fixture.fileID, "document_type_id": typeID,
	})
	callMCP(t, client, "file_inbox_document", map[string]any{
		"file_id": fixture.fileID, "destination_directory_id": fixture.rootID, "filename": "original.txt",
	})
	callMCP(t, client, "create_document_note", map[string]any{
		"file_id": fixture.fileID, "title": "Organization", "body": "must survive",
	})

	before := callMCP(t, client, "get_file", map[string]any{"file_id": fixture.fileID})
	notesBefore := callMCP(t, client, "list_document_notes", map[string]any{
		"file_id": fixture.fileID, "show_history": true, "limit": 50,
	})
	if before["source"] == nil || before["version_number"] == nil {
		t.Fatalf("file snapshot omitted source or version: %v", before)
	}

	dirA := stringField(t, callMCP(t, client, "create_directory", map[string]any{
		"parent_directory_id": fixture.rootID, "name": "A",
	}), "directory_id")
	dirB := stringField(t, callMCP(t, client, "create_directory", map[string]any{
		"parent_directory_id": fixture.rootID, "name": "B",
	}), "directory_id")

	renamed := callMCP(t, client, "rename_file", map[string]any{
		"file_id": fixture.fileID, "new_filename": "renamed.txt",
	})
	if renamed["file_id"] != fixture.fileID || renamed["name"] != "renamed.txt" ||
		renamed["parent_id"] != fixture.rootID || renamed["is_in_inbox"] != false {
		t.Fatalf("rename result: %v", renamed)
	}
	assertOrganizationUnchanged(t, client, fixture.fileID, before, notesBefore)
	assertMCPToolErrorCode(t, client, "rename_file", map[string]any{
		"file_id": fixture.fileID, "new_filename": "renamed.txt",
	}, "invalid_input", "same")
	for _, name := range []string{".", "../escape.txt", "nested/name.txt"} {
		assertMCPToolErrorCode(t, client, "rename_file", map[string]any{
			"file_id": fixture.fileID, "new_filename": name,
		}, "invalid_input", "filename")
	}

	assertMCPToolErrorCode(t, client, "move_file", map[string]any{
		"file_id": fixture.fileID, "destination_directory_id": fixture.rootID,
	}, "invalid_input", "current location")
	callMCP(t, client, "move_file", map[string]any{
		"file_id": fixture.fileID, "destination_directory_id": dirA, "filename": "moved.txt",
	})
	if got := callMCP(t, client, "get_file", map[string]any{"file_id": fixture.fileID}); got["file_id"] != fixture.fileID || got["name"] != "moved.txt" || got["parent_id"] != dirA {
		t.Fatalf("move result: %v", got)
	}
	assertOrganizationUnchanged(t, client, fixture.fileID, before, notesBefore)

	otherID := fixtureFileID(t, fixture)
	callMCP(t, client, "file_inbox_document", map[string]any{
		"file_id": otherID, "destination_directory_id": dirA, "filename": "other.txt",
	})
	assertMCPToolErrorCode(t, client, "rename_file", map[string]any{
		"file_id": fixture.fileID, "new_filename": "other.txt",
	}, "conflict", "already exists")
	assertMCPToolErrorCode(t, client, "move_file", map[string]any{
		"file_id": fixture.fileID, "destination_directory_id": dirA, "filename": "other.txt",
	}, "invalid_input", "current location")

	child := stringField(t, callMCP(t, client, "create_directory", map[string]any{
		"parent_directory_id": dirA, "name": "child",
	}), "directory_id")
	moved := callMCP(t, client, "move_file", map[string]any{
		"file_id": fixture.fileID, "destination_directory_id": dirB,
		"filename": "inside.txt", "new_directory_name": "new-child",
	})
	if moved["file_id"] != fixture.fileID || moved["name"] != "inside.txt" || moved["parent_id"] == dirB {
		t.Fatalf("move with child directory: %v", moved)
	}
	if !containsMCPChild(t, callMCP(t, client, "list_directory", map[string]any{
		"directory_id": dirB,
	})["children"], moved["parent_id"].(string), "new-child") {
		t.Fatalf("created child directory was not returned: %v", moved)
	}
	assertOrganizationUnchanged(t, client, fixture.fileID, before, notesBefore)
	assertMCPToolErrorCode(t, client, "move_file", map[string]any{
		"file_id": otherID, "destination_directory_id": dirB,
		"filename": "other.txt", "new_directory_name": "new-child",
	}, "invalid_input", "already exists")
	if got := callMCP(t, client, "get_file", map[string]any{"file_id": otherID}); got["parent_id"] != dirA {
		t.Fatalf("failed child creation partially moved its file: %v", got)
	}

	assertMCPToolErrorCode(t, client, "move_file", map[string]any{
		"file_id": dirA, "destination_directory_id": child,
	}, "invalid_input", "into one of its subfolders")
	renamedDirectory := callMCP(t, client, "rename_file", map[string]any{
		"file_id": dirA, "new_filename": "A-renamed",
	})
	if renamedDirectory["file_id"] != dirA || renamedDirectory["name"] != "A-renamed" {
		t.Fatalf("directory rename result: %v", renamedDirectory)
	}
	if got := callMCP(t, client, "get_file", map[string]any{"file_id": otherID}); got["parent_id"] != dirA {
		t.Fatalf("directory rename changed child parent: %v", got)
	}
	assertMCPToolErrorCode(t, client, "move_file", map[string]any{
		"file_id": dirA, "destination_directory_id": dirB, "filename": "new-child",
	}, "conflict", "already exists")
	movedDirectory := callMCP(t, client, "move_file", map[string]any{
		"file_id": dirA, "destination_directory_id": dirB,
	})
	if movedDirectory["file_id"] != dirA || movedDirectory["parent_id"] != dirB {
		t.Fatalf("directory move result: %v", movedDirectory)
	}
	if got := callMCP(t, client, "get_file", map[string]any{"file_id": otherID}); got["parent_id"] != dirA {
		t.Fatalf("directory move changed child parent: %v", got)
	}
	if !containsMCPChild(t, callMCP(t, client, "list_directory", map[string]any{
		"directory_id": dirB,
	})["children"], dirA, "A-renamed") {
		t.Fatal("directory move did not preserve the renamed directory")
	}
}

func TestMCPOrganizationAuthorization(t *testing.T) {
	h := newActionTestHarness(t)
	a := newMCPFixtureWithWrites(t, h, "organization-auth-a")
	b := newMCPFixtureWithWrites(t, h, "organization-auth-b")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	aClient := a.connect(t, server.URL+"/mcp")
	bClient := b.connect(t, server.URL+"/mcp")
	callMCP(t, aClient, "file_inbox_document", map[string]any{
		"file_id": a.fileID, "destination_directory_id": a.rootID,
	})

	var sameAccountSpaceID, sameAccountRootID, sameAccountFileID string
	if err := withTenantContext(t, h, a.account, a.tenant, a.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, h.actions, tc, "MCP organization same account")
		spacex := tc.TTx.Space.Query().Where(space.Name("MCP organization same account")).OnlyX(tc)
		if err := spacex.Update().SetIsFolderMode(true).Exec(tc); err != nil {
			return err
		}
		sameAccountSpaceID = spacex.PublicID.String()
		sc := ctxx.NewSpaceContext(tc, spacex)
		sameAccountRootID = sc.SpaceRootDir().PublicID.String()
		filex := createRegularFileForTest(sc, sc.SpaceRootDir().ID, "same-account.txt").Data
		sameAccountFileID = filex.PublicID.String()
		return seedStoredFilesForBenchmarkRows(sc, []*enttenant.File{filex})
	}); err != nil {
		t.Fatal(err)
	}
	spaceB := *a
	response := a.browser(h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(), url.Values{
		"Label":       {"organization-same-account-b"},
		"Destination": {a.tenant.PublicID.String() + ":" + sameAccountSpaceID},
		"AllowWrites": {"on"},
	})
	spaceB.token = regexp.MustCompile(`sdmcp_[a-z0-9]+\.[A-Za-z0-9_-]{43}`).FindString(response.Body.String())
	spaceBClient := spaceB.connect(t, server.URL+"/mcp")
	if got := callMCP(t, spaceBClient, "get_space", map[string]any{}); got["space_id"] != sameAccountSpaceID {
		t.Fatalf("same-account credential did not access Space B: %v", got)
	}
	rename := callMCP(t, spaceBClient, "rename_file", map[string]any{
		"file_id": sameAccountFileID, "new_filename": "same-account-renamed.txt",
	})
	if rename["file_id"] != sameAccountFileID || rename["name"] != "same-account-renamed.txt" {
		t.Fatalf("same-account Space B organization: %v", rename)
	}
	sameAccountDirID := stringField(t, callMCP(t, spaceBClient, "create_directory", map[string]any{
		"parent_directory_id": sameAccountRootID, "name": "same-account-dir",
	}), "directory_id")
	movedSameAccount := callMCP(t, spaceBClient, "move_file", map[string]any{
		"file_id": sameAccountFileID, "destination_directory_id": sameAccountDirID,
	})
	if movedSameAccount["file_id"] != sameAccountFileID || movedSameAccount["parent_id"] != sameAccountDirID {
		t.Fatalf("same-account Space B move: %v", movedSameAccount)
	}
	assertMCPToolErrorCode(t, aClient, "rename_file", map[string]any{
		"file_id": sameAccountFileID, "new_filename": "must-not-cross-space.txt",
	}, "not_found", "not found")
	assertMCPToolErrorCode(t, aClient, "move_file", map[string]any{
		"file_id": a.fileID, "destination_directory_id": sameAccountDirID,
	}, "not_found", "not found")
	if got := callMCP(t, aClient, "get_file", map[string]any{"file_id": a.fileID}); got["parent_id"] != a.rootID {
		t.Fatalf("foreign destination changed A's parent: %v", got)
	}

	bBefore := callMCP(t, bClient, "get_file", map[string]any{"file_id": b.fileID})
	assertMCPToolErrorCode(t, aClient, "rename_file", map[string]any{
		"file_id": b.fileID, "new_filename": "cross-account.txt",
	}, "not_found", "not found")
	assertMCPToolErrorCode(t, aClient, "move_file", map[string]any{
		"file_id": b.fileID, "destination_directory_id": a.rootID,
	}, "not_found", "not found")
	if got := callMCP(t, bClient, "get_file", map[string]any{"file_id": b.fileID}); !reflect.DeepEqual(got, bBefore) {
		t.Fatalf("cross-account organization changed B: before=%v after=%v", bBefore, got)
	}

	readOnly := *a
	response = a.browser(h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(), url.Values{
		"Label":       {"organization-read-only"},
		"Destination": {a.tenant.PublicID.String() + ":" + a.spaceID},
	})
	readOnly.token = regexp.MustCompile(`sdmcp_[a-z0-9]+\.[A-Za-z0-9_-]{43}`).FindString(response.Body.String())
	readOnlyClient := readOnly.connect(t, server.URL+"/mcp")
	assertMCPToolErrorCode(t, readOnlyClient, "rename_file", map[string]any{
		"file_id": a.fileID, "new_filename": "read-only.txt",
	}, "forbidden", "cannot write")
	assertMCPToolErrorCode(t, readOnlyClient, "move_file", map[string]any{
		"file_id": a.fileID, "destination_directory_id": a.rootID,
	}, "forbidden", "cannot write")
}

func assertOrganizationUnchanged(
	t *testing.T, client *sdk.ClientSession, fileID string, before, notesBefore map[string]any,
) {
	t.Helper()
	// This helper is intentionally kept at the MCP boundary: organization changes
	// must not rewrite classification, source, versions, or document-note history.
	after := callMCP(t, client, "get_file", map[string]any{"file_id": fileID})
	for _, field := range []string{
		"source", "version_number", "document_type", "direct_tags", "resolved_tags", "properties",
	} {
		if !reflect.DeepEqual(after[field], before[field]) {
			t.Fatalf("organization changed %s: before=%v after=%v", field, before[field], after[field])
		}
	}
	notesAfter := callMCP(t, client, "list_document_notes", map[string]any{
		"file_id": fileID, "show_history": true, "limit": 50,
	})
	if !reflect.DeepEqual(notesAfter, notesBefore) {
		t.Fatalf("organization changed note history: before=%v after=%v", notesBefore, notesAfter)
	}
}
