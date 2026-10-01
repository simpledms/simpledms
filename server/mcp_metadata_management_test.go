package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
)

func TestMCPMetadataManagementCRUD(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "metadata-management")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")

	tag := callMCP(t, client, "create_tag", map[string]any{
		"name": "Review", "type": "Simple",
	})
	tagID := stringField(t, tag, "tag_id")
	assertMCPToolErrorCode(t, client, "create_tag", map[string]any{
		"name": "Review", "type": "Simple",
	}, "invalid_input", "already")
	editedTag := callMCP(t, client, "edit_tag", map[string]any{
		"tag_id": tagID, "name": "Reviewed",
	})
	if stringField(t, editedTag, "tag_id") != tagID || stringField(t, editedTag, "name") != "Reviewed" {
		t.Fatalf("editing Tag changed identity or name: %v", editedTag)
	}
	if !hasMCPItem(t, callMCP(t, client, "list_tags", map[string]any{})["tags"],
		"tag_id", tagID, "name", "Reviewed") {
		t.Fatal("edited Tag was not discoverable")
	}

	property := callMCP(t, client, "create_property", map[string]any{
		"name": "Amount", "type": "Money", "unit": "CHF",
	})
	propertyID := stringField(t, property, "property_id")
	assertMCPToolErrorCode(t, client, "create_property", map[string]any{
		"name": "Amount", "type": "Money", "unit": "CHF",
	}, "invalid_input", "already")
	editedProperty := callMCP(t, client, "edit_property", map[string]any{
		"property_id": propertyID, "name": "Total",
	})
	if stringField(t, editedProperty, "property_id") != propertyID ||
		stringField(t, editedProperty, "name") != "Total" || stringField(t, editedProperty, "unit") != "CHF" ||
		stringField(t, editedProperty, "type") != "Money" {
		t.Fatalf("omitted unit did not preserve property state: %v", editedProperty)
	}
	editedProperty = callMCP(t, client, "edit_property", map[string]any{
		"property_id": propertyID, "name": "Total", "unit": "",
	})
	if stringField(t, editedProperty, "property_id") != propertyID ||
		(editedProperty["unit"] != nil && editedProperty["unit"] != "") {
		t.Fatalf("empty unit did not clear property state: %v", editedProperty)
	}
	if !hasMCPItem(t, callMCP(t, client, "list_properties", map[string]any{})["properties"],
		"property_id", propertyID, "name", "Total") {
		t.Fatal("edited property was not discoverable")
	}

	documentType := callMCP(t, client, "create_document_type", map[string]any{"name": "Invoice"})
	documentTypeID := stringField(t, documentType, "document_type_id")
	renamed := callMCP(t, client, "rename_document_type", map[string]any{
		"document_type_id": documentTypeID, "name": "Incoming invoice",
	})
	if stringField(t, renamed, "document_type_id") != documentTypeID ||
		stringField(t, renamed, "name") != "Incoming invoice" {
		t.Fatalf("renaming document type changed identity or name: %v", renamed)
	}
	if !hasMCPItem(t, callMCP(t, client, "list_document_types", map[string]any{})["document_types"],
		"document_type_id", documentTypeID, "name", "Incoming invoice") {
		t.Fatal("renamed document type was not discoverable")
	}

	callMCP(t, client, "assign_tag", map[string]any{"file_id": fixture.fileID, "tag_id": tagID})
	assertMCPToolError(t, client, "delete_tag", map[string]any{"tag_id": tagID})
	assertMCPItemPresent(t, callMCP(t, client, "list_tags", map[string]any{})["tags"],
		"tag_id", tagID)
	callMCP(t, client, "unassign_tag", map[string]any{"file_id": fixture.fileID, "tag_id": tagID})

	callMCP(t, client, "set_file_property", map[string]any{
		"file_id": fixture.fileID, "property_id": propertyID, "money_minor_units": 42,
	})
	assertMCPToolError(t, client, "delete_property", map[string]any{"property_id": propertyID})
	assertMCPItemPresent(t, callMCP(t, client, "list_properties", map[string]any{})["properties"],
		"property_id", propertyID)
	callMCP(t, client, "remove_file_property", map[string]any{
		"file_id": fixture.fileID, "property_id": propertyID,
	})

	callMCP(t, client, "set_document_type", map[string]any{
		"file_id": fixture.fileID, "document_type_id": documentTypeID,
	})
	assertMCPToolError(t, client, "delete_document_type", map[string]any{
		"document_type_id": documentTypeID,
	})
	assertMCPItemPresent(t, callMCP(t, client, "list_document_types", map[string]any{})["document_types"],
		"document_type_id", documentTypeID)
	callMCP(t, client, "clear_document_type", map[string]any{"file_id": fixture.fileID})
	if got := callMCP(t, client, "get_file", map[string]any{"file_id": fixture.fileID}); got["file_id"] != fixture.fileID {
		t.Fatalf("assigned metadata deletion affected fixture file: %v", got)
	}

	if deleted := callMCP(t, client, "delete_tag", map[string]any{"tag_id": tagID})["deleted"]; deleted != true {
		t.Fatalf("Tag deletion result: %v", deleted)
	}
	if deleted := callMCP(t, client, "delete_property", map[string]any{"property_id": propertyID})["deleted"]; deleted != true {
		t.Fatalf("property deletion result: %v", deleted)
	}
	if deleted := callMCP(t, client, "delete_document_type", map[string]any{
		"document_type_id": documentTypeID,
	})["deleted"]; deleted != true {
		t.Fatalf("document type deletion result: %v", deleted)
	}
}

