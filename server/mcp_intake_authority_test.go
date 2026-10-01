package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"entgo.io/ent/dialect/sql"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
)

func TestMCPConnectionLegacyRevocationSurvivesTransportRestart(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixture(t, h, "legacy-reconnect")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := connectLegacyMCP(t, f, server.URL+"/mcp")
	if result := client.InitializeResult(); result == nil || result.ProtocolVersion != "2024-11-05" {
		t.Fatalf("legacy protocol was not negotiated: %v", result)
	}
	if _, err := client.ListTools(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if response := f.browser(h.actions.Dashboard.RevokeMCPCredentialCmd.Endpoint(),
		url.Values{"CredentialPublicID": {f.credentialID}}); response.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", response.Code, response.Body.String())
	}
	if _, err := client.ListTools(context.Background(), nil); err == nil {
		t.Fatal("existing legacy client retained access after revocation")
	}
	restartedRouter := NewRouter(h.mainDB, h.tenantDBs, h.infra, true, h.metaPath, h.i18n, nil)
	restartedServer := httptest.NewServer(restartedRouter)
	t.Cleanup(restartedServer.Close)
	if session, err := connectLegacyMCPResult(f, restartedServer.URL+"/mcp"); err == nil {
		_ = session.Close()
		t.Fatal("revoked credential authenticated after a fresh legacy connection")
	}
}

func TestMCPToolErrorsKeepProtocolSchemaBusinessAndInternalDistinct(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "error-classes")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	if _, err := client.CallTool(context.Background(), &sdk.CallToolParams{Name: "not_a_tool"}); err == nil {
		t.Fatal("unknown tool was accepted as a tool result")
	}
	schemaResult, schemaErr := client.CallTool(context.Background(), &sdk.CallToolParams{
		Name: "get_file", Arguments: map[string]any{"file_id": 42},
	})
	if schemaErr == nil && (schemaResult == nil || !schemaResult.IsError) {
		t.Fatal("schema-invalid arguments were accepted")
	}
	if schemaErr == nil {
		if len(schemaResult.Content) == 0 {
			t.Fatal("schema error did not explain invalid arguments")
		}
		text := schemaResult.Content[0].(*sdk.TextContent).Text
		if strings.Contains(text, `"code":"invalid_input"`) ||
			strings.Contains(text, `"code":"internal_error"`) {
			t.Fatalf("SDK schema rejection was confused with a domain failure: %s", text)
		}
	}
	assertMCPToolErrorCode(t, client, "create_tag", map[string]any{"name": "", "type": "Simple"},
		"invalid_input", "Name")
	var dbResult sql.Result
	if err := f.db.ReadWriteConn.Driver().Exec(context.Background(),
		"DROP TABLE properties", []any{}, &dbResult); err != nil {
		t.Fatal(err)
	}
	result, err := client.CallTool(context.Background(), &sdk.CallToolParams{Name: "list_properties"})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("internal failure was not a tool error: %v, %v", err, result)
	}
	if len(result.Content) == 0 {
		t.Fatal("internal failure had no safe error content")
	}
	text := result.Content[0].(*sdk.TextContent).Text
	if !strings.Contains(text, `"code":"internal_error"`) ||
		strings.Contains(text, "public_id") || strings.Contains(text, f.token) {
		t.Fatalf("internal error was not sanitized: %s", text)
	}
}

func TestMCPSpaceAccessLossKeepsTenantMembershipButDeniesOnlySpace(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixture(t, h, "space-access-loss")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	a := f.connect(t, server.URL+"/mcp")
	var bID string
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, h.actions, tc, "MCP space-access-loss B")
		bID = tc.TTx.Space.Query().Where(space.Name("MCP space-access-loss B")).OnlyX(tc).PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	bResponse := f.browser(h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(), mapValues(
		f.tenant.PublicID.String()+":"+bID, "B credential"))
	if bResponse.Code != http.StatusOK {
		t.Fatalf("create B credential: %d %s", bResponse.Code, bResponse.Body.String())
	}
	b := *f
	b.token = tokenFromMCPResponse(t, bResponse.Body.String())
	bClient := b.connect(t, server.URL+"/mcp")
	if got := callMCP(t, bClient, "get_space", map[string]any{}); got["space_id"] != bID {
		t.Fatalf("B credential cannot access B: %v", got)
	}
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(space.Name("MCP space-access-loss")).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, spacex)
		assignment := tc.TTx.SpaceUserAssignment.Query().Where(
			spaceuserassignment.SpaceID(spacex.ID),
			spaceuserassignment.UserID(tc.User.ID),
		).OnlyX(sc)
		tc.TTx.User.UpdateOneID(tc.User.ID).SetRole(tenantrole.User).SaveX(tc)
		return tc.TTx.SpaceUserAssignment.DeleteOneID(assignment.ID).Exec(sc)
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ListTools(context.Background(), nil); err == nil {
		t.Fatal("credential retained access after its only space assignment was removed")
	}
	if _, err := bClient.ListTools(context.Background(), nil); err != nil {
		t.Fatalf("tenant membership was incorrectly removed with space access: %v", err)
	}
}

