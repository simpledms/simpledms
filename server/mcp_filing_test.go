package server

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	filingmodel "github.com/simpledms/simpledms/model/tenant/filing"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

func TestMCPFilingJourney(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "filing")
	readOnly := newMCPFixture(t, h, "filing-read-only")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")
	readClient := readOnly.connect(t, server.URL+"/mcp")

	var tagID, documentTypeID string
	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		if err := tc.TTx.Space.Update().SetIsFolderMode(true).Exec(tc); err != nil {
			return err
		}
		spaceCtx := ctxx.NewSpaceContext(tc, tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(fixture.spaceID)),
		).OnlyX(tc))
		filex := spaceCtx.TTx.File.Query().Where(
			file.PublicID(entx.NewCIText(fixture.fileID)),
		).OnlyX(spaceCtx)
		tagx, err := taggingmodel.NewTagService().Create(
			spaceCtx, spaceCtx.Space.ID, 0, "Filed", tagtype.Simple,
		)
		if err != nil {
			return err
		}
		if _, err := taggingmodel.NewTagService().AssignToFile(
			spaceCtx, filex.ID, tagx.ID, spaceCtx.Space.ID,
		); err != nil {
			return err
		}
		documentTypex, err := documenttypemodel.Create(spaceCtx, spaceCtx.Space.ID, "Filed type")
		if err != nil {
			return err
		}
		if _, err := documenttypemodel.NewAssignmentService().Set(
			spaceCtx, filex.ID, documentTypex.Data.ID,
		); err != nil {
			return err
		}
		tagID = tagx.PublicID.String()
		documentTypeID = documentTypex.Data.PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	root := callMCP(t, client, "list_directory", map[string]any{})
	if len(root["children"].([]any)) != 0 {
		t.Fatalf("Inbox files leaked into root directory: %v", root)
	}

	directory := callMCP(t, client, "create_directory", map[string]any{
		"parent_directory_id": fixture.rootID,
		"name":                "Filed",
	})
	directoryID := directory["directory_id"].(string)
	if directoryID == "" || directory["parent_id"] != fixture.rootID {
		t.Fatalf("directory public IDs were not returned: %v", directory)
	}
	listed := callMCP(t, client, "list_directory", map[string]any{})
	if !containsMCPChild(t, listed["children"], directoryID, "Filed") {
		t.Fatalf("created directory was not listable: %v", listed)
	}
	assertMCPToolError(t, client, "file_inbox_document", map[string]any{
		"file_id":                  fixture.fileID,
		"destination_directory_id": fixture.rootID,
		"filename":                 "../invalid.txt",
		"new_directory_name":       "must-not-exist",
	})
	if !containsMCPChild(
		t,
		callMCP(t, client, "list_inbox", map[string]any{})["files"],
		fixture.fileID,
		"",
	) {
		t.Fatal("failed filing removed the document from Inbox")
	}
	if containsMCPChild(
		t,
		callMCP(t, client, "list_directory", map[string]any{})["children"],
		"",
		"must-not-exist",
	) {
		t.Fatal("failed filing left its requested child directory behind")
	}

	filed := callMCP(t, client, "file_inbox_document", map[string]any{
		"file_id":                  fixture.fileID,
		"destination_directory_id": directoryID,
		"filename":                 "filed-renamed.txt",
	})
	if filed["file_id"] != fixture.fileID || filed["name"] != "filed-renamed.txt" ||
		filed["parent_id"] != directoryID || filed["is_in_inbox"] != false {
		t.Fatalf("filing result: %v", filed)
	}
	inbox := callMCP(t, client, "list_inbox", map[string]any{})
	if containsMCPChild(t, inbox["files"], fixture.fileID, "") {
		t.Fatalf("filed document remained in Inbox: %v", inbox)
	}

	otherID := fixtureFileID(t, fixture)
	rootFiled := callMCP(t, client, "file_inbox_document", map[string]any{
		// The file is already parented at root; filing must still complete it.
		"file_id":                  otherID,
		"destination_directory_id": fixture.rootID,
	})
	if rootFiled["file_id"] != otherID || rootFiled["parent_id"] != fixture.rootID ||
		rootFiled["is_in_inbox"] != false {
		t.Fatalf("same-parent root filing: %v", rootFiled)
	}

	filtered := callMCP(t, client, "search_files", map[string]any{
		"tag_ids":          []string{tagID},
		"document_type_id": documentTypeID,
	})
	if len(filtered["files"].([]any)) != 1 ||
		filtered["files"].([]any)[0].(map[string]any)["file_id"] != fixture.fileID {
		t.Fatalf("classification filters: %v", filtered)
	}

	page := callMCP(t, client, "search_files", map[string]any{"limit": 1, "sort": "name"})
	if len(page["files"].([]any)) != 1 || page["has_more"] != true || page["next_offset"] != float64(1) {
		t.Fatalf("search pagination: %v", page)
	}
	next := callMCP(t, client, "search_files", map[string]any{
		"limit": 1, "offset": 1, "sort": "name",
	})
	if len(next["files"].([]any)) != 1 || next["has_more"] != false {
		t.Fatalf("search second page: %v", next)
	}

	assertMCPToolError(t, readClient, "create_directory", map[string]any{
		"parent_directory_id": readOnly.rootID,
		"name":                "not-created",
	})
	assertMCPToolError(t, readClient, "file_inbox_document", map[string]any{
		"file_id":                  readOnly.fileID,
		"destination_directory_id": readOnly.rootID,
	})
	assertMCPToolError(t, client, "file_inbox_document", map[string]any{
		"file_id":                  readOnly.fileID,
		"destination_directory_id": fixture.rootID,
	})
}

