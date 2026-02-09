package server

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"filippo.io/age"

	migratemain "github.com/simpledms/simpledms/db/entmain/migrate"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/encryptor"
	"github.com/simpledms/simpledms/i18n"
	"github.com/simpledms/simpledms/ui"
)

func TestMaintenanceRootReturnsServiceUnavailable(t *testing.T) {
	deps := newMaintenanceTestDependencies(t)

	defer func() {
		encryptor.NilableX25519MainIdentity = nil
	}()

	handler := newMaintenanceModeHandler(
		deps.mainDB,
		os.DirFS(t.TempDir()),
		deps.i18n,
		deps.renderer,
		[]byte("irrelevant"),
		false,
		nil,
	)
	encryptor.NilableX25519MainIdentity = nil

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "Maintenance mode") {
		t.Fatalf("expected maintenance mode content, body was %q", rr.Body.String())
	}
}

func TestMaintenanceUnlockCmdInvalidJSONReturnsBadRequest(t *testing.T) {
	deps := newMaintenanceTestDependencies(t)

	defer func() {
		encryptor.NilableX25519MainIdentity = nil
	}()

	handler := newMaintenanceModeHandler(
		deps.mainDB,
		os.DirFS(t.TempDir()),
		deps.i18n,
		deps.renderer,
		[]byte("irrelevant"),
		false,
		nil,
	)
	encryptor.NilableX25519MainIdentity = nil

	req := httptest.NewRequest(http.MethodPost, "/-/unlock-cmd", strings.NewReader("{"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Invalid request payload") {
		t.Fatalf("expected invalid payload message, got %q", rr.Body.String())
	}
}

func TestMaintenanceUnlockCmdEmptyPassphraseReturnsBadRequest(t *testing.T) {
	deps := newMaintenanceTestDependencies(t)

	defer func() {
		encryptor.NilableX25519MainIdentity = nil
	}()

	handler := newMaintenanceModeHandler(
		deps.mainDB,
		os.DirFS(t.TempDir()),
		deps.i18n,
		deps.renderer,
		[]byte("irrelevant"),
		false,
		nil,
	)
	encryptor.NilableX25519MainIdentity = nil

	body := marshalUnlockRequestBody(t, "")
	req := httptest.NewRequest(http.MethodPost, "/-/unlock-cmd", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Passphrase is required") {
		t.Fatalf("expected passphrase required message, got %q", rr.Body.String())
	}
}

func TestMaintenanceUnlockCmdInvalidPassphraseReturnsBadRequest(t *testing.T) {
	deps := newMaintenanceTestDependencies(t)

	defer func() {
		encryptor.NilableX25519MainIdentity = nil
	}()

	encryptedIdentity := mustEncryptIdentityWithPassphrase(t, "correct-passphrase")

	handler := newMaintenanceModeHandler(
		deps.mainDB,
		os.DirFS(t.TempDir()),
		deps.i18n,
		deps.renderer,
		encryptedIdentity,
		false,
		nil,
	)
	encryptor.NilableX25519MainIdentity = nil

	body := marshalUnlockRequestBody(t, "wrong-passphrase")
	req := httptest.NewRequest(http.MethodPost, "/-/unlock-cmd", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Invalid passphrase") {
		t.Fatalf("expected invalid passphrase message, got %q", rr.Body.String())
	}
	if encryptor.NilableX25519MainIdentity != nil {
		t.Fatal("expected identity to stay nil after invalid passphrase")
	}
}

func TestMaintenanceUnlockCmdValidPassphraseSetsIdentityAndCallsShutdown(t *testing.T) {
	deps := newMaintenanceTestDependencies(t)

	defer func() {
		encryptor.NilableX25519MainIdentity = nil
	}()

	encryptedIdentity := mustEncryptIdentityWithPassphrase(t, "correct-passphrase")
	shutdownCalled := false

	handler := newMaintenanceModeHandler(
		deps.mainDB,
		os.DirFS(t.TempDir()),
		deps.i18n,
		deps.renderer,
		encryptedIdentity,
		false,
		func(ctx context.Context) error {
			shutdownCalled = true
			return nil
		},
	)
	encryptor.NilableX25519MainIdentity = nil

	body := marshalUnlockRequestBody(t, "correct-passphrase")
	req := httptest.NewRequest(http.MethodPost, "/-/unlock-cmd", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if encryptor.NilableX25519MainIdentity == nil {
		t.Fatal("expected identity to be set after valid passphrase")
	}
	if !shutdownCalled {
		t.Fatal("expected shutdown function to be called")
	}
}

func marshalUnlockRequestBody(t *testing.T, passphrase string) []byte {
	t.Helper()
	body, err := json.Marshal(struct {
		Passphrase string `json:"passphrase"`
	}{
		Passphrase: passphrase,
	})
	if err != nil {
		t.Fatalf("marshal unlock body: %v", err)
	}
	return body
}

func mustEncryptIdentityWithPassphrase(t *testing.T, passphrase string) []byte {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate x25519 identity: %v", err)
	}

	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		t.Fatalf("create scrypt recipient: %v", err)
	}

	buf := bytes.NewBuffer(nil)
	enc, err := age.Encrypt(buf, recipient)
	if err != nil {
		t.Fatalf("encrypt identity: %v", err)
	}

	if _, err := io.Copy(enc, strings.NewReader(identity.String())); err != nil {
		t.Fatalf("copy identity to encryptor: %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("close encryptor: %v", err)
	}

	return buf.Bytes()
}

type maintenanceTestDependencies struct {
	mainDB   *sqlx.MainDB
	renderer *ui.Renderer
	i18n     *i18n.I18n
}

func newMaintenanceTestDependencies(t *testing.T) *maintenanceTestDependencies {
	t.Helper()

	metaPath := t.TempDir()
	migrationsMainFS, err := migratemain.NewMigrationsMainFS()
	if err != nil {
		t.Fatalf("new migrations fs: %v", err)
	}

	mainDB := dbMigrationsMainDB(true, metaPath, migrationsMainFS)
	t.Cleanup(func() {
		if err := mainDB.Close(); err != nil {
			t.Fatalf("close main db: %v", err)
		}
	})

	_ = initSystemConfig(t, mainDB, true)

	tpl := template.New("app")
	tpl.Funcs(ui.TemplateFuncMap(tpl))
	tpl, err = tpl.ParseFS(ui.WidgetFS, "widget/*.gohtml")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	return &maintenanceTestDependencies{
		mainDB:   mainDB,
		renderer: ui.NewRenderer(tpl),
		i18n:     i18n.NewI18n(),
	}
}
