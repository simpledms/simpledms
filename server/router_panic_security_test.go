package server

import (
	"log"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogRecoveredPanicRedactsRequestAndPanicSecrets(t *testing.T) {
	const secret = "do-not-log-secret"
	req := httptest.NewRequest("POST", "/upload?token="+secret, nil)
	req.Header.Set("HX-Current-URL", "https://example.test/?token="+secret)
	var output strings.Builder
	previous := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(previous)
	(&Router{}).logRecoveredPanic(req, "panic-"+secret)
	got := output.String()
	if strings.Contains(got, secret) {
		t.Fatalf("panic log exposed request or panic secret: %s", got)
	}
	for _, diagnostic := range []string{"path=\"/upload\"", "panic_type=string"} {
		if !strings.Contains(got, diagnostic) {
			t.Errorf("panic log missing %q: %s", diagnostic, got)
		}
	}
}
