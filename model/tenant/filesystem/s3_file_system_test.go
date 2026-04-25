package filesystem

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"filippo.io/age"
	_ "github.com/mattn/go-sqlite3"

	"github.com/marcobeierer/go-core/db/entmain"
	"github.com/marcobeierer/go-core/db/entmain/enttest"
	"github.com/marcobeierer/go-core/db/entx"
	"github.com/marcobeierer/go-core/encryptor"

	"github.com/marcobeierer/go-core/ctxx"
	ctxx2 "github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/util/e"
)

func TestS3FileSystemEnsureUploadSizeLimitWithGlobalLimit(t *testing.T) {
	fileSystemx, mainCtx, cleanup := newS3FileSystemMainContext(t, 1)
	defer cleanup()

	err := fileSystemx.EnsureUploadSizeLimit(mainCtx, bytesPerMiB)
	if err != nil {
		t.Fatalf("expected no error at exact limit, got %v", err)
	}

	err = fileSystemx.EnsureUploadSizeLimit(mainCtx, 2*bytesPerMiB)
	httpErr := requireHTTPErrorStatus(t, err, http.StatusRequestEntityTooLarge)
	if !strings.Contains(httpErr.Message(), "Maximum allowed size") {
		t.Fatalf("expected max size message, got %q", httpErr.Message())
	}
}

func TestS3FileSystemEnsureUploadSizeLimitWithTenantOverride(t *testing.T) {
	fileSystemx, mainCtx, cleanup := newS3FileSystemMainContext(t, 10)
	defer cleanup()

	overrideMib := int64(1)
	tenantCtx := &ctxx2.TenantContext{
		MainContext: mainCtx,
		Tenant: &entmain.Tenant{
			MaxUploadSizeMibOverride: &overrideMib,
		},
	}

	err := fileSystemx.EnsureUploadSizeLimit(tenantCtx, 2*bytesPerMiB)
	_ = requireHTTPErrorStatus(t, err, http.StatusRequestEntityTooLarge)
}

func TestS3FileSystemEnsureUploadSizeLimitWithUnlimitedTenantOverride(t *testing.T) {
	fileSystemx, mainCtx, cleanup := newS3FileSystemMainContext(t, 1)
	defer cleanup()

	overrideMib := int64(0)
	tenantCtx := &ctxx2.TenantContext{
		MainContext: mainCtx,
		Tenant: &entmain.Tenant{
			MaxUploadSizeMibOverride: &overrideMib,
		},
	}

	err := fileSystemx.EnsureUploadSizeLimit(tenantCtx, 2*bytesPerMiB)
	if err != nil {
		t.Fatalf("expected no error for unlimited tenant override, got %v", err)
	}
}

func TestS3FileSystemUploadTooLargeErrorWithoutMaximum(t *testing.T) {
	fileSystemx := &S3FileSystem{}

	err := fileSystemx.uploadTooLargeError(0)
	httpErr := requireHTTPErrorStatus(t, err, http.StatusRequestEntityTooLarge)
	if httpErr.Message() != "Upload is too large." {
		t.Fatalf("unexpected message: %q", httpErr.Message())
	}
}

func newS3FileSystemMainContext(t *testing.T, globalLimitMib int64) (*S3FileSystem, *ctxx.MainContext, func()) {
	t.Helper()

	oldIdentity := encryptor.NilableX25519MainIdentity
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate test identity: %v", err)
	}
	encryptor.NilableX25519MainIdentity = identity

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "_"))
	mainDB := enttest.Open(t, "sqlite3", dsn)

	mainDB.SystemConfig.Create().
		SetX25519Identity([]byte("identity")).
		SetIsIdentityEncryptedWithPassphrase(false).
		SetS3Endpoint("http://localhost").
		SetS3AccessKeyID("access-key").
		SetS3SecretAccessKey(entx.NewEncryptedString("secret")).
		SetS3BucketName("bucket").
		SetTLSEnableAutocert(false).
		SetTLSCertFilepath("").
		SetTLSPrivateKeyFilepath("").
		SetTLSAutocertEmail("").
		SetTLSAutocertHosts([]string{}).
		SetMaxUploadSizeMib(globalLimitMib).
		SaveX(context.Background())

	mainTx, err := mainDB.Tx(context.Background())
	if err != nil {
		t.Fatalf("start main transaction: %v", err)
	}

	mainCtx := &ctxx.MainContext{
		VisitorContext: &ctxx2.VisitorContext{
			Context: context.Background(),
			MainTx:  mainTx,
		},
	}

	fileSystemx := &S3FileSystem{}

	cleanup := func() {
		if err := mainTx.Rollback(); err != nil {
			t.Fatalf("rollback main transaction: %v", err)
		}
		if err := mainDB.Close(); err != nil {
			t.Fatalf("close main db: %v", err)
		}
		encryptor.NilableX25519MainIdentity = oldIdentity
	}

	return fileSystemx, mainCtx, cleanup
}

func requireHTTPErrorStatus(t *testing.T, err error, expectedStatus int) *e.HTTPError {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with status %d", expectedStatus)
	}

	httpErr, ok := err.(*e.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode() != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, httpErr.StatusCode())
	}

	return httpErr
}
