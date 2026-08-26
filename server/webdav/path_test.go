package webdav

import (
	"net/http/httptest"
	"testing"
)

func TestParseWebDAVPath(t *testing.T) {
	for _, tc := range []struct {
		path string
		ok   bool
		file bool
	}{
		{path: "/webdav/t/s/", ok: true},
		{path: "/webdav/t/s/Inbox/", ok: true},
		{path: "/webdav/t/s/Inbox/scan.pdf", ok: true, file: true},
		{path: "/webdav/t/s/inbox/scan.pdf"},
		{path: "/webdav/t/s/Inbox/nested/scan.pdf"},
		{path: "/webdav/t/s/Inbox/a%2fb.pdf"},
		{path: "/webdav/t/s/Inbox/a%5Cb.pdf"},
	} {
		req := httptest.NewRequest("PUT", tc.path, nil)
		got, ok := parseWebDAVPath(req, "/webdav/t/s")
		if ok != tc.ok || got.isFile != tc.file {
			t.Fatalf("%s: got ok=%v file=%v", tc.path, ok, got.isFile)
		}
	}
}

func TestParseWebDAVDestinationRejectsNetworkPathOtherHost(t *testing.T) {
	req := httptest.NewRequest("MOVE", "/webdav/t/s/Inbox/old.pdf", nil)
	req.Header.Set("Destination", "//other-host/webdav/t/s/Inbox/new.pdf")

	if _, ok := parseWebDAVDestination(req, "/webdav/t/s"); ok {
		t.Fatal("network-path destination for another host was accepted")
	}
}
