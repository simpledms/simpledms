package filesystem

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestUnsafeRestoreUploadUsesValidatedNameAndRebasedPrefix(t *testing.T) {
	var key string
	var object []byte
	var deleteCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key = strings.TrimPrefix(r.URL.Path, "/bucket/")
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `<Error><Code>NoSuchKey</Code></Error>`)
		case http.MethodPut:
			requestBody, _ := io.ReadAll(r.Body)
			lineEnd := bytes.Index(requestBody, []byte("\r\n"))
			if lineEnd >= 0 {
				chunkSize, parseErr := strconv.ParseInt(strings.SplitN(string(requestBody[:lineEnd]), ";", 2)[0], 16, 64)
				if parseErr == nil && chunkSize >= 0 {
					size := int(chunkSize)
					if len(requestBody) < lineEnd+2+size {
						t.Error("truncated aws-chunked part")
					} else {
						object = requestBody[lineEnd+2 : lineEnd+2+size]
					}
				}
			}
			if object == nil {
				if r.Header.Get("Content-Encoding") == "aws-chunked" {
					t.Error("aws-chunked part had an invalid size line")
				}
				object = requestBody
			}
			w.Header().Set("ETag", `"etag"`)
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			w.Header().Set("ETag", `"etag"`)
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", strconv.Itoa(len(object)))
			_, _ = w.Write(object)
		case http.MethodDelete:
			deleteCount++
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			if r.URL.Query().Has("uploads") {
				w.Header().Set("Content-Type", "application/xml")
				_, _ = io.WriteString(w, `<InitiateMultipartUploadResult><Bucket>bucket</Bucket><Key>`+key+`</Key><UploadId>security-test-upload</UploadId></InitiateMultipartUploadResult>`)
				return
			}
			if r.URL.Query().Has("uploadId") {
				w.Header().Set("Content-Type", "application/xml")
				_, _ = io.WriteString(w, `<CompleteMultipartUploadResult><Location>http://localhost/bucket/`+key+`</Location><Bucket>bucket</Bucket><Key>`+key+`</Key><ETag>"etag"</ETag></CompleteMultipartUploadResult>`)
				return
			}
			t.Errorf("unexpected S3 POST %s", r.URL.String())
			w.WriteHeader(http.StatusBadRequest)
		default:
			t.Errorf("unexpected S3 method %s", r.Method)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	client, err := minio.New(strings.TrimPrefix(server.URL, "http://"), &minio.Options{
		Creds: credentials.NewStaticV4("access", "secret", ""), Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	fs := &S3FileSystem{client: client, bucketName: "bucket", disableFileEncryption: true}
	const content = "restored content"
	if err := fs.UnsafeUploadBlobToStorageLocation(context.Background(), nil, strings.NewReader(content), "report.pdf", "tenants/fresh", "storage-id.pdf.gz.age"); err != nil {
		t.Fatal(err)
	}
	if key != "tenants/fresh/storage-id.pdf.gz.age" {
		t.Fatalf("uploaded key %q escaped expected prefix", key)
	}
	reader, err := gzip.NewReader(bytes.NewReader(object))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != content {
		t.Fatalf("decompressed object = %q", decoded)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if deleteCount != 0 {
		t.Fatalf("successful restore upload triggered %d cleanup deletes", deleteCount)
	}
	if err := fs.UnsafeUploadBlobToStorageLocation(context.Background(), nil, strings.NewReader(content), "report.pdf", "tenants/fresh", "../escape.pdf.gz.age"); err == nil {
		t.Fatal("accepted traversal storage filename")
	}
}