func TestMCPMetadataManagementRejectsReadonlyAndInvalidTypes(t *testing.T) {
	h := newActionTestHarness(t)
	writeFixture := newMCPFixtureWithWrites(t, h, "metadata-management-errors")
	readFixture := newMCPFixture(t, h, "metadata-management-readonly")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	writeClient := writeFixture.connect(t, server.URL+"/mcp")
	readClient := readFixture.connect(t, server.URL+"/mcp")
	var readOnlyTagID, readOnlyGroupID, readOnlySuperID, readOnlyPropertyID, readOnlyTypeID string
	if err := withTenantContext(t, h, readFixture.account, readFixture.tenant, readFixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(readFixture.spaceID)),
		).OnlyX(tc)
		tag, err := taggingmodel.NewTagService().Create(
			ctxx.NewSpaceContext(tc, spacex), spacex.ID, 0, "Existing", tagtype.Simple,
		)
		if err != nil {
			return err
		}
		readOnlyTagID = tag.PublicID.String()
		group, err := taggingmodel.NewTagService().Create(
			ctxx.NewSpaceContext(tc, spacex), spacex.ID, 0, "Existing group", tagtype.Group,
		)
		if err != nil {
			return err
		}
		readOnlyGroupID = group.PublicID.String()
		super, err := taggingmodel.NewTagService().Create(
			ctxx.NewSpaceContext(tc, spacex), spacex.ID, 0, "Existing super", tagtype.Super,
		)
		if err != nil {
			return err
		}
		readOnlySuperID = super.PublicID.String()
		property, err := propertymodel.NewPropertyService().Create(
			ctxx.NewSpaceContext(tc, spacex), spacex.ID, "Existing field", fieldtype.Text, "",
		)
		if err != nil {
			return err
		}
		readOnlyPropertyID = property.PublicID.String()
		documentType, err := documenttypemodel.Create(
			ctxx.NewSpaceContext(tc, spacex), spacex.ID, "Existing type",
		)
		if err != nil {
			return err
		}
		readOnlyTypeID = documentType.Data.PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, input := range []map[string]any{
		{"name": "Unknown", "type": ""},
		{"name": "Unknown", "type": "Unknown"},
	} {
		assertMCPToolErrorCode(t, writeClient, "create_tag", input, "invalid_input", "type")
		assertMCPToolErrorCode(t, writeClient, "create_property", input, "invalid_input", "type")
	}
	for _, call := range []struct {
		name string
		args map[string]any
	}{
		{"create_tag", map[string]any{"name": "Read only", "type": "Simple"}},
		{"create_property", map[string]any{"name": "Read only", "type": "Text"}},
		{"create_document_type", map[string]any{"name": "Read only"}},
	} {
		assertMCPToolErrorCode(t, readClient, call.name, call.args, "forbidden", "cannot write")
	}
	assertMCPToolErrorCode(t, readClient, "delete_tag", map[string]any{
		"tag_id": readOnlyTagID,
	}, "forbidden", "cannot write")
	for _, call := range []struct {
		name string
		args map[string]any
	}{
		{"edit_tag", map[string]any{"tag_id": readOnlyTagID, "name": "Read only Tag"}},
		{"move_tag_to_group", map[string]any{"tag_id": readOnlyTagID, "group_id": readOnlyGroupID}},
		{"assign_sub_tag", map[string]any{"super_tag_id": readOnlySuperID, "sub_tag_id": readOnlyTagID}},
		{"unassign_sub_tag", map[string]any{"super_tag_id": readOnlySuperID, "sub_tag_id": readOnlyTagID}},
		{"create_and_assign_tag", map[string]any{
			"name": "Read only assigned", "type": "Simple", "file_id": readFixture.fileID,
		}},
		{"edit_property", map[string]any{"property_id": readOnlyPropertyID, "name": "Read only field"}},
		{"delete_property", map[string]any{"property_id": readOnlyPropertyID}},
		{"rename_document_type", map[string]any{
			"document_type_id": readOnlyTypeID, "name": "Read only type",
		}},
		{"delete_document_type", map[string]any{"document_type_id": readOnlyTypeID}},
		{"create_document_type_tag_attribute", map[string]any{
			"document_type_id": readOnlyTypeID, "tag_id": readOnlyGroupID, "name": "Read only attr",
		}},
		{"create_document_type_property_attribute", map[string]any{
			"document_type_id": readOnlyTypeID, "property_id": readOnlyPropertyID,
		}},
		{"edit_document_type_tag_attribute", map[string]any{
			"document_type_id": readOnlyTypeID, "tag_id": readOnlyGroupID, "name": "Read only attr",
			"is_name_giving": false,
		}},
		{"edit_document_type_property_attribute", map[string]any{
			"document_type_id": readOnlyTypeID, "property_id": readOnlyPropertyID,
			"is_name_giving": false,
		}},
		{"delete_document_type_attribute", map[string]any{
			"document_type_id": readOnlyTypeID, "tag_id": readOnlyGroupID,
		}},
	} {
		assertMCPToolErrorCode(t, readClient, call.name, call.args, "forbidden", "cannot write")
	}
	templates := callMCP(t, readClient, "list_document_type_templates", map[string]any{})["templates"].([]any)
	foundInvoice := false
	for _, item := range templates {
		if item.(map[string]any)["key"] == "invoice" {
			foundInvoice = true
			break
		}
	}
	if !foundInvoice {
		t.Fatal("readonly template discovery omitted invoice")
	}
	assertMCPToolErrorCode(t, readClient, "import_document_types", map[string]any{
		"template_keys": []any{"invoice"},
	}, "forbidden", "cannot write")
}

