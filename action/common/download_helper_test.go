package common

import (
	"net/http"
	"testing"
)

func TestSetDownloadSecurityHeadersSandboxesEverythingExceptInlinePDFs(t *testing.T) {
	testCases := []struct {
		name        string
		mimeType    string
		isInline    bool
		wantSandbox bool
	}{
		{name: "inline html", mimeType: "text/html; charset=utf-8", isInline: true, wantSandbox: true},
		{name: "inline svg", mimeType: "image/svg+xml", isInline: true, wantSandbox: true},
		{name: "inline xml", mimeType: "text/xml; charset=utf-8", isInline: true, wantSandbox: true},
		{name: "inline image", mimeType: "image/png", isInline: true, wantSandbox: true},
		{name: "inline pdf", mimeType: "Application/PDF", isInline: true, wantSandbox: false},
		{name: "attachment pdf", mimeType: "application/pdf", isInline: false, wantSandbox: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			header := http.Header{}
			setDownloadSecurityHeaders(header, tc.mimeType, tc.isInline)

			if got := header.Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("X-Content-Type-Options = %q", got)
			}
			hasSandbox := header.Get("Content-Security-Policy") == "sandbox"
			if hasSandbox != tc.wantSandbox {
				t.Fatalf("sandbox = %v, want %v", hasSandbox, tc.wantSandbox)
			}
		})
	}
}
