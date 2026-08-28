package filesystem

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"entgo.io/ent/privacy"
	"filippo.io/age"
	_ "github.com/mattn/go-sqlite3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	entmaintest "github.com/simpledms/simpledms/db/entmain/enttest"
	"github.com/simpledms/simpledms/db/enttenant"
	enttenanttest "github.com/simpledms/simpledms/db/enttenant/enttest"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/encryptor"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
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
		Context:     mainCtx,
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
		Context:     mainCtx,
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

func TestS3FileSystemVerifyObjectUsesSHA256(t *testing.T) {
	const contents = "data"
	checksum := sha256.Sum256([]byte(contents))
	storageSHA256 := hex.EncodeToString(checksum[:])

	for _, tc := range []struct {
		name     string
		expected string
		wantErr  bool
	}{
		{name: "matching contents", expected: storageSHA256},
		{name: "mismatching contents", expected: "wrong", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var getRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.Header().Set("Content-Length", "4")
				rw.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
				rw.Header().Set("X-Amz-Checksum-Crc32c", "crc32c")
				rw.Header().Set("X-Amz-Checksum-Type", minio.ChecksumFullObjectMode.String())
				if req.Method == http.MethodGet {
					getRequests.Add(1)
					_, _ = io.WriteString(rw, contents)
				}
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

			err = fileSystemx.verifyObject(context.Background(), "object", 4, tc.expected)
			if (err != nil) != tc.wantErr {
				t.Fatalf("verify object error = %v, want error %v", err, tc.wantErr)
			}
			if getRequests.Load() == 0 {
				t.Fatal("expected object contents to be verified")
			}
		})
	}
}

func TestS3FileSystemRemoveObjectCompletelyDeletesRetainedVersions(t *testing.T) {
	var mu sync.Mutex
	var deletedVersions []string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodGet && req.URL.Query().Has("versions"):
			_, _ = io.WriteString(rw, `<?xml version="1.0" encoding="UTF-8"?>
<ListVersionsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>bucket</Name><Prefix>object</Prefix><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated>
  <Version><Key>object</Key><VersionId>v1</VersionId><IsLatest>false</IsLatest>
    <LastModified>2026-08-28T08:00:00.000Z</LastModified><ETag>"etag"</ETag>
    <Size>4</Size><StorageClass>STANDARD</StorageClass>
  </Version>
  <DeleteMarker><Key>object</Key><VersionId>marker1</VersionId><IsLatest>true</IsLatest>
    <LastModified>2026-08-28T09:00:00.000Z</LastModified>
  </DeleteMarker>
</ListVersionsResult>`)
		case req.Method == http.MethodDelete:
			if versionID := req.URL.Query().Get("versionId"); versionID != "" {
				mu.Lock()
				deletedVersions = append(deletedVersions, versionID)
				mu.Unlock()
			}
			rw.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected S3 request: %s %s", req.Method, req.URL.String())
			rw.WriteHeader(http.StatusBadRequest)
		}
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

	if err := fileSystemx.RemoveObjectCompletely(context.Background(), "object"); err != nil {
		t.Fatalf("remove object completely: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if fmt.Sprint(deletedVersions) != "[v1 marker1]" {
		t.Fatalf("expected retained versions to be deleted, got %v", deletedVersions)
	}
}