func TestMCPFilingRollsBackCreatedChildDirectoryOnCommitFailure(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "filing-rollback")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")

	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		if err := tc.TTx.Space.Update().SetIsFolderMode(true).Exec(tc); err != nil {
			return err
		}
		spaceCtx := ctxx.NewSpaceContext(tc, tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(fixture.spaceID)),
		).OnlyX(tc))
		tc.TTx.OnCommit(func(enttenant.Committer) enttenant.Committer {
			return enttenant.CommitFunc(func(context.Context, *enttenant.Tx) error {
				return errors.New("injected filing commit failure")
			})
		})
		_, err := filingmodel.NewFilingService(h.infra.FileSystem()).FileInboxDocument(
			spaceCtx, fixture.fileID, fixture.rootID, "filed.txt", "created-then-rolled-back",
		)
		if err != nil {
			return err
		}
		return tc.TTx.Commit()
	}); err == nil {
		t.Fatal("filing unexpectedly succeeded despite commit failure")
	}

	if containsMCPChild(t, callMCP(t, client, "list_directory", map[string]any{
		"directory_id": fixture.rootID,
	})["children"], "", "created-then-rolled-back") {
		t.Fatal("failed filing left a child directory behind")
	}
	if !containsMCPChild(t, callMCP(t, client, "list_inbox", map[string]any{})["files"], fixture.fileID, "") {
		t.Fatal("failed filing removed the document from Inbox")
	}
}

func TestMCPFilingConcurrentCompletionOnlySucceedsOnce(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "filing-concurrent")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	first := fixture.connect(t, server.URL+"/mcp")
	second := fixture.connect(t, server.URL+"/mcp")

	type outcome struct {
		result *mcp.CallToolResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	var group sync.WaitGroup
	for _, client := range []*mcp.ClientSession{first, second} {
		group.Add(1)
		go func(client *mcp.ClientSession) {
			defer group.Done()
			result, err := client.CallTool(context.Background(), &mcp.CallToolParams{
				Name: "mark_inbox_file_done", Arguments: map[string]any{"file_id": fixture.fileID},
			})
			outcomes <- outcome{result: result, err: err}
		}(client)
	}
	group.Wait()
	close(outcomes)

	successes := 0
	failures := 0
	for result := range outcomes {
		if result.err == nil && result.result != nil && !result.result.IsError {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("concurrent completion outcomes: successes=%d failures=%d", successes, failures)
	}
	if containsMCPChild(t, callMCP(t, first, "list_inbox", map[string]any{})["files"], fixture.fileID, "") {
		t.Fatal("successful concurrent completion left the file in Inbox")
	}
}

func TestMCPFilingConflictAndRepeatedFilingPreserveInboxState(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "filing-conflict")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")
	otherID := fixtureFileID(t, fixture)

	callMCP(t, client, "file_inbox_document", map[string]any{
		"file_id": fixture.fileID, "destination_directory_id": fixture.rootID, "filename": "same-name.txt",
	})
	assertMCPToolError(t, client, "file_inbox_document", map[string]any{
		"file_id": otherID, "destination_directory_id": fixture.rootID, "filename": "same-name.txt",
	})
	if !containsMCPChild(t, callMCP(t, client, "list_inbox", map[string]any{})["files"], otherID, "") {
		t.Fatal("filename conflict removed the second file from Inbox")
	}
	assertMCPToolError(t, client, "file_inbox_document", map[string]any{
		"file_id": fixture.fileID, "destination_directory_id": fixture.rootID,
	})
}

