package server

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/util/cookiex"
)

func TestCreateMCPCredentialReturnsSecretAndInvalidatesListForSeparateQuery(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixture(t, h, "command-query")
	var destinationSpaceID string
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, h.actions, tenantCtx, "Zulu archive")
		destinationSpaceID = tenantCtx.TTx.Space.Query().Where(space.Name("Zulu archive")).
			OnlyX(tenantCtx).PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	created := f.browser(h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(), url.Values{
		"Label":       {"Archive client"},
		"Destination": {f.tenant.PublicID.String() + ":" + destinationSpaceID},
	})
	body := created.Body.String()
	if created.Code != http.StatusOK || !strings.Contains(body, "Copy the secret now") ||
		strings.Count(body, "<dialog") != 1 {
		t.Fatalf("expected one-time secret dialog, status %d", created.Code)
	}
	if strings.Contains(body, `id="mcpCredentials"`) ||
		strings.Contains(body, `hx-swap-oob="outerHTML"`) {
		t.Fatal("create command must not return credential list HTML")
	}
	secret := regexp.MustCompile(`sdmcp_[a-z0-9]+\.[A-Za-z0-9_-]{43}`).FindString(body)
	if secret == "" {
		t.Fatal("expected generated MCP secret in the one-time dialog")
	}
	var events map[string]struct {
		Destination string `json:"destination"`
	}
	if err := json.Unmarshal([]byte(created.Header().Get("HX-Trigger")), &events); err != nil {
		t.Fatalf("decode creation invalidation event %q: %v", created.Header().Get("HX-Trigger"), err)
	}
	if event, ok := events["mcpCredentialChanged"]; !ok || event.Destination !=
		fmt.Sprintf("%s:%s", f.tenant.PublicID, destinationSpaceID) {
		t.Fatalf("unexpected credential invalidation event: %#v", events)
	}
	if strings.Contains(body, "snackbar") {
		t.Fatal("one-time secret response must not include a second snackbar")
	}

	partial := f.browserAt("/dashboard/mcp-credentials/", h.actions.Dashboard.MCPCredentialListPartial.Endpoint(), url.Values{
		"CreatedDestination": {f.tenant.PublicID.String() + ":" + destinationSpaceID},
	})
	partialBody := partial.Body.String()
	if partial.Code != http.StatusOK || !strings.Contains(partialBody, `role="tab"`) ||
		!strings.Contains(activeTabLabel(t, partialBody), "Zulu archive") ||
		!strings.Contains(partialBody, "Archive client") || strings.Contains(partialBody, secret) {
		t.Fatalf("separate list query should select the created destination without returning its secret; status %d", partial.Code)
	}
}

func TestCreateWebDAVCredentialInvalidatesAndQueriesSelectedDestination(t *testing.T) {
	t.Setenv("SIMPLEDMS_PUBLIC_ORIGIN", "")
	h := newActionTestHarness(t)
	accountx, tenantx := signUpAccount(t, h, "webdav-command-query@example.com")
	tenantDB := initTenantDB(t, h, tenantx)
	var destinationSpaceID string
	if err := withTenantContext(t, h, accountx, tenantx, tenantDB, func(
		_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, h.actions, tenantCtx, "Zulu archive")
		destinationSpaceID = tenantCtx.TTx.Space.Query().Where(space.Name("Zulu archive")).
			OnlyX(tenantCtx).PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	session := createSessionForAccountForRulesTest(t, h, accountx.ID)
	commandPath := h.actions.Dashboard.CreateWebDAVCredentialCmd.Endpoint()
	command := newWebDAVCredentialRequest(commandPath, url.Values{
		"Destination": {tenantx.PublicID.String() + ":" + destinationSpaceID},
		"Label":       {"Archive scanner"},
	})
	command.Header.Set("HX-Request", "true")
	command.Header.Set("HX-Current-URL", "/dashboard/webdav-credentials/")
	command.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session})
	created := httptest.NewRecorder()
	h.router.ServeHTTP(created, command)
	body := created.Body.String()
	if created.Code != http.StatusOK || !strings.Contains(body, "Copy the secret now") ||
		strings.Count(body, "<dialog") != 1 {
		t.Fatalf("expected one-time WebDAV secret dialog, status %d", created.Code)
	}
	if strings.Contains(body, `id="webDAVCredentialOverview"`) {
		t.Fatal("create command must not return replacement overview HTML")
	}
	for _, field := range []string{"WebDAV URL", "WebDAV Inbox URL", "Username", "Secret"} {
		if !strings.Contains(body, field) {
			t.Fatalf("WebDAV secret dialog is missing %q", field)
		}
	}
	secretValue := regexp.MustCompile(`data-copy-value="([^"]+)"`).FindAllStringSubmatch(body, -1)
	if len(secretValue) == 0 {
		t.Fatal("expected copyable values in WebDAV result dialog")
	}
	secret := html.UnescapeString(secretValue[len(secretValue)-1][1])
	var events map[string]struct {
		Destination string `json:"destination"`
	}
	if err := json.Unmarshal([]byte(created.Header().Get("HX-Trigger")), &events); err != nil {
		t.Fatalf("decode WebDAV invalidation event %q: %v", created.Header().Get("HX-Trigger"), err)
	}
	if event, ok := events["webDAVCredentialChanged"]; !ok || event.Destination !=
		fmt.Sprintf("%s:%s", tenantx.PublicID, destinationSpaceID) {
		t.Fatalf("unexpected WebDAV creation event: %#v", events)
	}

	queryData := url.Values{
		"CreatedDestination": {tenantx.PublicID.String() + ":" + destinationSpaceID},
	}
	query := newWebDAVCredentialRequest(
		h.actions.Dashboard.WebDAVCredentialListPartial.Endpoint(), queryData,
	)
	query.Header.Set("HX-Request", "true")
	query.Header.Set("HX-Current-URL", "/dashboard/webdav-credentials/?credential_status=active")
	query.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: session})
	queried := httptest.NewRecorder()
	h.router.ServeHTTP(queried, query)
	queryBody := queried.Body.String()
	if queried.Code != http.StatusOK || !strings.Contains(queryBody, `role="tab"`) ||
		!strings.Contains(activeTabLabel(t, queryBody), "Zulu archive") ||
		!strings.Contains(queryBody, "Archive scanner") ||
		strings.Contains(queryBody, secret) {
		t.Fatalf("separate query should select created destination without returning its secret; status %d", queried.Code)
	}
	queryHTML := html.UnescapeString(queryBody)
	if !strings.Contains(queryHTML, `"CredentialStatusValues":["active"]`) ||
		strings.Contains(queryBody, "Revoked:") {
		t.Fatal("WebDAV query must retain active filter selection")
	}
}
