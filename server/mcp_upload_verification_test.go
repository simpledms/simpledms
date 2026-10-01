package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"testing/iotest"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/mcpcredential"
	mainprivacy "github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	tenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	credentialmodel "github.com/simpledms/simpledms/model/main/mcpcredential"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
)

func TestMCPUploadAcceptsConfiguredSizeBoundariesWithoutDuplicates(t *testing.T) {
	h := newActionTestHarnessWithS3AndEncryption(t, true)
	f := newMCPFixtureWithWrites(t, h, "upload-boundaries")
	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")

	const mib = 1024 * 1024
	h.mainDB.ReadWriteConn.SystemConfig.Update().SetMaxUploadSizeMib(0).ExecX(context.Background())
	for _, test := range []struct {
		name string
		size int
	}{
		{name: "unlimited-10MiB", size: 10 * mib},
		{name: "configured-1MiB", size: mib},
	} {
		content := bytes.Repeat([]byte("m"), test.size)
		result := callMCP(t, client, "upload_file", map[string]any{
			"filename":       test.name + ".bin",
			"content_base64": base64.StdEncoding.EncodeToString(content),
		})
		fileID := stringField(t, result, "file_id")
		if result["size"] != float64(test.size) {
			t.Fatalf("%s size = %v, want %d", test.name, result["size"], test.size)
		}
		metadata := callMCP(t, client, "get_file", map[string]any{"file_id": fileID})
		if metadata["size"] != float64(test.size) || metadata["source"] != filesource.MCP.String() {
			t.Fatalf("%s metadata = %v", test.name, metadata)
		}
		if test.size == 10*mib {
			h.mainDB.ReadWriteConn.SystemConfig.Update().SetMaxUploadSizeMib(1).ExecX(context.Background())
		}
	}

	assertMCPToolError(t, client, "upload_file", map[string]any{
		"filename":       "configured-1MiB-plus-one.bin",
		"content_base64": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("m"), mib+1)),
	})
	if err := withTenantContext(t, h, f.account, f.tenant, f.db, func(
		_ *entmain.Tx, _ *enttenant.Tx, tc *ctxx.TenantContext,
	) error {
		spacex := tc.TTx.Space.Query().Where(space.PublicID(entx.NewCIText(f.spaceID))).OnlyX(tc)
		sc := ctxx.NewSpaceContext(tc, spacex)
		if count := tc.TTx.File.Query().Where(
			file.SpaceID(spacex.ID), file.Name("configured-1MiB-plus-one.bin"),
		).CountX(sc); count != 0 {
			return errors.New("rejected upload left a duplicate file")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestMCPUploadIngestionFailuresLeaveNoVisibleFileAndRecover(t *testing.T) {
	h := newActionTestHarnessWithS3AndEncryption(t, true)
	f := newMCPFixtureWithWrites(t, h, "upload-recovery")

	_, err := credentialmodel.NewCredentialService().Execute(
		context.Background(), h.mainDB, h.tenantDBs, h.i18n, false, f.token, func(
			sc *ctxx.SpaceContext, _ *entmain.MCPCredential,
		) error {
			service := filesystem.NewFileIngestionService(h.infra.FileSystem())
			for _, test := range []struct {
				name     string
				reader   io.Reader
				expected int64
			}{
				{
					name:     "cancelled.txt",
					reader:   io.MultiReader(bytes.NewReader([]byte("cancel")), iotest.ErrReader(context.Canceled)),
					expected: 12,
				},
				{name: "short.txt", reader: bytes.NewReader([]byte("short")), expected: 99},
			} {
				_, ingestErr := service.Ingest(sc, test.reader, test.name, sc.SpaceRootDir().ID, true,
					filesource.MCP, &test.expected, func(context.Context, *entmain.Tx) error { return nil })
				if ingestErr == nil {
					return errors.New("invalid upload was finalized")
				}
				if count := sc.TTx.File.Query().Where(file.Name(test.name)).CountX(sc); count != 0 {
					return errors.New("failed upload left a visible file")
				}
			}
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(h.router)
	t.Cleanup(server.Close)
	client := f.connect(t, server.URL+"/mcp")
	result := callMCP(t, client, "upload_file", map[string]any{
		"filename":       "recovered.txt",
		"content_base64": base64.StdEncoding.EncodeToString([]byte("recovered")),
	})
	if result["filename"] != "recovered.txt" {
		t.Fatalf("recovery result = %v", result)
	}
}

func TestMCPUploadLosesSpaceAccessDuringFinalizationWithoutVisibleFile(t *testing.T) {
	h := newActionTestHarnessWithS3AndEncryption(t, true)
	f := newMCPFixtureWithWrites(t, h, "upload-space-revoked")
	service := credentialmodel.NewCredentialService()

	expected := int64(len("space access"))
	finalizationReached := false
	_, err := service.Execute(
		context.Background(), h.mainDB, h.tenantDBs, h.i18n, false, f.token,
		func(sc *ctxx.SpaceContext, credential *entmain.MCPCredential) error {
			assignment := sc.TTx.SpaceUserAssignment.Query().Where(
				spaceuserassignment.SpaceID(sc.Space.ID), spaceuserassignment.UserID(sc.User.ID),
			).OnlyX(sc)
			_, err := filesystem.NewFileIngestionService(h.infra.FileSystem()).Ingest(
				sc, bytes.NewReader([]byte("space access")), "space-revoked.txt", sc.SpaceRootDir().ID,
				true, filesource.MCP, &expected, func(checkCtx context.Context, tx *entmain.Tx) error {
					finalizationReached = true
					deleteCtx := tenantprivacy.DecisionContext(context.Background(), tenantprivacy.Allow)
					if err := f.db.ReadWriteConn.User.UpdateOneID(sc.User.ID).SetRole(tenantrole.User).
						Exec(deleteCtx); err != nil {
						return err
					}
					if err := f.db.ReadWriteConn.SpaceUserAssignment.DeleteOneID(assignment.ID).Exec(deleteCtx); err != nil {
						return err
					}
					activeCredential, err := tx.MCPCredential.Query().Where(
						mcpcredential.PublicID(entx.NewCIText(f.credentialID)),
					).Only(mainprivacy.DecisionContext(checkCtx, mainprivacy.Allow))
					if err != nil {
						return err
					}
					if activeCredential.ID != credential.ID {
						return errors.New("finalization checked the wrong credential")
					}
					return service.AuthorizeFinalization(checkCtx, tx, activeCredential)
				})
			return err
		},
	)
	if err == nil || !finalizationReached {
		t.Fatal("upload finalized after space access was removed")
	}
	queryCtx := tenantprivacy.DecisionContext(context.Background(), tenantprivacy.Allow)
	if count := f.db.ReadOnlyConn.File.Query().Where(file.Name("space-revoked.txt")).CountX(queryCtx); count != 0 {
		t.Fatal("Space-revoked upload left a visible file")
	}
}