func TestMCPFilingRejectsInvalidDirectoryNamesWithoutCreatingRows(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "filing-invalid-directory")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")

	for _, name := range []string{".", "folder/../name"} {
		assertMCPToolError(t, client, "create_directory", map[string]any{
			"parent_directory_id": fixture.rootID,
			"name":                name,
		})
	}

	children := callMCP(t, client, "list_directory", map[string]any{})["children"]
	if len(children.([]any)) != 0 {
		t.Fatalf("invalid directory names created rows: %v", children)
	}
	for _, name := range []string{".", "folder/../name"} {
		if containsMCPChild(t, children, "", name) {
			t.Fatalf("invalid directory %q created a row: %v", name, children)
		}
	}
}

func TestMCPFilingRejectsDotFilenameAndPreservesInbox(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "filing-invalid-filename")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")

	assertMCPToolError(t, client, "file_inbox_document", map[string]any{
		"file_id":                  fixture.fileID,
		"destination_directory_id": fixture.rootID,
		"filename":                 ".",
	})
	if !containsMCPChild(t, callMCP(t, client, "list_inbox", map[string]any{})["files"], fixture.fileID, "") {
		t.Fatal("invalid filename removed the document from Inbox")
	}
}

func TestMCPMarkInboxFileDoneInNonFolderSpace(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "done-non-folder")
	foreign := newMCPFixtureWithWrites(t, h, "done-foreign")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")

	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		return tc.TTx.Space.Update().SetIsFolderMode(false).Exec(tc)
	}); err != nil {
		t.Fatal(err)
	}
	done := callMCP(t, client, "mark_inbox_file_done", map[string]any{"file_id": fixture.fileID})
	if done["file_id"] != fixture.fileID || done["parent_id"] != fixture.rootID || done["is_in_inbox"] != false {
		t.Fatalf("non-folder completion: %v", done)
	}
	assertMCPToolError(t, client, "mark_inbox_file_done", map[string]any{"file_id": fixture.fileID})
	assertMCPToolError(t, client, "mark_inbox_file_done", map[string]any{"file_id": foreign.fileID})
}

func fixtureFileID(t *testing.T, fixture *mcpFixture) string {
	t.Helper()
	var fileID string
	if err := withTenantContext(t, fixture.h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sc := ctxx.NewSpaceContext(tc, tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(fixture.spaceID)),
		).OnlyX(tc))
		files, err := sc.TTx.File.Query().Where(
			file.IsInInbox(true), file.PublicIDNEQ(entx.NewCIText(fixture.fileID)),
		).All(sc)
		if err != nil {
			return err
		}
		if len(files) != 1 {
			t.Fatalf("fixture file count: %d", len(files))
		}
		fileID = files[0].PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return fileID
}

func containsMCPChild(t *testing.T, values any, id, name string) bool {
	t.Helper()
	for _, value := range values.([]any) {
		child := value.(map[string]any)
		if (id == "" || child["file_id"] == id || child["directory_id"] == id) &&
			(name == "" || child["name"] == name) {
			return true
		}
	}
	return false
}
