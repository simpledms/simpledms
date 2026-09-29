package server

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/mcpcredential"
	mainprivacy "github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	credentialmodel "github.com/simpledms/simpledms/model/main/mcpcredential"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/ui/uix/route"
)

func TestMCPUploadPersistsBytesAndSource(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		h := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)
		f := newMCPFixtureWithWrites(t, h, "upload")
		if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
			_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
		) error {
			return tenantCtx.TTx.Space.Update().Where(
				space.PublicID(entx.NewCIText(f.spaceID)),
			).SetIsFolderMode(true).Exec(tenantCtx)
		}); err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(h.router)
		t.Cleanup(server.Close)
		client := f.connect(t, server.URL+"/mcp")
		content := []byte("MCP upload content\n")

		result := callMCP(t, client, "upload_file", map[string]any{
			"filename":       "mcp-upload.txt",
			"content_base64": base64.StdEncoding.EncodeToString(content),
		})
		fileID, _ := result["file_id"].(string)
		wantURL := server.URL + route.BrowseFile(
			f.tenant.PublicID.String(), f.spaceID, f.rootID, fileID,
		)
		if fileID == "" || result["filename"] != "mcp-upload.txt" ||
			result["size"] != float64(len(content)) || result["is_in_inbox"] != true ||
			result["url"] != wantURL {
			t.Fatalf("unexpected upload result: %v", result)
		}
		metadata := callMCP(t, client, "get_file", map[string]any{"file_id": fileID})
		if metadata["source"] != filesource.MCP.String() || metadata["url"] != wantURL {
			t.Fatalf("unexpected uploaded metadata: %v", metadata)
		}
		assertMCPToolError(t, client, "upload_file", map[string]any{
			"filename":       "mcp-upload.txt",
			"content_base64": base64.StdEncoding.EncodeToString(content),
		})

		err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
			_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
		) error {
			filex := tenantCtx.TTx.File.Query().Where(
				file.PublicID(entx.NewCIText(fileID)),
			).OnlyX(tenantCtx)
			version := filex.QueryFileVersions().WithStoredFile().OnlyX(tenantCtx)
			reader, err := h.infra.FileSystem().OpenFile(
				tenantCtx, storedfilemodel.NewStoredFile(version.Edges.StoredFile),
			)
			if err != nil {
				return err
			}
			defer func() {
				if err := reader.Close(); err != nil {
					t.Errorf("close stored file: %v", err)
				}
			}()
			storedContent, err := io.ReadAll(reader)
			if err != nil {
				return err
			}
			if string(storedContent) != string(content) {
				t.Fatalf("stored content = %q, want %q", storedContent, content)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestMCPUploadRejectsInvalidInputAndReadOnlyCredential(t *testing.T) {
	h := newActionTestHarness(t)
	writeFixture := newMCPFixtureWithWrites(t, h, "upload-invalid")
	readFixture := newMCPFixture(t, h, "upload-read-only")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	writeClient := writeFixture.connect(t, server.URL+"/mcp")
	readClient := readFixture.connect(t, server.URL+"/mcp")

	inputs := []map[string]any{
		{"filename": "", "content_base64": "YQ=="},
		{"filename": ".", "content_base64": "YQ=="},
		{"filename": "folder/../file.txt", "content_base64": "YQ=="},
		{"filename": "empty.txt", "content_base64": ""},
		{"filename": "invalid.txt", "content_base64": "%%%="},
		{"filename": "multiline.txt", "content_base64": "YQ==\n"},
		{
			"filename": "too-large.txt",
			"content_base64": base64.StdEncoding.EncodeToString(
				make([]byte, 10*1024*1024+1),
			),
		},
	}
	for _, input := range inputs {
		assertMCPToolError(t, writeClient, "upload_file", input)
	}
	assertMCPToolError(t, readClient, "upload_file", map[string]any{
		"filename": "read-only.txt", "content_base64": "YQ==",
	})

	h.mainDB.ReadWriteConn.SystemConfig.Update().SetMaxUploadSizeMib(1).ExecX(context.Background())
	assertMCPToolError(t, writeClient, "upload_file", map[string]any{
		"filename":       "configured-limit.txt",
		"content_base64": base64.StdEncoding.EncodeToString(make([]byte, 1024*1024+1)),
	})
}

func TestMCPUploadRechecksRevocationBeforeFinalization(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		h := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)
		f := newMCPFixtureWithWrites(t, h, "upload-revoked")
		service := credentialmodel.NewCredentialService()
		content := []byte("revoked during upload")
		expectedBytes := int64(len(content))

		_, err := service.ExecuteWrite(
			context.Background(),
			h.mainDB,
			h.tenantDBs,
			h.i18n,
			false,
			f.token,
			func(spaceCtx *ctxx.SpaceContext, credential *entmain.MCPCredential) error {
				reader, writer := io.Pipe()
				writeDone := make(chan error, 1)
				go func() {
					ctx := mainprivacy.DecisionContext(context.Background(), mainprivacy.Allow)
					_, revokeErr := h.mainDB.ReadWriteConn.MCPCredential.Update().Where(
						mcpcredential.ID(credential.ID), mcpcredential.RevokedAtIsNil(),
					).SetRevokedAt(time.Now()).Save(ctx)
					if revokeErr == nil {
						_, revokeErr = writer.Write(content)
					}
					if closeErr := writer.CloseWithError(revokeErr); revokeErr == nil {
						revokeErr = closeErr
					}
					writeDone <- revokeErr
				}()
				mainCheck := func(checkCtx context.Context, tx *entmain.Tx) error {
					return service.AuthorizeFinalization(checkCtx, tx, credential)
				}
				_, ingestErr := filesystem.NewFileIngestionService(h.infra.FileSystem()).Ingest(
					spaceCtx,
					reader,
					"revoked-upload.txt",
					spaceCtx.SpaceRootDir().ID,
					true,
					filesource.MCP,
					&expectedBytes,
					mainCheck,
				)
				if writeErr := <-writeDone; writeErr != nil {
					t.Fatalf("write upload body: %v", writeErr)
				}
				return ingestErr
			},
		)
		if err == nil {
			t.Fatal("upload finalized after credential revocation")
		}
		err = withTenantContext(t, h, f.account, f.tenant, f.db, func(
			_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
		) error {
			if count := tenantCtx.TTx.File.Query().Where(
				file.Name("revoked-upload.txt"),
			).CountX(tenantCtx); count != 0 {
				t.Fatalf("revoked upload left %d visible files", count)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func assertMCPToolError(
	t *testing.T, client *sdk.ClientSession, name string, arguments map[string]any,
) {
	t.Helper()
	result, err := client.CallTool(context.Background(), &sdk.CallToolParams{
		Name: name, Arguments: arguments,
	})
	if err == nil && (result == nil || !result.IsError) {
		t.Fatalf("%s accepted invalid arguments", name)
	}
}

func TestMCPUploadRequestBodyLimitWithoutContentLength(t *testing.T) {
	h := newActionTestHarness(t)
	f := newMCPFixtureWithWrites(t, h, "upload-body-limit")
	body := strings.NewReader(strings.Repeat("x", 16*1024*1024+1))
	req := httptest.NewRequest("POST", "/mcp", body)
	req.ContentLength = -1
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	response := httptest.NewRecorder()
	h.router.ServeHTTP(response, req)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf(
			"oversized request without Content-Length returned %d, want %d",
			response.Code,
			http.StatusRequestEntityTooLarge,
		)
	}
}
