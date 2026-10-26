package server

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type blockingStatusWriter struct {
	*httptest.ResponseRecorder
	status      int
	seen        chan struct{}
	releaseChan chan struct{}
	releaseOnce sync.Once
}

func newBlockingStatusWriter(status int) *blockingStatusWriter {
	return &blockingStatusWriter{
		ResponseRecorder: httptest.NewRecorder(),
		status:           status,
		seen:             make(chan struct{}),
		releaseChan:      make(chan struct{}),
	}
}

func (writer *blockingStatusWriter) WriteHeader(status int) {
	if status == writer.status {
		close(writer.seen)
		<-writer.releaseChan
	}
	writer.ResponseRecorder.WriteHeader(status)
}

func (writer *blockingStatusWriter) wait(t *testing.T) {
	t.Helper()
	select {
	case <-writer.seen:
	case <-time.After(time.Second):
		t.Fatalf("handler did not write status %d", writer.status)
	}
}

func (writer *blockingStatusWriter) release() {
	writer.releaseOnce.Do(func() { close(writer.releaseChan) })
}

func (writer *blockingStatusWriter) cleanup(t *testing.T, done <-chan struct{}) {
	t.Helper()
	t.Cleanup(func() {
		writer.release()
		<-done
	})
}

var _ http.ResponseWriter = (*blockingStatusWriter)(nil)