func TestMCPMetadataManagementRelations(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "metadata-relations")
	templateFixture := newMCPFixtureWithWrites(t, h, "metadata-template-import")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")
	templateClient := templateFixture.connect(t, server.URL+"/mcp")

	group := callMCP(t, client, "create_tag", map[string]any{
		"name": "Departments", "type": "Group",
	})
	groupID := stringField(t, group, "tag_id")
	child := callMCP(t, client, "create_tag", map[string]any{
		"name": "Finance", "type": "Simple", "group_id": groupID,
	})
	childID := stringField(t, child, "tag_id")
	if stringField(t, child, "group_id") != groupID {
		t.Fatalf("child was not created in its Group: %v", child)
	}
	if got := callMCP(t, client, "move_tag_to_group", map[string]any{"tag_id": childID}); got["group_id"] != nil {
		t.Fatalf("omitted group did not clear grouping: %v", got)
	}
	callMCP(t, client, "move_tag_to_group", map[string]any{"tag_id": childID, "group_id": groupID})
	assertMCPToolError(t, client, "move_tag_to_group", map[string]any{
		"tag_id": groupID, "group_id": childID,
	})
	assertMCPToolError(t, client, "move_tag_to_group", map[string]any{
		"tag_id": groupID, "group_id": groupID,
	})

	super := callMCP(t, client, "create_tag", map[string]any{
		"name": "Classification", "type": "Super",
	})
	superID := stringField(t, super, "tag_id")
	callMCP(t, client, "assign_sub_tag", map[string]any{
		"super_tag_id": superID, "sub_tag_id": childID,
	})
	assertMCPToolError(t, client, "assign_sub_tag", map[string]any{
		"super_tag_id": superID, "sub_tag_id": superID,
	})
	assertMCPToolError(t, client, "assign_sub_tag", map[string]any{
		"super_tag_id": superID, "sub_tag_id": groupID,
	})
	super = callMCP(t, client, "assign_sub_tag", map[string]any{
		"super_tag_id": superID, "sub_tag_id": childID,
	})
	if len(super["sub_tag_ids"].([]any)) != 1 || super["sub_tag_ids"].([]any)[0] != childID {
		t.Fatalf("composed Tag relationship was not unique: %v", super)
	}
	callMCP(t, client, "unassign_sub_tag", map[string]any{
		"super_tag_id": superID, "sub_tag_id": childID,
	})

	assigned := callMCP(t, client, "create_and_assign_tag", map[string]any{
		"name": "Urgent", "type": "Simple", "file_id": fixture.fileID,
	})
	assignedID := stringField(t, assigned, "tag_id")
	if assigned["file_id"] != fixture.fileID || assigned["directly_assigned"] != true {
		t.Fatalf("create_and_assign_tag response: %v", assigned)
	}
	before := len(callMCP(t, client, "list_tags", map[string]any{})["tags"].([]any))
	assertMCPToolError(t, client, "create_and_assign_tag", map[string]any{
		"name": "Bad group assignment", "type": "Group", "group_id": groupID, "file_id": fixture.fileID,
	})
	if got := len(callMCP(t, client, "list_tags", map[string]any{})["tags"].([]any)); got != before {
		t.Fatalf("invalid atomic create left an orphan Tag: before=%d after=%d", before, got)
	}

	property := callMCP(t, client, "create_property", map[string]any{
		"name": "Reference", "type": "Text",
	})
	propertyID := stringField(t, property, "property_id")
	documentType := callMCP(t, client, "create_document_type", map[string]any{"name": "Classified"})
	documentTypeID := stringField(t, documentType, "document_type_id")
	callMCP(t, client, "create_document_type_tag_attribute", map[string]any{
		"document_type_id": documentTypeID, "tag_id": groupID, "name": "Department", "is_name_giving": true,
	})
	attributeDoc := callMCP(t, client, "create_document_type_property_attribute", map[string]any{
		"document_type_id": documentTypeID, "property_id": propertyID, "is_name_giving": false,
	})
	assertAttribute(t, attributeDoc, "tag_id", groupID, "Department", true)
	assertAttribute(t, attributeDoc, "property_id", propertyID, "", false)
	if _, ok := attributeDoc["attribute_id"]; ok {
		t.Fatalf("attribute projection leaked an internal ID: %v", attributeDoc)
	}
	callMCP(t, client, "edit_document_type_tag_attribute", map[string]any{
		"document_type_id": documentTypeID, "tag_id": groupID, "name": "Division", "is_name_giving": false,
	})
	attributeDoc = callMCP(t, client, "edit_document_type_property_attribute", map[string]any{
		"document_type_id": documentTypeID, "property_id": propertyID, "is_name_giving": true,
	})
	assertAttribute(t, attributeDoc, "tag_id", groupID, "Division", false)
	assertAttribute(t, attributeDoc, "property_id", propertyID, "", true)
	assertMCPToolError(t, client, "create_document_type_tag_attribute", map[string]any{
		"document_type_id": documentTypeID, "tag_id": groupID, "name": "",
	})
	assertMCPToolError(t, client, "create_document_type_tag_attribute", map[string]any{
		"document_type_id": documentTypeID, "tag_id": childID, "name": "Simple tags are invalid here",
	})
	assertMCPToolError(t, client, "delete_document_type_attribute", map[string]any{
		"document_type_id": documentTypeID, "tag_id": groupID, "property_id": propertyID,
	})
	assertMCPToolError(t, client, "delete_document_type_attribute", map[string]any{
		"document_type_id": documentTypeID,
	})

	callMCP(t, client, "assign_tag", map[string]any{"file_id": fixture.fileID, "tag_id": assignedID})
	callMCP(t, client, "set_file_property", map[string]any{
		"file_id": fixture.fileID, "property_id": propertyID, "text_value": "ABC",
	})
	callMCP(t, client, "set_document_type", map[string]any{
		"file_id": fixture.fileID, "document_type_id": documentTypeID,
	})
	attributeDoc = callMCP(t, client, "delete_document_type_attribute", map[string]any{
		"document_type_id": documentTypeID, "tag_id": groupID,
	})
	if len(attributeDoc["attributes"].([]any)) != 1 {
		t.Fatalf("Tag attribute deletion changed unrelated attributes: %v", attributeDoc)
	}
	file := callMCP(t, client, "get_file", map[string]any{"file_id": fixture.fileID})
	if file["file_id"] != fixture.fileID || len(file["properties"].([]any)) != 1 {
		t.Fatalf("attribute deletion changed file assignments: %v", file)
	}
	callMCP(t, client, "delete_document_type_attribute", map[string]any{
		"document_type_id": documentTypeID, "property_id": propertyID,
	})
	callMCP(t, client, "clear_document_type", map[string]any{"file_id": fixture.fileID})
	callMCP(t, client, "remove_file_property", map[string]any{
		"file_id": fixture.fileID, "property_id": propertyID,
	})
	callMCP(t, client, "unassign_tag", map[string]any{"file_id": fixture.fileID, "tag_id": assignedID})

	templates := callMCP(t, templateClient, "list_document_type_templates", map[string]any{})["templates"].([]any)
	invoiceKey := ""
	for _, item := range templates {
		if item.(map[string]any)["key"] == "invoice" {
			invoiceKey = "invoice"
		}
	}
	if invoiceKey == "" {
		t.Fatal("built-in invoice template was not advertised")
	}
	assertMCPToolError(t, templateClient, "import_document_types", map[string]any{
		"template_keys": []any{invoiceKey, "unknown-template"},
	})
	if got := callMCP(t, templateClient, "list_document_types", map[string]any{}); len(got["document_types"].([]any)) != 0 {
		t.Fatalf("invalid template selection partially imported metadata: %v", got)
	}
	imported := callMCP(t, templateClient, "import_document_types", map[string]any{
		"template_keys": []any{invoiceKey},
	})
	if len(imported["document_types"].([]any)) == 0 {
		t.Fatalf("template import returned no document types: %v", imported)
	}
	beforeImport := len(imported["document_types"].([]any))
	assertMCPToolError(t, templateClient, "import_document_types", map[string]any{
		"template_keys": []any{invoiceKey, "unknown-template"},
	})
	afterImport := callMCP(t, templateClient, "list_document_types", map[string]any{})
	if len(afterImport["document_types"].([]any)) != beforeImport {
		t.Fatalf("invalid template import was not all-or-nothing: %v", afterImport)
	}
}

