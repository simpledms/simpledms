package server

import (
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

func TestMCPBearerCredentialCannotCrossSpaces(t *testing.T) {
	h := newActionTestHarness(t)
	a := newMCPFixtureWithWrites(t, h, "space-isolation")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	aClient := a.connect(t, server.URL+"/mcp")

	var bID, bRootID, bFileID, aFiledID, bFiledID string
	var aTagID, bTagID, bGroupID, aPropertyID, bPropertyID, aTypeID, bTypeID string
	if err := withTenantContext(t, h, a.account, a.tenant, a.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, h.actions, tc, "MCP isolation B")
		b := tc.TTx.Space.Query().Where(space.Name("MCP isolation B")).OnlyX(tc)
		bID = b.PublicID.String()
		for _, spacex := range []*enttenant.Space{tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(a.spaceID)),
		).OnlyX(tc), b} {
			if err := spacex.Update().SetIsFolderMode(true).Exec(tc); err != nil {
				return err
			}
		}
		asc := ctxx.NewSpaceContext(tc, tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(a.spaceID)),
		).OnlyX(tc))
		bsc := ctxx.NewSpaceContext(tc, b)
		bRootID = bsc.SpaceRootDir().PublicID.String()
		at, err := taggingmodel.NewTagService().Create(asc, asc.Space.ID, 0, "A tag", tagtype.Simple)
		if err != nil {
			return err
		}
		bt, err := taggingmodel.NewTagService().Create(bsc, bsc.Space.ID, 0, "B tag", tagtype.Simple)
		if err != nil {
			return err
		}
		bg, err := taggingmodel.NewTagService().Create(bsc, bsc.Space.ID, 0, "B group", tagtype.Group)
		if err != nil {
			return err
		}
		ap, err := propertymodel.NewPropertyService().Create(asc, asc.Space.ID, "A field", fieldtype.Text, "")
		if err != nil {
			return err
		}
		bp, err := propertymodel.NewPropertyService().Create(bsc, bsc.Space.ID, "B field", fieldtype.Text, "")
		if err != nil {
			return err
		}
		ay, err := documenttypemodel.Create(asc, asc.Space.ID, "A type")
		if err != nil {
			return err
		}
		by, err := documenttypemodel.Create(bsc, bsc.Space.ID, "B type")
		if err != nil {
			return err
		}
		aTagID, bTagID, bGroupID = at.PublicID.String(), bt.PublicID.String(), bg.PublicID.String()
		aPropertyID, bPropertyID = ap.PublicID.String(), bp.PublicID.String()
		aTypeID, bTypeID = ay.Data.PublicID.String(), by.Data.PublicID.String()
		file := bsc.TTx.File.Create().SetName("b-inbox.txt").SetIsDirectory(false).
			SetIndexedAt(time.Now()).SetModifiedAt(time.Now()).SetParentID(bsc.SpaceRootDir().ID).
			SetSpaceID(bsc.Space.ID).SetIsInInbox(true).SetSource(filesource.MCP).
			SaveX(bsc)
		file = file.Update().SetOcrContent("B OCR text").SetOcrSuccessAt(time.Now()).SaveX(bsc)
		bFileID = file.PublicID.String()
		if err := seedStoredFilesForBenchmarkRows(bsc, []*enttenant.File{file}); err != nil {
			return err
		}
		aFiled := createRegularFileForTest(asc, asc.SpaceRootDir().ID, "a-filed.txt").Data
		bFiled := createRegularFileForTest(bsc, bsc.SpaceRootDir().ID, "b-filed.txt").Data
		aFiledID, bFiledID = aFiled.PublicID.String(), bFiled.PublicID.String()
		if err := seedStoredFilesForBenchmarkRows(asc, []*enttenant.File{aFiled}); err != nil {
			return err
		}
		return seedStoredFilesForBenchmarkRows(bsc, []*enttenant.File{bFiled})
	}); err != nil {
		t.Fatal(err)
	}

	// Creating this second credential proves that the account really is a member of B;
	// the test below nevertheless uses only the A credential for every rejected call.
	data := url.Values{
		"Label":       {"MCP isolation B"},
		"Destination": {a.tenant.PublicID.String() + ":" + bID},
		"AllowWrites": {"on"},
	}
	rr := a.browser(h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(), data)
	if rr.Code != 200 {
		t.Fatalf("create B credential: %d %s", rr.Code, rr.Body.String())
	}
	b := *a
	b.spaceID, b.rootID = bID, bRootID
	b.token = regexp.MustCompile(`sdmcp_[a-z0-9]+\.[A-Za-z0-9_-]{43}`).FindString(rr.Body.String())
	if b.token == "" {
		t.Fatalf("missing B token: %s", rr.Body.String())
	}
	bClient := b.connect(t, server.URL+"/mcp")

	if got := callMCP(t, bClient, "get_space", map[string]any{}); got["space_id"] != bID {
		t.Fatalf("B credential did not access B: %v", got)
	}
	bFile := callMCP(t, bClient, "get_file", map[string]any{"file_id": bFileID})
	if bFile["source"] != filesource.MCP.String() {
		t.Fatalf("B source was not projected: %v", bFile)
	}
	if text := callMCP(t, bClient, "read_file_text", map[string]any{"file_id": bFileID}); text["text"] != "B OCR text" {
		t.Fatalf("B OCR was not readable: %v", text)
	}
	callMCP(t, bClient, "assign_tag", map[string]any{"file_id": bFileID, "tag_id": bTagID})
	callMCP(t, bClient, "set_file_property", map[string]any{
		"file_id": bFileID, "property_id": bPropertyID, "text_value": "B value",
	})
	callMCP(t, bClient, "set_document_type", map[string]any{"file_id": bFileID, "document_type_id": bTypeID})
	bDir := callMCP(t, bClient, "create_directory", map[string]any{
		"parent_directory_id": bRootID, "name": "B subdir",
	})["directory_id"].(string)
	if len(callMCP(t, bClient, "list_inbox", map[string]any{})["files"].([]any)) != 1 ||
		!containsMCPChild(t, callMCP(t, bClient, "list_directory", map[string]any{})["children"], bDir, "B subdir") {
		t.Fatal("B credential did not see its own document and directory")
	}
	aBefore := callMCP(t, aClient, "get_file", map[string]any{"file_id": a.fileID})

	if got := callMCP(t, aClient, "get_space", map[string]any{}); got["space_id"] != a.spaceID {
		t.Fatalf("A credential changed space: %v", got)
	}
	for _, call := range []struct {
		name string
		args map[string]any
	}{
		{"get_file", map[string]any{"file_id": bFileID}},
		{"read_file_text", map[string]any{"file_id": bFileID}},
		{"list_directory", map[string]any{"directory_id": bRootID}},
		{"create_directory", map[string]any{"parent_directory_id": bRootID, "name": "must-not-exist"}},
		{"mark_inbox_file_done", map[string]any{"file_id": bFileID}},
		{"file_inbox_document", map[string]any{"file_id": bFileID, "destination_directory_id": a.rootID}},
		{"file_inbox_document", map[string]any{"file_id": a.fileID, "destination_directory_id": bDir}},
		{"list_tags", map[string]any{"group_id": bGroupID}},
		{"get_document_type", map[string]any{"document_type_id": bTypeID}},
		{"clear_document_type", map[string]any{"file_id": bFileID}},
		{"search_files", map[string]any{"tag_ids": []string{bTagID}}},
		{"search_files", map[string]any{"document_type_id": bTypeID}},
	} {
		assertMCPToolError(t, aClient, call.name, call.args)
	}
	for _, fileID := range []string{a.fileID, bFileID} {
		for _, metadata := range []struct {
			name string
			args map[string]any
		}{
			{"assign_tag", map[string]any{"file_id": fileID, "tag_id": bTagID}},
			{"unassign_tag", map[string]any{"file_id": fileID, "tag_id": bTagID}},
			{"set_file_property", map[string]any{"file_id": fileID, "property_id": bPropertyID, "text_value": "bad"}},
			{"remove_file_property", map[string]any{"file_id": fileID, "property_id": bPropertyID}},
			{"set_document_type", map[string]any{"file_id": fileID, "document_type_id": bTypeID}},
		} {
			assertMCPToolError(t, aClient, metadata.name, metadata.args)
		}
	}
	for _, metadata := range []struct {
		name string
		args map[string]any
	}{
		{"assign_tag", map[string]any{"file_id": bFileID, "tag_id": aTagID}},
		{"unassign_tag", map[string]any{"file_id": bFileID, "tag_id": aTagID}},
		{"set_file_property", map[string]any{"file_id": bFileID, "property_id": aPropertyID, "text_value": "bad"}},
		{"remove_file_property", map[string]any{"file_id": bFileID, "property_id": aPropertyID}},
		{"set_document_type", map[string]any{"file_id": bFileID, "document_type_id": aTypeID}},
	} {
		assertMCPToolError(t, aClient, metadata.name, metadata.args)
	}

	if files := callMCP(t, bClient, "search_files", map[string]any{})["files"]; !hasMCPID(files, "file_id", bFiledID) {
		t.Fatal("B credential did not see its filed document")
	}
	for _, listing := range []struct {
		name     string
		field    string
		expected string
	}{
		{"list_inbox", "files", a.fileID},
		{"search_files", "files", aFiledID},
		{"list_directory", "children", aFiledID},
	} {
		result := callMCP(t, aClient, listing.name, map[string]any{})
		values := result[listing.field]
		if !hasMCPID(values, "file_id", listing.expected) ||
			hasMCPID(values, "file_id", bFileID) || hasMCPID(values, "file_id", bFiledID) ||
			hasMCPID(values, "file_id", bDir) {
			t.Fatalf("%s did not stay scoped to A: %v", listing.name, result)
		}
	}
	for _, listing := range []struct {
		name, field, key, own, foreign string
	}{
		{"list_tags", "tags", "tag_id", aTagID, bTagID},
		{"list_properties", "properties", "property_id", aPropertyID, bPropertyID},
		{"list_document_types", "document_types", "document_type_id", aTypeID, bTypeID},
	} {
		values := callMCP(t, aClient, listing.name, map[string]any{})[listing.field]
		if !hasMCPID(values, listing.key, listing.own) || hasMCPID(values, listing.key, listing.foreign) {
			t.Fatalf("%s leaked another Space's metadata: %v", listing.name, values)
		}
	}
	finalB := callMCP(t, bClient, "get_file", map[string]any{"file_id": bFileID})
	if finalB["is_in_inbox"] != true || finalB["parent_id"] != bRootID || finalB["name"] != "b-inbox.txt" {
		t.Fatalf("rejected A calls changed B lifecycle: %v", finalB)
	}
	if propertyByID(t, finalB["properties"].([]any), bPropertyID)["text_value"] != "B value" {
		t.Fatalf("rejected A calls changed B property: %v", finalB)
	}
	if !hasMCPID(finalB["direct_tags"], "tag_id", bTagID) {
		t.Fatalf("rejected A calls changed B Tags: %v", finalB)
	}
	if finalB["document_type"].(map[string]any)["document_type_id"] != bTypeID {
		t.Fatalf("rejected A calls changed B type: %v", finalB)
	}
	if len(callMCP(t, bClient, "list_directory", map[string]any{})["children"].([]any)) != 2 {
		t.Fatal("rejected A calls changed B directory state")
	}
	if got := callMCP(t, aClient, "get_file", map[string]any{"file_id": a.fileID}); got["file_id"] != a.fileID {
		t.Fatalf("A document state became inaccessible: %v", got)
	} else if !reflect.DeepEqual(got, aBefore) {
		t.Fatalf("rejected cross-space writes changed A document: before=%v after=%v", aBefore, got)
	}
}

func hasMCPID(values any, key, wanted string) bool {
	for _, value := range values.([]any) {
		if value.(map[string]any)[key] == wanted {
			return true
		}
	}
	return false
}