func TestS3FileSystemCopyFallsBackToStreamedPut(t *testing.T) {
	const contents = "data"
	checksum := sha256.Sum256([]byte(contents))
	storageSHA256 := hex.EncodeToString(checksum[:])
	objects := map[string][]byte{"/bucket/source": []byte(contents)}
	var objectsMu sync.RWMutex
	var copyRequests atomic.Int32
	var putRequests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPut && req.Header.Get("X-Amz-Copy-Source") != "" {
			copyRequests.Add(1)
			rw.Header().Set("Content-Type", "application/xml")
			rw.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(rw, `<Error><Code>InvalidRequest</Code><Message>copy unavailable</Message></Error>`)
			return
		}

		switch req.Method {
		case http.MethodHead:
			objectsMu.RLock()
			contents, ok := objects[req.URL.Path]
			objectsMu.RUnlock()
			if !ok {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			rw.Header().Set("Content-Length", fmt.Sprint(len(contents)))
			rw.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		case http.MethodGet:
			objectsMu.RLock()
			contents, ok := objects[req.URL.Path]
			objectsMu.RUnlock()
			if !ok {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			rw.Header().Set("Content-Length", fmt.Sprint(len(contents)))
			rw.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
			_, _ = rw.Write(contents)
		case http.MethodPut:
			contents, err := io.ReadAll(req.Body)
			if err != nil {
				t.Errorf("read put body: %v", err)
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}
			if req.Header.Get("Content-Encoding") == "aws-chunked" {
				lineEnd := bytes.Index(contents, []byte("\r\n"))
				if lineEnd < 0 {
					t.Error("invalid aws-chunked body")
					rw.WriteHeader(http.StatusInternalServerError)
					return
				}
				chunkHeader := strings.SplitN(string(contents[:lineEnd]), ";", 2)[0]
				chunkSize, err := strconv.ParseInt(chunkHeader, 16, 64)
				if err != nil {
					t.Errorf("parse aws chunk size: %v", err)
					rw.WriteHeader(http.StatusInternalServerError)
					return
				}
				contents = contents[lineEnd+2 : lineEnd+2+int(chunkSize)]
			}
			putRequests.Add(1)
			objectsMu.Lock()
			objects[req.URL.Path] = contents
			objectsMu.Unlock()
			rw.Header().Set("ETag", `"etag"`)
		default:
			t.Errorf("unexpected S3 request: %s %s", req.Method, req.URL.String())
			rw.WriteHeader(http.StatusBadRequest)
		}
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
	storedFile := &enttenant.StoredFile{SizeInStorage: 4, Sha256: storageSHA256}

	if err := fileSystemx.copyVerifiedTemporaryTenantObject(
		context.Background(),
		storedFile,
		"source",
		"destination",
	); err != nil {
		t.Fatalf("copy with streamed fallback: %v", err)
	}
	if copyRequests.Load() != 1 || putRequests.Load() != 1 {
		t.Fatalf("expected one copy and one streamed put, got copy=%d put=%d", copyRequests.Load(), putRequests.Load())
	}
	objectsMu.RLock()
	destinationContents := string(objects["/bucket/destination"])
	objectsMu.RUnlock()
	if destinationContents != contents {
		t.Fatalf("unexpected destination contents: %q", destinationContents)
	}
}

func TestS3FileSystemEstablishesMissingStoredSHA256(t *testing.T) {
	const contents = "data"
	checksum := sha256.Sum256([]byte(contents))
	storageSHA256 := hex.EncodeToString(checksum[:])
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Length", "4")
		rw.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		_, _ = io.WriteString(rw, contents)
	}))
	defer server.Close()

	client, err := minio.New(strings.TrimPrefix(server.URL, "http://"), &minio.Options{
		Creds:  credentials.NewStaticV4("access-key", "secret-key", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("create S3 client: %v", err)
	}
	tenantClient := enttenanttest.Open(t, "sqlite3", "file:missing-stored-sha?mode=memory&cache=shared&_fk=1")
	defer tenantClient.Close()
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	storedFile := tenantClient.StoredFile.Create().
		SetFilename("document.pdf").
		SetSize(4).
		SetSizeInStorage(4).
		SetStorageType(storagetype.S3).
		SetStoragePath("tenant/final").
		SetStorageFilename("object").
		SetTemporaryStoragePath("tenant/tmp").
		SetTemporaryStorageFilename("object").
		SetUploadSucceededAt(time.Now()).
		SaveX(ctx)
	fileSystemx := &S3FileSystem{client: client, bucketName: "bucket"}

	updated, err := fileSystemx.ensureStoredFileSHA256(ctx, storedFile, "object")
	if err != nil {
		t.Fatalf("establish missing stored SHA-256: %v", err)
	}
	if updated.Sha256 != storageSHA256 {
		t.Fatalf("expected SHA-256 %q, got %q", storageSHA256, updated.Sha256)
	}
	if reloaded := tenantClient.StoredFile.GetX(ctx, storedFile.ID); reloaded.Sha256 != storageSHA256 {
		t.Fatalf("expected persisted SHA-256 %q, got %q", storageSHA256, reloaded.Sha256)
	}
}

func TestS3FileSystemVerifyObjectDoesNotRequireChecksumHead(t *testing.T) {
	const contents = "data"
	checksum := sha256.Sum256([]byte(contents))
	storageSHA256 := hex.EncodeToString(checksum[:])

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodHead && req.Header.Get("X-Amz-Checksum-Mode") != "" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.Header().Set("Content-Length", "4")
		rw.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		if req.Method == http.MethodGet {
			_, _ = io.WriteString(rw, contents)
		}
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

	if err := fileSystemx.verifyObject(context.Background(), "object", 4, storageSHA256); err != nil {
		t.Fatalf("verify object without checksum HEAD support: %v", err)
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
	mainDB := entmaintest.Open(t, "sqlite3", dsn)

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
		Context: context.Background(),
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
