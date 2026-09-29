package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/mcpcredential"
	"github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	credentialmodel "github.com/simpledms/simpledms/model/main/mcpcredential"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestMCPConnectionScopesReadsAndRevocation(t *testing.T) {
	h := newActionTestHarness(t)
	first := newMCPFixture(t, h, "first")
	second := newMCPFixture(t, h, "second")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	a := first.connect(t, server.URL+"/mcp")
	b := second.connect(t, server.URL+"/mcp")
	tools, err := a.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
		wantReadOnly := tool.Name != "upload_file"
		if tool.Annotations == nil || tool.Annotations.ReadOnlyHint != wantReadOnly {
			t.Errorf("wrong read-only annotation: %s", tool.Name)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{
		"get_file", "get_space", "list_inbox", "read_file_text", "upload_file",
	}) {
		t.Fatalf("unexpected tools: %v", names)
	}
	for _, call := range []struct {
		client  *sdk.ClientSession
		fixture *mcpFixture
	}{{a, first}, {b, second}, {a, first}} {
		data := callMCP(t, call.client, "get_space", map[string]any{})
		if data["space_id"] != call.fixture.spaceID || data["read_only"] != true {
			t.Fatalf("wrong scope/mode: %v", data)
		}
	}
	page := callMCP(t, a, "list_inbox", map[string]any{"limit": 1, "sort": "name"})
	if page["has_more"] != true || len(page["files"].([]any)) != 1 {
		t.Fatalf("bad page: %v", page)
	}
	wantURL := server.URL + route.BrowseFile(
		first.tenant.PublicID.String(), first.spaceID, first.rootID, first.fileID,
	)
	listedFile := page["files"].([]any)[0].(map[string]any)
	if listedFile["url"] != wantURL {
		t.Fatalf("wrong Inbox document URL: got %v, want %s", listedFile["url"], wantURL)
	}
	fileData := callMCP(t, a, "get_file", map[string]any{"file_id": first.fileID})
	if fileData["url"] != wantURL {
		t.Fatalf("wrong document URL: got %v, want %s", fileData["url"], wantURL)
	}
	text := callMCP(t, a, "read_file_text", map[string]any{
		"file_id": first.fileID, "offset": 0, "length": 2,
	})
	if text["text"] != "ä🙂" || text["next_offset"] != float64(2) {
		t.Fatalf("incorrect Unicode window: %v", text)
	}
	for _, input := range []struct {
		tool      string
		arguments map[string]any
	}{
		{"get_file", map[string]any{"file_id": second.fileID}},
		{"get_file", map[string]any{"file_id": "unknown"}},
		{"list_inbox", map[string]any{"limit": 0}},
		{"list_inbox", map[string]any{"sources": []string{"invalid"}}},
		{"read_file_text", map[string]any{"file_id": first.fileID, "length": 50001}},
	} {
		result, err := a.CallTool(context.Background(), &sdk.CallToolParams{
			Name: input.tool, Arguments: input.arguments,
		})
		if err == nil && !result.IsError {
			t.Fatalf("accepted invalid call: %v", input)
		}
	}
	listed := first.browser(h.actions.Dashboard.MCPCredentialListPartial.Endpoint(), nil)
	if listed.Code != http.StatusOK || strings.Contains(listed.Body.String(), first.token) {
		t.Fatalf("credential list disclosed secret or failed: %d", listed.Code)
	}
	foreignRevoke := second.browser(h.actions.Dashboard.RevokeMCPCredentialCmd.Endpoint(),
		url.Values{"CredentialID": {first.credentialID}})
	if foreignRevoke.Code != http.StatusNotFound {
		t.Fatalf("foreign revoke status: %d", foreignRevoke.Code)
	}
	revoked := first.browser(h.actions.Dashboard.RevokeMCPCredentialCmd.Endpoint(),
		url.Values{"CredentialID": {first.credentialID}})
	if revoked.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", revoked.Code, revoked.Body.String())
	}
	if _, err := a.ListTools(context.Background(), nil); err == nil {
		t.Fatal("existing client retained access after revocation")
	}
	callMCP(t, b, "get_file", map[string]any{"file_id": second.fileID})
}

