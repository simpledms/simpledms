package temporaryfile

import (
	"testing"
)

func TestUploadFromURLRejectsInternalTargetAddresses(t *testing.T) {
	service := NewUploadFromURLService(nil, false, "", "")

	for _, rawURL := range []string{
		"http://127.0.0.1/file.pdf",
		"http://0.0.0.0/file.pdf",
		"http://0.1.2.3/file.pdf",
		"http://10.0.0.1/file.pdf",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/file.pdf",
		"http://[::ffff:127.0.0.1]/file.pdf",
		"http://[64:ff9b::a00:1]/file.pdf",
		"http://[fd00::1]/file.pdf",
		"http://localhost/file.pdf",
	} {
		if _, err := service.ValidateURLForSource(rawURL, ""); err == nil {
			t.Fatalf("expected %s to be rejected", rawURL)
		}
	}
	if _, err := service.ValidateURLForSource("https://203.0.113.10/file.pdf", ""); err != nil {
		t.Fatalf("expected public address to be allowed: %v", err)
	}
}
