package server

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/util/httpx"
)

func TestUploadFileVersionCmdAcceptsFilenameOfVersionedFile(t *testing.T) {
	harness := newActionTestHarnessWithS3(t)
	account, tenant := signUpAccount(t, harness, "same-name-version@example.com")
	tenantDB := initTenantDB(t, harness, tenant)
	tenant = harness.mainDB.ReadWriteConn.Tenant.GetX(context.Background(), tenant.ID)

	var doc *enttenant.File
	var spacex *enttenant.Space
	err := withTenantContext(t, harness, account, tenant, tenantDB, func(
		_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
	) error {
		createSpaceViaCmd(t, harness.actions, tenantCtx, "Same name versions")
		spacex = tenantCtx.TTx.Space.Query().Where(space.Name("Same name versions")).OnlyX(tenantCtx)
		ctx := ctxx.NewSpaceContext(tenantCtx, spacex)
		doc = uploadSpaceFile(t, harness, ctx, ctx.SpaceRootDir().ID, "contract.pdf", []byte("draft"), false)
		return nil
	})
	if err != nil {
		t.Fatalf("upload original version: %v", err)
	}

	// The command owns its write transactions, so its request context must be read-only.
	mainTx, tenantTx, tenantCtx, err := newTenantContextForUpload(harness, account, tenant, tenantDB)
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if err := mainTx.Rollback(); err != nil {
				t.Error(err)
			}
		}()
		defer func() {
			if err := tenantTx.Rollback(); err != nil {
				t.Error(err)
			}
		}()

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("FileID", doc.PublicID.String()); err != nil {
			t.Fatal(err)
		}
		part, err := writer.CreateFormFile("File", "contract.pdf")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte("signed")); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/-/browse/upload-file-version-cmd", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		err = harness.actions.Browse.UploadFileVersionCmd.Handler(
			httpx.NewResponseWriter(httptest.NewRecorder()),
			httpx.NewRequest(req),
			ctxx.NewSpaceContext(tenantCtx, spacex),
		)
		if err != nil {
			t.Fatalf("upload version with the same filename: %v", err)
		}
	}()

	err = withTenantContext(t, harness, account, tenant, tenantDB, func(
		_ *entmain.Tx, _ *enttenant.Tx, tenantCtx *ctxx.TenantContext,
	) error {
		ctx := ctxx.NewSpaceContext(tenantCtx, spacex)
		if count := ctx.TTx.FileVersion.Query().Where(fileversion.FileID(doc.ID)).CountX(ctx); count != 2 {
			t.Fatalf("expected 2 versions, got %d", count)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("verify versions: %v", err)
	}
}
