package webdav

import (
	"io"
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/webdav"
)

func TestWebDAVLockSystemLimitsAndNamespaces(t *testing.T) {
	locks := newWebDAVLockSystem()
	first := locks.forCredential("first")
	second := locks.forCredential("second")
	now := time.Now()

	var token string
	for i := range webDAVMaxLocksPerCredential {
		created, err := first.Create(now, webdav.LockDetails{
			Root:     "/Inbox/scan-" + strconv.Itoa(i) + ".pdf",
			Duration: time.Minute,
		})
		if err != nil {
			t.Fatalf("lock %d: %v", i, err)
		}
		token = created
	}
	if _, err := first.Create(now, webdav.LockDetails{Root: "/Inbox/extra.pdf"}); err == nil {
		t.Fatal("expected excess lock to fail")
	}
	if err := second.Unlock(now, token); err == nil {
		t.Fatal("other credential unlocked namespaced token")
	}
	if err := first.Unlock(now, token); err != nil {
		t.Fatalf("owner unlock: %v", err)
	}
}

func TestWebDAVUploadFailureUnblocksWriterAndReportsOnce(t *testing.T) {
	pipeReader, pipeWriter := io.Pipe()
	uploadFile := &webDAVUploadFile{
		pipeReader: pipeReader,
		pipeWriter: pipeWriter,
		done:       make(chan error, 1),
	}

	writeDone := make(chan error, 1)
	go func() {
		_, err := pipeWriter.Write([]byte("x"))
		writeDone <- err
	}()

	go uploadFile.upload()

	select {
	case err := <-writeDone:
		if err == nil {
			t.Fatal("writer succeeded after upload failure")
		}
	case <-time.After(time.Second):
		t.Fatal("writer stayed blocked after upload failure")
	}

	select {
	case err := <-uploadFile.done:
		if err == nil {
			t.Fatal("upload failure was reported as success")
		}
	case <-time.After(time.Second):
		t.Fatal("upload failure was not reported")
	}
	select {
	case err := <-uploadFile.done:
		t.Fatalf("upload reported more than once: %v", err)
	default:
	}
}
