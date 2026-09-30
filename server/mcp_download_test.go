package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
)

func TestMCPDownloadRoundTrip(t *testing.T) {
	runWithFileEncryptionModes(t, func(t *testing.T, disableEncryption bool) {
		h := newActionTestHarnessWithS3AndEncryption(t, disableEncryption)
		fixture := newMCPFixtureWithWrites(t, h, "download")
		server := httptest.NewServer(h.router)
		t.Cleanup(server.Close)
		client := fixture.connect(t, server.URL+"/mcp")

		content := []byte{0, 1, 2, 3, 0xff, 0xfe, 0xfd, 0x80, 0x81, 0x00, 0x7f, 0x42, 0x43, 0x44, 0x45}
		upload := callMCP(t, client, "upload_file", map[string]any{
			"filename":       "download.bin",
			"content_base64": base64.StdEncoding.EncodeToString(content),
		})
		fileID := upload["file_id"].(string)
		if upload["size"] != float64(len(content)) {
			t.Fatalf("uploaded size = %v, want %d", upload["size"], len(content))
		}

		var downloaded []byte
		offset := int64(0)
		version := 0
		chunkIndex := 0
		for {
			length := 7
			if chunkIndex%2 == 1 {
				length = 8
			}
			args := map[string]any{"file_id": fileID, "offset": offset, "length": length}
			if version != 0 {
				args["version_number"] = version
			}
			result := callMCP(t, client, "download_file", args)
			if result["file_id"] != fileID || result["filename"] != "download.bin" ||
				result["offset"] != float64(offset) || result["size"] != float64(len(content)) {
				t.Fatalf("unexpected download metadata: %v", result)
			}
			if version == 0 {
				version = int(result["version_number"].(float64))
				if version < 1 {
					t.Fatalf("download version = %d", version)
				}
				wantSHA256 := sha256.Sum256(content)
				if result["content_sha256"] != hex.EncodeToString(wantSHA256[:]) {
					t.Fatalf("content_sha256 = %v, want %x", result["content_sha256"], wantSHA256)
				}
			}
			chunk, err := base64.StdEncoding.DecodeString(result["content_base64"].(string))
			if err != nil {
				t.Fatal(err)
			}
			downloaded = append(downloaded, chunk...)
			hasMore := result["has_more"].(bool)
			if !hasMore {
				if _, ok := result["next_offset"]; ok {
					t.Fatalf("final download returned next_offset: %v", result)
				}
				break
			}
			next := result["next_offset"].(float64)
			offset = int64(next)
			if offset != int64(len(downloaded)) {
				t.Fatalf("next offset = %d, downloaded = %d", offset, len(downloaded))
			}
			chunkIndex++
		}
		if string(downloaded) != string(content) {
			t.Fatalf("downloaded bytes = %v, want %v", downloaded, content)
		}

		empty := callMCP(t, client, "download_file", map[string]any{
			"file_id": fileID, "version_number": version, "offset": int64(len(content)), "length": 7,
		})
		if empty["content_base64"] != "" || empty["has_more"] != false || empty["offset"] != float64(len(content)) {
			t.Fatalf("unexpected empty terminal chunk: %v", empty)
		}

		data := url.Values{
			"Label":       {"MCP download read-only"},
			"Destination": {fixture.tenant.PublicID.String() + ":" + fixture.spaceID},
		}
		response := fixture.browser(h.actions.Dashboard.CreateMCPCredentialCmd.Endpoint(), data)
		if response.Code != 200 {
			t.Fatalf("create read-only credential: %d %s", response.Code, response.Body.String())
		}
		readOnly := *fixture
		readOnly.token = regexp.MustCompile(`sdmcp_[a-z0-9]+\.[A-Za-z0-9_-]{43}`).FindString(response.Body.String())
		if readOnly.token == "" {
			t.Fatalf("missing read-only token: %s", response.Body.String())
		}
		readOnlyClient := readOnly.connect(t, server.URL+"/mcp")
		readOnlyResult := callMCP(t, readOnlyClient, "download_file", map[string]any{
			"file_id": fileID, "version_number": version, "length": len(content),
		})
		got, err := base64.StdEncoding.DecodeString(readOnlyResult["content_base64"].(string))
		if err != nil || string(got) != string(content) {
			t.Fatalf("read-only download = %v, want %v (err=%v)", got, content, err)
		}

		for _, args := range []map[string]any{
			{"file_id": fileID, "offset": int64(-1), "length": 7},
			{"file_id": fileID, "offset": int64(0), "length": 0},
			{"file_id": fileID, "offset": int64(0), "length": 1024*1024 + 1},
			{"file_id": fileID, "offset": int64(1), "length": 7},
			{"file_id": fileID, "version_number": 999, "length": 7},
			{"file_id": fixture.rootID, "length": 7},
			{"file_id": "missing-file", "length": 7},
		} {
			assertMCPToolError(t, client, "download_file", args)
		}
	})
}

func TestMCPDownloadRejectsForeignSpaceFile(t *testing.T) {
	h := newActionTestHarness(t)
	fixture := newMCPFixture(t, h, "download-isolation")
	var foreignFileID string
	if err := withTenantContext(t, h, fixture.account, fixture.tenant, fixture.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, h.actions, tenantCtx, "MCP download foreign")
		foreignSpace := tenantCtx.TTx.Space.Query().Where(space.Name("MCP download foreign")).OnlyX(tenantCtx)
		foreign := ctxx.NewSpaceContext(tenantCtx, foreignSpace)
		file := createRegularFileForTest(foreign, foreign.SpaceRootDir().ID, "foreign.bin").Data
		foreignFileID = file.PublicID.String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := fixture.connect(t, server.URL+"/mcp")
	assertMCPToolError(t, client, "download_file", map[string]any{
		"file_id": foreignFileID, "length": 7,
	})
}
