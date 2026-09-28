package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/cookiex"
)

type mcpFixture struct {
	h            *actionTestHarness
	account      *entmain.Account
	tenant       *entmain.Tenant
	db           *sqlx.TenantDB
	spaceID      string
	rootID       string
	fileID       string
	token        string
	credentialID string
	session      string
}

func newMCPFixture(t *testing.T, h *actionTestHarness, suffix string) *mcpFixture {
	t.Helper()
	actor, tenantx := signUpAccount(t, h, "mcp-"+suffix+"@example.com")
	db := initTenantDB(t, h, tenantx)
	f := &mcpFixture{
		h:       h,
		account: actor,
		tenant:  tenantx,
		db:      db,
		session: createSessionForAccountForRulesTest(t, h, actor.ID),
	}
	if err := withTenantContext(t, h, actor, tenantx, db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, h.actions, tc, "MCP "+suffix)
		spacex := tc.TTx.Space.Query().Where(space.Name("MCP " + suffix)).OnlyX(tc)
		f.spaceID = spacex.PublicID.String()
		sc := ctxx.NewSpaceContext(tc, spacex)
		f.rootID = sc.SpaceRootDir().PublicID.String()
		for index := range 2 {
			file := createRegularFileForTest(sc, sc.SpaceRootDir().ID,
				fmt.Sprintf("mcp-%s-%d.txt", suffix, index)).Data.Update().
				SetIsInInbox(true).SetOcrContent("ä🙂abcdef").SetOcrSuccessAt(time.Now()).SaveX(sc)
			if index == 0 {
				f.fileID = file.PublicID.String()
			}
			if err := seedStoredFilesForBenchmarkRows(sc, []*enttenant.File{file}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rr := f.browser(h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(), url.Values{
		"Label":       {"MCP " + suffix},
		"Destination": {tenantx.PublicID.String() + ":" + f.spaceID},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("create credential: %d %s", rr.Code, rr.Body.String())
	}
	f.token = regexp.MustCompile(`sdmcp_[a-z0-9]+\.[A-Za-z0-9_-]{43}`).FindString(rr.Body.String())
	if f.token == "" {
		t.Fatalf("missing token in creation response: %s", rr.Body.String())
	}
	f.credentialID, _, _ = strings.Cut(strings.TrimPrefix(f.token, "sdmcp_"), ".")
	return f
}

func (qq *mcpFixture) browser(endpoint string, data url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Current-URL", route.MCPCredentials())
	req.AddCookie(&http.Cookie{Name: cookiex.SessionCookieName(), Value: qq.session})
	rr := httptest.NewRecorder()
	qq.h.router.ServeHTTP(rr, req)
	return rr
}

type mcpRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn mcpRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func (qq *mcpFixture) connect(t *testing.T, endpoint string) *sdk.ClientSession {
	t.Helper()
	client := sdk.NewClient(&sdk.Implementation{Name: "slice-test", Version: "1"}, nil)
	httpClient := &http.Client{Transport: mcpRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+qq.token)
		return http.DefaultTransport.RoundTrip(req)
	})}
	session, err := client.Connect(context.Background(), &sdk.StreamableClientTransport{
		Endpoint:             endpoint,
		HTTPClient:           httpClient,
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}
