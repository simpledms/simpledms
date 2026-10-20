package filesystem

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"filippo.io/age"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestS3FileSystemUnsafeVerifyStoredObject(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}

	content := []byte("verified stored object content")
	for _, encrypted := range []bool{false, true} {
		name := "gzip only"
		if encrypted {
			name = "age and gzip"
		}
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Setenv("TMPDIR", tempDir)
			stored := verifiedStoredObjectTestData(t, content, encrypted, identity)
			fileSystemx := newVerifiedStoredObjectTestFileSystem(t, stored, !encrypted)
			var suppliedIdentity *age.X25519Identity
			if encrypted {
				suppliedIdentity = identity
			}
			verified, err := fileSystemx.UnsafeVerifyStoredObject(
				context.Background(), suppliedIdentity, "object",
			)
			if err != nil {
				t.Fatalf("verify stored object: %v", err)
			}
			storageHash := sha256.Sum256(stored)
			crc := crc32.Checksum(stored, crc32.MakeTable(crc32.Castagnoli))
			contentHash := sha256.Sum256(content)
			if verified.StorageSize() != int64(len(stored)) ||
				verified.StorageSHA256() != hex.EncodeToString(storageHash[:]) ||
				verified.StorageCRC32C() != base64.StdEncoding.EncodeToString(
					[]byte{byte(crc >> 24), byte(crc >> 16), byte(crc >> 8), byte(crc)},
				) {
				t.Errorf("unexpected storage metadata: size=%d sha=%q crc=%q",
					verified.StorageSize(), verified.StorageSHA256(), verified.StorageCRC32C())
			}
			if verified.ContentSize() != int64(len(content)) ||
				verified.ContentSHA256() != hex.EncodeToString(contentHash[:]) {
				t.Errorf("unexpected content metadata: size=%d sha=%q",
					verified.ContentSize(), verified.ContentSHA256())
			}
			localPath := verified.file.Name()
			if err := verified.Close(); err != nil {
				t.Fatalf("close snapshot: %v", err)
			}
			if _, err := os.Stat(localPath); !os.IsNotExist(err) {
				t.Fatalf("snapshot still exists after Close: stat error=%v", err)
			}
		})
	}
}

func TestS3FileSystemUnsafeVerifyStoredObjectRejectsCorruption(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	content := []byte("content that must be fully checked")
	validGzip := verifiedStoredObjectTestData(t, content, false, nil)
	badChecksum := append([]byte(nil), validGzip...)
	badChecksum[len(badChecksum)-8] ^= 0xff
	truncatedGzip := append([]byte(nil), validGzip[:len(validGzip)-5]...)
	tamperedAge := encryptCompressedTestObject(t, validGzip, identity)
	tamperedAge[len(tamperedAge)-1] ^= 0xff
	truncatedAge := append([]byte(nil), tamperedAge[:len(tamperedAge)-1]...)

	for _, tc := range []struct {
		name              string
		object            []byte
		disableEncryption bool
	}{
		{
			name:              "gzip checksum",
			object:            badChecksum,
			disableEncryption: true,
		},
		{
			name:              "truncated gzip",
			object:            truncatedGzip,
			disableEncryption: true,
		},
		{
			name:   "encrypted gzip checksum",
			object: encryptCompressedTestObject(t, badChecksum, identity),
		},
		{
			name:   "encrypted truncated gzip",
			object: encryptCompressedTestObject(t, truncatedGzip, identity),
		},
		{
			name:   "tampered age trailer",
			object: tamperedAge,
		},
		{
			name:   "truncated age trailer",
			object: truncatedAge,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			t.Setenv("TMPDIR", tempDir)
			fileSystemx := newVerifiedStoredObjectTestFileSystem(t, tc.object, tc.disableEncryption)
			verified, err := fileSystemx.UnsafeVerifyStoredObject(
				context.Background(), identity, "object",
			)
			if err == nil {
				if verified != nil {
					if err := verified.Close(); err != nil {
						t.Errorf("close snapshot: %v", err)
					}
				}
				t.Fatal("expected corrupt object to fail verification")
			}
			leftovers, err := os.ReadDir(tempDir)
			if err != nil {
				t.Fatalf("list temporary snapshots: %v", err)
			}
			if len(leftovers) != 0 {
				t.Fatalf("failed verification left snapshots: %v", leftovers)
			}
		})
	}
}

func TestS3FileSystemUnsafeVerifyStoredObjectAcceptsEmptyContent(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	stored := verifiedStoredObjectTestData(t, nil, false, nil)
	fileSystemx := newVerifiedStoredObjectTestFileSystem(t, stored, true)
	verified, err := fileSystemx.UnsafeVerifyStoredObject(context.Background(), nil, "object")
	if err != nil {
		t.Fatalf("verify empty content: %v", err)
	}
	localPath := verified.file.Name()
	emptyHash := sha256.Sum256(nil)
	if verified.ContentSize() != 0 || verified.ContentSHA256() != hex.EncodeToString(emptyHash[:]) {
		t.Errorf("unexpected empty content metadata: size=%d sha=%q",
			verified.ContentSize(), verified.ContentSHA256())
	}
	if err := verified.Close(); err != nil {
		t.Fatalf("close snapshot: %v", err)
	}
	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		t.Fatalf("snapshot still exists after Close: stat error=%v", err)
	}
}

func verifiedStoredObjectTestData(
	t *testing.T,
	content []byte,
	encrypted bool,
	identity *age.X25519Identity,
) []byte {
	t.Helper()
	compressed := &bytes.Buffer{}
	writer := gzip.NewWriter(compressed)
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("write gzip content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	if !encrypted {
		return compressed.Bytes()
	}
	return encryptCompressedTestObject(t, compressed.Bytes(), identity)
}

func encryptCompressedTestObject(
	t *testing.T,
	compressed []byte,
	identity *age.X25519Identity,
) []byte {
	t.Helper()
	ciphertext := &bytes.Buffer{}
	writerAge, err := age.Encrypt(ciphertext, identity.Recipient())
	if err != nil {
		t.Fatalf("encrypt gzip content: %v", err)
	}
	if _, err := io.Copy(writerAge, bytes.NewReader(compressed)); err != nil {
		t.Fatalf("write age content: %v", err)
	}
	if err := writerAge.Close(); err != nil {
		t.Fatalf("close age writer: %v", err)
	}
	return ciphertext.Bytes()
}

func newVerifiedStoredObjectTestFileSystem(
	t *testing.T,
	contents []byte,
	disableEncryption bool,
) *S3FileSystem {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			t.Errorf("unexpected S3 request: %s %s", req.Method, req.URL.String())
			rw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		rw.Header().Set("Content-Length", fmt.Sprint(len(contents)))
		rw.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		_, _ = rw.Write(contents)
	}))
	t.Cleanup(server.Close)
	client, err := minio.New(strings.TrimPrefix(server.URL, "http://"), &minio.Options{
		Creds:  credentials.NewStaticV4("access-key", "secret-key", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("create S3 client: %v", err)
	}
	return NewS3FileSystem(client, "bucket", nil, disableEncryption, nil)
}
