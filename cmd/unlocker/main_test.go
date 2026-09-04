package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendUnlockRequestEscapesPassphrase(t *testing.T) {
	const passphrase = "quote \" slash \\ tab\t newline\n"

	serverx := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Errorf("expected method %q, got %q", http.MethodPost, req.Method)
		}
		if req.URL.Path != "/-/unlock-cmd" {
			t.Errorf("expected unlock path, got %q", req.URL.Path)
		}
		if contentType := req.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected JSON content type, got %q", contentType)
		}

		var body struct {
			Passphrase string `json:"passphrase"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		if body.Passphrase != passphrase {
			t.Errorf("expected passphrase %q, got %q", passphrase, body.Passphrase)
		}
	}))
	t.Cleanup(serverx.Close)

	response, err := sendUnlockRequest(serverx.URL+"/-/unlock-cmd", passphrase)
	if err != nil {
		t.Fatalf("send unlock request: %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
}
