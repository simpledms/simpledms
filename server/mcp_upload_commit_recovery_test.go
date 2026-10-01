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
	"github.com/simpledms/simpledms/db/enttenant/file"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/model/main/common/plan"
	credentialmodel "github.com/simpledms/simpledms/model/main/mcpcredential"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
)

func TestMCPUploadEnforcesTenantStorageQuotaAndRecoversAfterUsageIsCleared(t *testing.T) {
	h := newActionTestHarnessWithS3AndEncryption(t, true)
	f := newMCPFixtureWithWrites(t, h, "quota-recovery")

	h.mainDB.ReadWriteConn.Tenant.UpdateOneID(f.tenant.ID).SetPlan(plan.Trial).ExecX(context.Background())
	privacyContext := tenantprivacy.DecisionContext(context.Background(), tenantprivacy.Allow)
	stored := f.db.ReadWriteConn.StoredFile.Query().FirstX(privacyContext)
	f.db.ReadWriteConn.StoredFile.UpdateOneID(stored.ID).
		SetSize(1 * 1024 * 1024 * 1024).ExecX(privacyContext)

	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")

	result, err := client.CallTool(context.Background(), &sdk.CallToolParams{
		Name: "upload_file", Arguments: map[string]any{
			"filename":       "quota-rejected.txt",
			"content_base64": "eA==",
		},
	})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("quota upload was accepted: err=%v result=%+v", err, result)
	}
	payload, marshalErr := json.Marshal(result.Content)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if !strings.Contains(string(payload), "invalid_input") ||
		!strings.Contains(string(payload), "Storage limit reached") {
		t.Fatalf("quota error = %s", payload)
	}
	if count := f.db.ReadOnlyConn.File.Query().Where(file.Name("quota-rejected.txt")).CountX(
		privacyContext,
	); count != 0 {
		t.Fatalf("rejected upload left %d visible files", count)
	}

	f.db.ReadWriteConn.StoredFile.UpdateOneID(stored.ID).SetSize(0).ExecX(privacyContext)
	accepted := callMCP(t, client, "upload_file", map[string]any{
		"filename":       "quota-accepted.txt",
		"content_base64": "eA==",
	})
	metadata := callMCP(t, client, "get_file", map[string]any{"file_id": accepted["file_id"]})
	if metadata["source"] != filesource.MCP.String() {
		t.Fatalf("accepted upload source = %v", metadata["source"])
	}
}

func TestMCPUploadReportsAmbiguousCommitButPreservesCanonicalFile(t *testing.T) {
	h := newActionTestHarnessWithS3AndEncryption(t, true)
	f := newMCPFixtureWithWrites(t, h, "commit-recovery")
	original := f.db.ReadWriteConn
	driver := &mcpCommitErrorDriver{Driver: original.Driver()}
	f.db.ReadWriteConn = enttenant.NewClient(enttenant.Driver(driver))
	t.Cleanup(func() { f.db.ReadWriteConn = original })

	var expected int64 = 7
	service := credentialmodel.NewCredentialService()
	ok, err := service.Execute(
		context.Background(), h.mainDB, h.tenantDBs, h.i18n, false, f.token,
		func(sc *ctxx.SpaceContext, _ *entmain.MCPCredential) error {
			_, ingestErr := filesystem.NewFileIngestionService(h.infra.FileSystem()).Ingest(
				sc, strings.NewReader("recover"), "ambiguous.txt", sc.SpaceRootDir().ID,
				true, filesource.MCP, &expected, func(context.Context, *entmain.Tx) error {
					driver.armed.Store(true)
					return nil
				},
			)
			return ingestErr
		},
	)
	if ok || err == nil || !driver.injected.Load() || driver.armed.Load() {
		t.Fatalf("ambiguous commit result: ok=%t err=%v injected=%t armed=%t",
			ok, err, driver.injected.Load(), driver.armed.Load())
	}

	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	assertMCPToolError(t, client, "upload_file", map[string]any{
		"filename": "ambiguous.txt", "content_base64": "cmVjb3Zlcg==",
	})
	filex := f.db.ReadOnlyConn.File.Query().Where(file.Name("ambiguous.txt")).OnlyX(
		tenantprivacy.DecisionContext(context.Background(), tenantprivacy.Allow),
	)
	metadata := callMCP(t, client, "get_file", map[string]any{
		"file_id": filex.PublicID.String(),
	})
	if metadata["source"] != filesource.MCP.String() {
		t.Fatalf("recovered file source = %v", metadata["source"])
	}
	download := callMCP(t, client, "download_file", map[string]any{
		"file_id": filex.PublicID.String(), "offset": 0, "length": 7,
	})
	if download["content_base64"] != "cmVjb3Zlcg==" {
		t.Fatalf("recovered file content = %v", download["content_base64"])
	}
	if count := f.db.ReadOnlyConn.File.Query().Where(file.Name("ambiguous.txt")).CountX(
		tenantprivacy.DecisionContext(context.Background(), tenantprivacy.Allow),
	); count != 1 {
		t.Fatalf("ambiguous upload count = %d, want 1", count)
	}
}