func TestMCPConcurrentMetadataAssignmentsAreIdempotentAndSearchResolvedTags(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "concurrent-authority")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	first, second := f.connect(t, server.URL+"/mcp"), f.connect(t, server.URL+"/mcp")
	group := callMCP(t, first, "create_tag", map[string]any{"name": "Project", "type": "Super"})
	child := callMCP(t, first, "create_tag", map[string]any{"name": "Review", "type": "Simple"})
	property := callMCP(t, first, "create_property", map[string]any{"name": "Status", "type": "Text"})
	docType := callMCP(t, first, "create_document_type", map[string]any{"name": "Project type"})
	groupID, childID := stringField(t, group, "tag_id"), stringField(t, child, "tag_id")
	propertyID, typeID := stringField(t, property, "property_id"), stringField(t, docType, "document_type_id")
	callMCP(t, first, "assign_sub_tag", map[string]any{"super_tag_id": groupID, "sub_tag_id": childID})
	var wg sync.WaitGroup
	errs := make(chan error, 6)
	for _, client := range []*sdk.ClientSession{first, second} {
		wg.Add(1)
		go func(client *sdk.ClientSession) {
			defer wg.Done()
			for _, call := range []struct {
				name string
				args map[string]any
			}{
				{"assign_tag", map[string]any{"file_id": f.fileID, "tag_id": groupID}},
				{"set_document_type", map[string]any{
					"file_id": f.fileID, "document_type_id": typeID,
				}},
				{"set_file_property", map[string]any{
					"file_id": f.fileID, "property_id": propertyID, "text_value": "ready",
				}},
			} {
				result, err := client.CallTool(context.Background(), &sdk.CallToolParams{
					Name: call.name, Arguments: call.args,
				})
				if err != nil || result == nil || result.IsError {
					errs <- fmt.Errorf("%s failed: %v, result=%v", call.name, err, result)
				}
			}
		}(client)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	data := callMCP(t, first, "get_file", map[string]any{"file_id": f.fileID})
	if len(data["direct_tags"].([]any)) != 1 ||
		data["document_type"].(map[string]any)["document_type_id"] != typeID ||
		propertyByID(t, data["properties"].([]any), propertyID)["text_value"] != "ready" {
		t.Fatalf("concurrent desired state was not retained: %v", data)
	}
	callMCP(t, first, "mark_inbox_file_done", map[string]any{"file_id": f.fileID})
	search := callMCP(t, first, "search_files", map[string]any{
		"tag_ids": []string{groupID, childID}, "query": "mcp",
	})
	if !hasMCPID(search["files"], "file_id", f.fileID) {
		t.Fatalf("resolved super/simple tag intersection did not match: %v", search)
	}
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(space.Name("MCP concurrent-authority")).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, spacex)
		return sc.TTx.File.Update().Where(
			file.PublicID(entx.NewCIText(f.fileID)),
		).SetOcrContent("intakeauthorityocr").Exec(sc)
	}); err != nil {
		t.Fatal(err)
	}
	contentSearch := callMCP(t, first, "search_files", map[string]any{
		"query": "intakeauthorityocr", "sort": "rank", "tag_ids": []string{childID},
		"document_type_id": typeID,
	})
	if !hasMCPID(contentSearch["files"], "file_id", f.fileID) {
		t.Fatalf("OCR content search did not compose with resolved Tag/type filters: %v", contentSearch)
	}
	noMatch := callMCP(t, first, "search_files", map[string]any{"query": "definitelyabsentintakephrase"})
	if len(noMatch["files"].([]any)) != 0 || noMatch["has_more"] != false {
		t.Fatalf("no-match search did not return an empty bounded result: %v", noMatch)
	}
}

func connectLegacyMCP(t *testing.T, f *mcpFixture, endpoint string) *sdk.ClientSession {
	t.Helper()
	client, err := connectLegacyMCPResult(f, endpoint)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func connectLegacyMCPResult(f *mcpFixture, endpoint string) (*sdk.ClientSession, error) {
	client := sdk.NewClient(&sdk.Implementation{Name: "legacy-slice-test", Version: "1"}, nil)
	httpClient := &http.Client{Transport: mcpRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+f.token)
		return http.DefaultTransport.RoundTrip(req)
	})}
	return client.Connect(context.Background(), &sdk.StreamableClientTransport{
		Endpoint: endpoint, HTTPClient: httpClient, DisableStandaloneSSE: true, MaxRetries: -1,
	}, &sdk.ClientSessionOptions{ProtocolVersion: "2024-11-05"})
}

func mapValues(destination, label string) url.Values {
	return url.Values{"Destination": {destination}, "Label": {label}}
}

func tokenFromMCPResponse(t *testing.T, body string) string {
	t.Helper()
	token := regexp.MustCompile(`sdmcp_[a-z0-9]+\.[A-Za-z0-9_-]{43}`).FindString(body)
	if token == "" {
		t.Fatalf("missing credential token: %s", body)
	}
	return token
}
