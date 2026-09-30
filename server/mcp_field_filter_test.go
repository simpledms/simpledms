package server

import (
	"net/http/httptest"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
)

func TestMCPFieldFilters(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "field-filters")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")

	propertyIDs := make(map[string]string)
	for _, property := range []struct {
		name string
		kind string
	}{
		{"Text", "Text"}, {"Number", "Number"}, {"Money", "Money"},
		{"Date", "Date"}, {"Checkbox", "Checkbox"},
	} {
		created := callMCP(t, client, "create_property", map[string]any{
			"name": property.name, "type": property.kind,
		})
		propertyIDs[property.name] = stringField(t, created, "property_id")
	}

	secondFileID := fixtureFileID(t, fixture)
	for _, fileID := range []string{fixture.fileID, secondFileID} {
		callMCP(t, client, "file_inbox_document", map[string]any{
			"file_id":                  fileID,
			"destination_directory_id": fixture.rootID,
		})
	}
	// Keep an Inbox document in the same Space so search_files must not return it.
	inboxFileID := createMCPFieldFilterInboxFile(t, fixture)

	setProperty := func(fileID, propertyID string, value map[string]any) {
		args := map[string]any{"file_id": fileID, "property_id": propertyID}
		for key, value := range value {
			args[key] = value
		}
		callMCP(t, client, "set_file_property", args)
	}
	setProperty(fixture.fileID, propertyIDs["Text"], map[string]any{"text_value": "Alpha"})
	setProperty(fixture.fileID, propertyIDs["Number"], map[string]any{"number_value": 10})
	setProperty(fixture.fileID, propertyIDs["Money"], map[string]any{"money_minor_units": 1250})
	setProperty(fixture.fileID, propertyIDs["Date"], map[string]any{"date_value": "2026-01-15"})
	setProperty(fixture.fileID, propertyIDs["Checkbox"], map[string]any{"checkbox_value": true})
	setProperty(secondFileID, propertyIDs["Text"], map[string]any{"text_value": ""})
	setProperty(secondFileID, propertyIDs["Number"], map[string]any{"number_value": 20})
	setProperty(secondFileID, propertyIDs["Money"], map[string]any{"money_minor_units": 2000})
	setProperty(secondFileID, propertyIDs["Date"], map[string]any{"date_value": "2026-02-15"})
	// The second filed document has another property, but no Checkbox assignment.

	search := func(filter map[string]any) []any {
		return callMCP(t, client, "search_files", map[string]any{
			"property_filters": []map[string]any{filter},
		})["files"].([]any)
	}
	if !hasMCPID(search(map[string]any{
		"property_id": propertyIDs["Text"], "operator": "equals", "text_value": "aLpHa",
	}), "file_id", fixture.fileID) {
		t.Fatal("text equals was not case-insensitive")
	}
	if !hasMCPID(search(map[string]any{
		"property_id": propertyIDs["Text"], "operator": "equals", "text_value": "",
	}), "file_id", secondFileID) {
		t.Fatal("empty text value was not preserved")
	}
	for _, operator := range []string{"contains", "starts_with"} {
		if !hasMCPID(search(map[string]any{
			"property_id": propertyIDs["Text"], "operator": operator, "text_value": "alp",
		}), "file_id", fixture.fileID) {
			t.Fatalf("text operator %q did not match", operator)
		}
	}

	for _, test := range []struct {
		name   string
		filter map[string]any
		wanted string
	}{
		{"number scalar", map[string]any{"property_id": propertyIDs["Number"], "operator": "greater_than_or_equal", "number_value": 20}, secondFileID},
		{"number greater than", map[string]any{"property_id": propertyIDs["Number"], "operator": "greater_than", "number_value": 10}, secondFileID},
		{"money scalar", map[string]any{"property_id": propertyIDs["Money"], "operator": "equals", "money_minor_units": 1250}, fixture.fileID},
		{"date scalar", map[string]any{"property_id": propertyIDs["Date"], "operator": "less_than", "date_value": "2026-02-15"}, fixture.fileID},
		{"date less than or equal", map[string]any{"property_id": propertyIDs["Date"], "operator": "less_than_or_equal", "date_value": "2026-01-15"}, fixture.fileID},
		{"number inclusive range", map[string]any{"property_id": propertyIDs["Number"], "operator": "between", "number_value": 10, "end_number_value": 20}, fixture.fileID},
		{"money inclusive range", map[string]any{"property_id": propertyIDs["Money"], "operator": "between", "money_minor_units": 1250, "end_money_minor_units": 2000}, secondFileID},
		{"date inclusive range", map[string]any{"property_id": propertyIDs["Date"], "operator": "between", "date_value": "2026-01-15", "end_date_value": "2026-02-15"}, fixture.fileID},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !hasMCPID(search(test.filter), "file_id", test.wanted) {
				t.Fatalf("filter did not include %s", test.wanted)
			}
		})
	}
	if !hasMCPID(search(map[string]any{
		"property_id": propertyIDs["Checkbox"], "operator": "is_checked", "checkbox_value": true,
	}), "file_id", fixture.fileID) || hasMCPID(search(map[string]any{
		"property_id": propertyIDs["Checkbox"], "operator": "is_checked", "checkbox_value": true,
	}), "file_id", secondFileID) {
		t.Fatal("checkbox true semantics were incorrect")
	}
	if !hasMCPID(search(map[string]any{
		"property_id": propertyIDs["Checkbox"], "operator": "equals", "checkbox_value": false,
	}), "file_id", secondFileID) || hasMCPID(search(map[string]any{
		"property_id": propertyIDs["Checkbox"], "operator": "equals", "checkbox_value": false,
	}), "file_id", fixture.fileID) {
		t.Fatal("checkbox false semantics were incorrect")
	}
	if hasMCPID(search(map[string]any{
		"property_id": propertyIDs["Checkbox"], "operator": "equals", "checkbox_value": false,
	}), "file_id", inboxFileID) {
		t.Fatal("Inbox document leaked into search_files")
	}

	tag := callMCP(t, client, "create_tag", map[string]any{"name": "Filtered", "type": "Simple"})
	tagID := stringField(t, tag, "tag_id")
	docType := callMCP(t, client, "create_document_type", map[string]any{"name": "Filtered type"})
	docTypeID := stringField(t, docType, "document_type_id")
	callMCP(t, client, "assign_tag", map[string]any{"file_id": fixture.fileID, "tag_id": tagID})
	callMCP(t, client, "set_document_type", map[string]any{"file_id": fixture.fileID, "document_type_id": docTypeID})
	combined := callMCP(t, client, "search_files", map[string]any{
		"query": "mcp", "tag_ids": []string{tagID}, "document_type_id": docTypeID,
		"property_filters": []map[string]any{{"property_id": propertyIDs["Number"], "operator": "equals", "number_value": 10}},
		"limit":            1,
	})
	if len(combined["files"].([]any)) != 1 || combined["has_more"] != false ||
		!hasMCPID(combined["files"], "file_id", fixture.fileID) {
		t.Fatalf("filters did not compose: %v", combined)
	}
	page := callMCP(t, client, "search_files", map[string]any{
		"limit": 1, "sort": "name", "property_filters": []map[string]any{{
			"property_id": propertyIDs["Number"], "operator": "greater_than_or_equal", "number_value": 10,
		}},
	})
	if len(page["files"].([]any)) != 1 || page["has_more"] != true || page["next_offset"] != float64(1) {
		t.Fatalf("filtered search pagination: %v", page)
	}

	for _, filter := range []map[string]any{
		{"property_id": propertyIDs["Text"], "operator": "equals", "number_value": 1},
		{"property_id": propertyIDs["Number"], "operator": "equals"},
		{"property_id": propertyIDs["Date"], "operator": "equals", "date_value": "15-01-2026"},
		{"property_id": propertyIDs["Number"], "operator": "between", "number_value": 20, "end_number_value": 10},
		{"property_id": propertyIDs["Number"], "operator": "equals", "number_value": 1, "end_number_value": 2},
	} {
		assertMCPToolError(t, client, "search_files", map[string]any{"property_filters": []map[string]any{filter}})
	}
	tooMany := make([]map[string]any, 33)
	for index := range tooMany {
		tooMany[index] = map[string]any{"property_id": propertyIDs["Number"], "operator": "equals", "number_value": 10}
	}
	assertMCPToolError(t, client, "search_files", map[string]any{"property_filters": tooMany})

	var foreignPropertyID string
	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, h.actions, tc, "MCP field filter foreign")
		foreignSpace := tc.TTx.Space.Query().Where(space.Name("MCP field filter foreign")).OnlyX(tc)
		property, err := propertymodel.NewPropertyService().Create(
			ctxx.NewSpaceContext(tc, foreignSpace), foreignSpace.ID, "Foreign", fieldtype.Number, "",
		)
		if err != nil {
			return err
		}
		foreignPropertyID = property.PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	assertMCPToolError(t, client, "search_files", map[string]any{"property_filters": []map[string]any{{
		"property_id": foreignPropertyID, "operator": "equals", "number_value": 10,
	}}})
}

func createMCPFieldFilterInboxFile(t *testing.T, fixture *mcpFixture) string {
	t.Helper()
	var fileID string
	if err := withTenantContext(t, fixture.h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sc := ctxx.NewSpaceContext(tc, tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(fixture.spaceID)),
		).OnlyX(tc))
		file := createRegularFileForTest(sc, sc.SpaceRootDir().ID, "field-filter-inbox.txt").Data.Update().
			SetIsInInbox(true).SetOcrContent("ä🙂abcdef").SaveX(sc)
		fileID = file.PublicID.String()
		return seedStoredFilesForBenchmarkRows(sc, []*enttenant.File{file})
	}); err != nil {
		t.Fatal(err)
	}
	return fileID
}
