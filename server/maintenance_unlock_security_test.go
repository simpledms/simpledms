package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/encryptor"
)

func TestMaintenanceUnlockRejectsGET(t *testing.T) {
	clearMainIdentity(t)
	handler := newMaintenanceCommandHandler(t, mustEncryptIdentityWithPassphrase(t, "post-only"), nil)
	req := httptest.NewRequest(http.MethodGet, "/-/unlock-cmd", bytes.NewReader(marshalUnlockRequestBody(t, "post-only")))
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)
	if rw.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", rw.Code)
	}
	if rw.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("Allow = %q, want POST", rw.Header().Get("Allow"))
	}
	if encryptor.NilableX25519MainIdentity != nil {
		t.Fatal("GET unlocked maintenance mode")
	}
}

func TestMaintenanceUnlockRejectsOversizedJSONWithoutUnlocking(t *testing.T) {
	clearMainIdentity(t)
	handler := newMaintenanceCommandHandler(t, mustEncryptIdentityWithPassphrase(t, "bounded-post"), nil)
	body := `{"passphrase":"bounded-post","extra":"` + strings.Repeat("x", 64*1024) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/-/unlock-cmd", strings.NewReader(body))
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)
	if rw.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized POST status = %d, want 413", rw.Code)
	}
	if encryptor.NilableX25519MainIdentity != nil {
		t.Fatal("oversized request unlocked maintenance mode")
	}
}

func TestMaintenanceUnlockAcceptsNormalPOST(t *testing.T) {
	clearMainIdentity(t)
	handler := newMaintenanceCommandHandler(t, mustEncryptIdentityWithPassphrase(t, "normal-post"), nil)
	req := httptest.NewRequest(http.MethodPost, "/-/unlock-cmd", bytes.NewReader(marshalUnlockRequestBody(t, "normal-post")))
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		t.Fatalf("valid POST status = %d, want 200: %s", rw.Code, rw.Body.String())
	}
	if encryptor.NilableX25519MainIdentity == nil {
		t.Fatal("valid POST did not unlock maintenance mode")
	}
}