func TestMCPConnectionAccessLossAndHTTPBoundary(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixture(t, h, "boundary")
	for _, testCase := range []struct {
		token  string
		origin string
		want   int
	}{
		{"", "", http.StatusUnauthorized},
		{"bad", "", http.StatusUnauthorized},
		{f.token, "https://other.example", http.StatusForbidden},
		{f.token, "null", http.StatusForbidden},
	} {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+testCase.token)
		req.Header.Set("Origin", testCase.origin)
		req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: f.session})
		rr := httptest.NewRecorder()
		h.router.ServeHTTP(rr, req)
		if rr.Code != testCase.want || rr.Header().Get("Location") != "" {
			t.Fatalf("HTTP boundary: %d, want %d", rr.Code, testCase.want)
		}
	}
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.CallTool(ctx, &sdk.CallToolParams{Name: "get_space"}); err == nil {
		t.Fatal("cancelled call succeeded")
	}
	h.mainDB.ReadWriteConn.TenantAccountAssignment.Update().Where(
		tenantaccountassignment.AccountID(f.account.ID),
		tenantaccountassignment.TenantID(f.tenant.ID),
	).SetExpiresAt(time.Now().Add(-time.Minute)).ExecX(context.Background())
	if _, err := client.ListTools(context.Background(), nil); err == nil {
		t.Fatal("expired tenant assignment retained MCP access")
	}
}

func TestMCPConnectionCommitFailureDoesNotDiscloseToken(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixture(t, h, "commit")
	queryCtx := privacy.DecisionContext(context.Background(), privacy.Allow)
	before := h.mainDB.ReadOnlyConn.MCPCredential.Query().CountX(queryCtx)
	wrapped := h.router.wrapTxResponse(func(
		rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
	) error {
		ctx.MainCtx().MainTx.OnCommit(func(entmain.Committer) entmain.Committer {
			return entmain.CommitFunc(func(context.Context, *entmain.Tx) error {
				return errors.New("injected commit failure")
			})
		})
		return h.actions.Dashboard.CreateMCPCredentialCmd.Handler(rw, req, ctx)
	}, false, true)
	values := url.Values{
		"Label":       {"must roll back"},
		"Destination": {f.tenant.PublicID.String() + ":" + f.spaceID},
	}
	req := httptest.NewRequest(http.MethodPost, h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(),
		strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: f.session})
	rr := httptest.NewRecorder()
	wrapped(rr, req)
	if rr.Code != http.StatusInternalServerError || strings.Contains(rr.Body.String(), "sdmcp_") ||
		rr.Header().Get("HX-Trigger") != "" {
		t.Fatalf("commit failure released response: %d %s", rr.Code, rr.Body.String())
	}
	if count := h.mainDB.ReadOnlyConn.MCPCredential.Query().CountX(queryCtx); count != before {
		t.Fatalf("failed credential committed: before=%d after=%d", before, count)
	}
}

