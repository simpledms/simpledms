package server

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"entgo.io/ent/dialect/sql"

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
	"github.com/simpledms/simpledms/ui/uix/route"
)

func TestMCPClassificationUsesDesiredStateAcrossTransports(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixtureWithWrites(t, h, "classify")
	readOnly := newMCPFixture(t, h, "classify-read-only")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")
	readClient := readOnly.connect(t, server.URL+"/mcp")

	var simpleTagID, groupTagID, superTagID, childTagID string
	var textID, numberID, moneyID, dateID, checkboxID string
	var documentTypeID string
	var documentTypeInternalID int64
	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(fixture.spaceID)),
		).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, spacex)
		tagService := taggingmodel.NewTagService()
		simple, err := tagService.Create(sc, spacex.ID, 0, "Reviewed", tagtype.Simple)
		if err != nil {
			return err
		}
		group, err := tagService.Create(sc, spacex.ID, 0, "Department", tagtype.Group)
		if err != nil {
			return err
		}
		super, err := tagService.Create(sc, spacex.ID, 0, "Invoice", tagtype.Super)
		if err != nil {
			return err
		}
		child, err := tagService.Create(sc, spacex.ID, group.ID, "Accounts", tagtype.Simple)
		if err != nil {
			return err
		}
		if _, _, err := tagService.AssignSubTag(sc, super.ID, child.ID); err != nil {
			return err
		}
		simpleTagID = simple.PublicID.String()
		groupTagID = group.PublicID.String()
		superTagID = super.PublicID.String()
		childTagID = child.PublicID.String()

		propertyService := propertymodel.NewPropertyService()
		text, err := propertyService.Create(sc, spacex.ID, "Reference", fieldtype.Text, "")
		if err != nil {
			return err
		}
		number, err := propertyService.Create(sc, spacex.ID, "Pages", fieldtype.Number, "")
		if err != nil {
			return err
		}
		money, err := propertyService.Create(sc, spacex.ID, "Amount", fieldtype.Money, "CHF")
		if err != nil {
			return err
		}
		date, err := propertyService.Create(sc, spacex.ID, "Invoice date", fieldtype.Date, "")
		if err != nil {
			return err
		}
		checkbox, err := propertyService.Create(sc, spacex.ID, "Paid", fieldtype.Checkbox, "")
		if err != nil {
			return err
		}
		textID = text.PublicID.String()
		numberID = number.PublicID.String()
		moneyID = money.PublicID.String()
		dateID = date.PublicID.String()
		checkboxID = checkbox.PublicID.String()

		documentTypex, err := documenttypemodel.Create(sc, spacex.ID, "Invoice")
		if err != nil {
			return err
		}
		if _, err := documentTypex.CreateTagAttribute(sc, "Department", group.ID, false); err != nil {
			return err
		}
		if _, err := documentTypex.CreatePropertyAttribute(sc, money.ID, false); err != nil {
			return err
		}
		documentTypeID = documentTypex.Data.PublicID.String()
		documentTypeInternalID = documentTypex.Data.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	tags := callMCP(t, client, "list_tags", map[string]any{"group_id": groupTagID})
	if len(tags["tags"].([]any)) != 1 {
		t.Fatalf("group Tag projection: %v", tags)
	}
	properties := callMCP(t, client, "list_properties", map[string]any{})
	if len(properties["properties"].([]any)) != 5 {
		t.Fatalf("property projection: %v", properties)
	}
	documentType := callMCP(t, client, "get_document_type", map[string]any{
		"document_type_id": documentTypeID,
	})
	if len(documentType["attributes"].([]any)) != 2 {
		t.Fatalf("document-type attributes: %v", documentType)
	}

	for range 2 {
		callMCP(t, client, "set_document_type", map[string]any{
			"file_id": fixture.fileID, "document_type_id": documentTypeID,
		})
		callMCP(t, client, "assign_tag", map[string]any{
			"file_id": fixture.fileID, "tag_id": simpleTagID,
		})
	}
	callMCP(t, client, "assign_tag", map[string]any{
		"file_id": fixture.fileID, "tag_id": superTagID,
	})
	callMCP(t, client, "assign_tag", map[string]any{
		"file_id": fixture.fileID, "tag_id": childTagID,
	})
	unassigned := callMCP(t, client, "unassign_tag", map[string]any{
		"file_id": fixture.fileID, "tag_id": childTagID,
	})
	if unassigned["directly_assigned"] != false || unassigned["resolved"] != true {
		t.Fatalf("resolved Tag state was lost: %v", unassigned)
	}
	assertMCPToolError(t, client, "assign_tag", map[string]any{
		"file_id": fixture.fileID, "tag_id": groupTagID,
	})

	values := []map[string]any{
		{"file_id": fixture.fileID, "property_id": textID, "text_value": ""},
		{"file_id": fixture.fileID, "property_id": numberID, "number_value": 0},
		{"file_id": fixture.fileID, "property_id": moneyID, "money_minor_units": 12345},
		{"file_id": fixture.fileID, "property_id": dateID, "date_value": "2026-09-29"},
		{"file_id": fixture.fileID, "property_id": checkboxID, "checkbox_value": false},
	}
	for _, value := range values {
		callMCP(t, client, "set_file_property", value)
	}
	assertMCPToolError(t, client, "set_file_property", map[string]any{
		"file_id": fixture.fileID, "property_id": numberID, "text_value": "wrong",
	})
	assertMCPToolError(t, client, "set_file_property", map[string]any{
		"file_id": fixture.fileID, "property_id": numberID,
		"number_value": 1, "text_value": "multiple",
	})
	for _, invalid := range []map[string]any{
		{"file_id": fixture.fileID, "property_id": numberID},
		{"file_id": fixture.fileID, "property_id": numberID, "number_value": nil},
		{"file_id": fixture.fileID, "property_id": checkboxID},
		{"file_id": fixture.fileID, "property_id": checkboxID, "checkbox_value": nil},
		{
			"file_id": fixture.fileID, "property_id": numberID,
			"number_value": int64(9007199254740992),
		},
		{
			"file_id": fixture.fileID, "property_id": dateID,
			"date_value": "29-09-2026",
		},
	} {
		assertMCPToolError(t, client, "set_file_property", invalid)
	}

	fileData := callMCP(t, client, "get_file", map[string]any{"file_id": fixture.fileID})
	if fileData["document_type"].(map[string]any)["document_type_id"] != documentTypeID {
		t.Fatalf("document type not projected: %v", fileData)
	}
	if len(fileData["direct_tags"].([]any)) != 2 || len(fileData["resolved_tags"].([]any)) != 3 {
		t.Fatalf("direct/resolved Tags not projected: %v", fileData)
	}
	assignedProperties := fileData["properties"].([]any)
	if propertyByID(t, assignedProperties, textID)["text_value"] != "" ||
		propertyByID(t, assignedProperties, numberID)["number_value"] != float64(0) ||
		propertyByID(t, assignedProperties, moneyID)["money_minor_units"] != float64(12345) ||
		propertyByID(t, assignedProperties, dateID)["date_value"] != "2026-09-29" ||
		propertyByID(t, assignedProperties, checkboxID)["checkbox_value"] != false {
		t.Fatalf("typed values not projected: %v", assignedProperties)
	}

	for range 2 {
		removed := callMCP(t, client, "remove_file_property", map[string]any{
			"file_id": fixture.fileID, "property_id": dateID,
		})
		if removed["assigned"] != false {
			t.Fatalf("property removal was not desired-state: %v", removed)
		}
	}

	toggled := fixture.browserAt(
		route.Inbox(fixture.tenant.PublicID.String(), fixture.spaceID, fixture.fileID),
		h.actions.Browse.SelectDocumentTypePartial.Endpoint(), url.Values{
			"FileID": {fixture.fileID}, "DocumentTypeID": {strconv.FormatInt(documentTypeInternalID, 10)},
		})
	if toggled.Code != 200 {
		t.Fatalf("browser document-type toggle: %d %s", toggled.Code, toggled.Body.String())
	}
	fileData = callMCP(t, client, "get_file", map[string]any{"file_id": fixture.fileID})
	if _, exists := fileData["document_type"]; exists {
		t.Fatalf("browser toggle was not reflected through MCP: %v", fileData)
	}

	assertMCPToolError(t, readClient, "assign_tag", map[string]any{
		"file_id": readOnly.fileID, "tag_id": simpleTagID,
	})
	var result sql.Result
	if err := fixture.db.ReadWriteConn.Driver().Exec(
		context.Background(),
		"UPDATE `properties` SET `public_id` = NULL WHERE `public_id` = ?",
		[]any{entx.NewCIText(textID)},
		&result,
	); err != nil {
		t.Fatal(err)
	}
	assertMCPToolError(t, client, "list_properties", map[string]any{})
}

func propertyByID(t *testing.T, properties []any, propertyID string) map[string]any {
	t.Helper()
	for _, value := range properties {
		propertyx := value.(map[string]any)
		if propertyx["property_id"] == propertyID {
			return propertyx
		}
	}
	t.Fatalf("property %s not found in %v", propertyID, properties)
	return nil
}