func assertAttribute(
	t *testing.T, documentType map[string]any, idField, id, name string, nameGiving bool,
) {
	t.Helper()
	for _, item := range documentType["attributes"].([]any) {
		attribute := item.(map[string]any)
		if attribute[idField] == id {
			attributeName, _ := attribute["name"].(string)
			if attributeName != name || attribute["is_name_giving"] != nameGiving {
				t.Fatalf("attribute %s=%q: %v", idField, id, attribute)
			}
			return
		}
	}
	t.Fatalf("attribute %s=%q not found: %v", idField, id, documentType)
}

func stringField(t *testing.T, object map[string]any, field string) string {
	t.Helper()
	value, ok := object[field].(string)
	if !ok {
		t.Fatalf("%s missing string field %q: %v", field, field, object)
	}
	return value
}

func hasMCPItem(
	t *testing.T, value any, idField, id string, nameField, name string,
) bool {
	t.Helper()
	for _, item := range value.([]any) {
		object, ok := item.(map[string]any)
		if ok && object[idField] == id && object[nameField] == name {
			return true
		}
	}
	return false
}

func assertMCPItemPresent(t *testing.T, value any, idField, id string) {
	t.Helper()
	for _, item := range value.([]any) {
		if object, ok := item.(map[string]any); ok && object[idField] == id {
			return
		}
	}
	t.Fatalf("item %s=%q not found in %v", idField, id, value)
}

func assertMCPToolErrorCode(
	t *testing.T, client *sdk.ClientSession, name string, arguments map[string]any,
	wantCode, wantMessage string,
) {
	t.Helper()
	result, err := client.CallTool(context.Background(), &sdk.CallToolParams{
		Name: name, Arguments: arguments,
	})
	if err == nil && (result == nil || !result.IsError) {
		t.Fatalf("%s accepted invalid arguments: %+v", name, result)
	}
	if err != nil {
		t.Fatalf("%s returned an MCP protocol error: %v", name, err)
	}
	var payload map[string]string
	for _, content := range result.Content {
		if text, ok := content.(*sdk.TextContent); ok {
			if err := json.Unmarshal([]byte(text.Text), &payload); err == nil {
				break
			}
		}
	}
	if payload == nil {
		t.Fatalf("%s returned an unstructured error result: %+v", name, result.Content)
	}
	if payload["code"] != wantCode || !strings.Contains(payload["message"], wantMessage) {
		t.Fatalf("%s error = %v, want %s containing %q", name, payload, wantCode, wantMessage)
	}
}
