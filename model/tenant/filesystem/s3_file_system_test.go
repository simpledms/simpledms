package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"filippo.io/age"
	_ "github.com/mattn/go-sqlite3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/enttest"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/encryptor"
	"github.com/simpledms/simpledms/util/e"
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
	tenantCtx := &ctxx.TenantContext{
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
	tenantCtx := &ctxx.TenantContext{
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

func TestS3FileSystemVerifyObjectAllowsMissingChecksumMode(t *testing.T) {
	const checksum = "crc32c"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Length", "4")
		rw.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		rw.Header().Set("X-Amz-Checksum-Crc32c", checksum)
		rw.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := minio.New(strings.TrimPrefix(server.URL, "http://"), &minio.Options{
		Creds:  credentials.NewStaticV4("access-key", "secret-key", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("create S3 client: %v", err)
	}
	fileSystemx := &S3FileSystem{client: client, bucketName: "bucket"}

	if err := fileSystemx.verifyObject(context.Background(), "object", 4, checksum); err != nil {
		t.Fatalf("verify object without checksum mode: %v", err)
	}
}

func TestMaxBytesReaderAcceptsExactLimitAndRejectsFirstExtraByte(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		max     int64
		want    string
		wantErr error
	}{
		{name: "below", body: "ab", max: 3, want: "ab"},
		{name: "exact", body: "abc", max: 3, want: "abc"},
		{name: "over", body: "abcd", max: 3, want: "abc", wantErr: errUploadTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &maxBytesReader{r: strings.NewReader(tc.body), max: tc.max}
			got, err := io.ReadAll(reader)
			if string(got) != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, string(got))
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
		})
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
		VisitorContext: &ctxx.VisitorContext{
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