func TestMCPConnectionCreateResponseShowsTokenDialog(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixture(t, h, "token-dialog")
	response := f.browser(h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(), url.Values{
		"Label":       {"Visible token"},
		"Destination": {f.tenant.PublicID.String() + ":" + f.spaceID},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("create credential: %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("HX-Reswap") == "none" {
		t.Fatal("HTMX would discard the one-time token dialog")
	}
	if !strings.Contains(response.Body.String(), "MCP credential created") ||
		!strings.Contains(response.Body.String(), "sdmcp_") {
		t.Fatal("successful response did not include the one-time token dialog")
	}
}

func TestMCPConnectionSetupSessionAndReadOnlyExecution(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixture(t, h, "read")
	err := withMainContext(t, h, f.account, func(_ *entmain.Tx, ctx *ctxx.MainContext) error {
		ctx.IsTemporarySession = true
		_, err := credentialmodel.NewCredentialService().Create(ctx,
			f.tenant.PublicID.String(), f.spaceID, "setup", true)
		if err == nil {
			t.Error("setup Session issued credential")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = credentialmodel.NewCredentialService().Execute(context.Background(), h.mainDB,
		h.tenantDBs, h.i18n, false, f.token, func(ctx *ctxx.SpaceContext, _ *entmain.MCPCredential) error {
			return ctx.TTx.File.Update().SetName("must not write").Exec(ctx)
		})
	if err == nil {
		t.Fatal("inspection transaction allowed a write")
	}
	credential := h.mainDB.ReadOnlyConn.MCPCredential.Query().Where(
		mcpcredential.PublicID(entx.NewCIText(f.credentialID)),
	).OnlyX(privacy.DecisionContext(context.Background(), privacy.Allow))
	if strings.Contains(credential.SecretHash, f.token) || len(credential.SecretHash) != 64 {
		t.Fatal("credential did not store a SHA-256 verification hash")
	}
}

func TestMCPConnectionDocumentStatesAndUnavailableTenant(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixture(t, h, "states")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	var rootID string
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sc := ctxx.NewSpaceContext(tc, tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(f.spaceID)),
		).OnlyX(tc))
		rootID = sc.SpaceRootDir().PublicID.String()
		return sc.TTx.File.Update().Where(file.PublicID(entx.NewCIText(f.fileID))).
			ClearOcrSuccessAt().Exec(sc)
	}); err != nil {
		t.Fatal(err)
	}
	text := callMCP(t, client, "read_file_text", map[string]any{"file_id": f.fileID})
	if text["available"] != false || text["text"] != "" {
		t.Fatalf("unavailable OCR exposed stale content: %v", text)
	}
	root := callMCP(t, client, "get_file", map[string]any{"file_id": rootID})
	if root["is_directory"] != true {
		t.Fatalf("folder inspection failed: %v", root)
	}
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		sc := ctxx.NewSpaceContext(tc, tc.TTx.Space.Query().Where(
			space.PublicID(entx.NewCIText(f.spaceID)),
		).OnlyX(tc))
		return sc.TTx.File.Update().Where(file.IsDirectory(false)).SetDeletedAt(time.Now()).Exec(sc)
	}); err != nil {
		t.Fatal(err)
	}
	page := callMCP(t, client, "list_inbox", map[string]any{})
	if len(page["files"].([]any)) != 0 || page["has_more"] != false {
		t.Fatalf("deleted files remained in Inbox: %v", page)
	}
	for _, call := range []struct {
		name string
		args map[string]any
	}{
		{"get_file", map[string]any{"file_id": f.fileID}},
		{"get_file", map[string]any{"file_id": 42}},
		{"upload_file", map[string]any{}},
	} {
		result, err := client.CallTool(context.Background(), &sdk.CallToolParams{
			Name: call.name, Arguments: call.args,
		})
		if err == nil && !result.IsError {
			t.Fatalf("unsupported/invalid call succeeded: %s", call.name)
		}
	}
	h.mainDB.ReadWriteConn.Tenant.UpdateOneID(f.tenant.ID).
		SetMaintenanceModeEnabledAt(time.Now()).ExecX(context.Background())
	if _, err := client.ListTools(context.Background(), nil); err == nil {
		t.Fatal("tenant maintenance did not stop MCP access")
	}
}

func TestMCPConnectionCredentialFormAndValidation(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixture(t, h, "form")
	form := f.browser(h.actions.Dashboard.CreateMCPCredentialCmd.FormEndpoint(), nil)
	if form.Code != http.StatusOK || !strings.Contains(form.Body.String(), `<select`) ||
		!strings.Contains(form.Body.String(), `name="Destination"`) ||
		!strings.Contains(form.Body.String(), "MCP form") {
		t.Fatalf("credential form missing accessible destination: %d %s", form.Code, form.Body.String())
	}
	invalid := f.browser(h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(), url.Values{
		"Label":       {strings.Repeat("x", 101)},
		"Destination": {f.tenant.PublicID.String() + ":" + f.spaceID},
	})
	if invalid.Code != http.StatusBadRequest || strings.Contains(invalid.Body.String(), "sdmcp_") {
		t.Fatalf("invalid label accepted: %d", invalid.Code)
	}
	actor, tenantx := signUpAccount(t, h, "mcp-no-space@example.com")
	empty := &mcpFixture{
		h:       h,
		account: actor,
		tenant:  tenantx,
		session: createSessionForAccountForRulesTest(t, h, actor.ID),
	}
	response := empty.browser(h.actions.Dashboard.CreateMCPCredentialCmd.FormEndpoint(), nil)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), "No spaces available yet.") ||
		strings.Contains(response.Body.String(), `name="Destination"`) {
		t.Fatalf("no-Space form: %d %s", response.Code, response.Body.String())
	}
}

func callMCP(t *testing.T, client *sdk.ClientSession, name string, arguments map[string]any) map[string]any {
	t.Helper()
	result, err := client.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil || result.IsError {
		t.Fatalf("%s: %v, result=%+v", name, err, result)
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]any
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatal(err)
	}
	return output
}
