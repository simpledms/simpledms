package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"filippo.io/age"

	ui2 "github.com/simpledms/simpledms/core/ui"
	migratemain "github.com/simpledms/simpledms/db/entmain/migrate"
	"github.com/simpledms/simpledms/db/entmain/systemconfig"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/encryptor"
	"github.com/simpledms/simpledms/i18n"
	systemconfigmodel "github.com/simpledms/simpledms/model/main/systemconfig"
	"github.com/simpledms/simpledms/ui"
	"github.com/simpledms/simpledms/ui/uix/partial"
)

func TestInitializeMainConfigOverrideDBConfigDoesNotReadEncryptedFieldsBeforeUnlock(t *testing.T) {
	deps := newMaintenanceTestDependencies(t)

	t.Setenv("SIMPLEDMS_INITIAL_ACCOUNT_EMAIL", "")
	t.Setenv("SIMPLEDMS_INITIAL_TENANT_NAME", "")
	t.Setenv("SIMPLEDMS_TLS_ENABLE_AUTOCERT", "false")
	t.Setenv("SIMPLEDMS_TLS_CERT_FILEPATH", "/tmp/simpledms-cert.pem")
	t.Setenv("SIMPLEDMS_TLS_PRIVATE_KEY_FILEPATH", "/tmp/simpledms-key.pem")
	t.Setenv("SIMPLEDMS_TLS_AUTOCERT_EMAIL", "")
	t.Setenv("SIMPLEDMS_TLS_AUTOCERT_HOSTS", "")

	clearMainIdentity(t)

	serverx := &Server{}
	serverx.initializeMainConfig(context.Background(), deps.mainDB, true)

	configx := deps.mainDB.ReadOnlyConn.SystemConfig.Query().
		Select(
			systemconfig.FieldTLSCertFilepath,
			systemconfig.FieldTLSPrivateKeyFilepath,
		).
		FirstX(context.Background())

	if configx.TLSCertFilepath != "/tmp/simpledms-cert.pem" {
		t.Fatalf("expected TLS cert path to be overridden, got %q", configx.TLSCertFilepath)
	}
	if configx.TLSPrivateKeyFilepath != "/tmp/simpledms-key.pem" {
		t.Fatalf("expected TLS key path to be overridden, got %q", configx.TLSPrivateKeyFilepath)
	}
}

func TestMaintenancePageUnlockFormRendering(t *testing.T) {
	const passphrase = "render-only-sentinel-passphrase-8f27"
	deps := newMaintenanceTestDependencies(t)
	clearMainIdentity(t)
	var stopCalls atomic.Int32
	handler := newMaintenanceModeHandler(
		deps.mainDB,
		os.DirFS(t.TempDir()),
		false,
		deps.i18n,
		deps.renderer,
		[]byte(passphrase),
		nil,
		false,
		func() {
			stopCalls.Add(1)
		},
	)

	var logs bytes.Buffer
	previousLogWriter := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() {
		log.SetOutput(previousLogWriter)
	})

	testCases := []struct {
		name          string
		url           string
		isFormVisible bool
	}{
		{name: "root absent", url: "/"},
		{name: "deep path absent", url: "/spaces/example?view=list"},
		{name: "root presence only", url: "/?unlock", isFormVisible: true},
		{name: "root empty", url: "/?unlock=", isFormVisible: true},
		{name: "deep path false", url: "/spaces/example?unlock=false", isFormVisible: true},
		{
			name:          "deep path arbitrary",
			url:           "/spaces/example?view=list&unlock=show-form",
			isFormVisible: true,
		},
		{
			name:          "deep path duplicate",
			url:           "/spaces/example?unlock=false&view=list&unlock=true",
			isFormVisible: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, testCase.url, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
			}
			if value := rr.Header().Get("X-SimpleDMS-Maintenance"); value != "true" {
				t.Fatalf("expected maintenance marker, got %q", value)
			}
			assertMaintenanceCommandCacheHeaders(t, rr)
			body := rr.Body.String()
			if !strings.Contains(body, "Maintenance mode") ||
				!strings.Contains(body, "Maintenance mode is enabled.") {
				t.Fatalf("expected ordinary maintenance content, body was %q", body)
			}

			assertMaintenanceFormVisibility(t, rr, testCase.isFormVisible)
			assertStringAbsent(t, "request URL", req.URL.String(), passphrase)
			assertStringAbsent(t, "response body", body, passphrase)
			for name, values := range rr.Header() {
				for _, value := range values {
					assertStringAbsent(t, "response header "+name, value, passphrase)
				}
			}
		})
	}

	if encryptor.NilableX25519MainIdentity != nil || stopCalls.Load() != 0 {
		t.Fatal("expected reveal requests not to transition maintenance mode")
	}
	assertStringAbsent(t, "logs", logs.String(), passphrase)
}

func TestMaintenanceUnlockFormTranslations(t *testing.T) {
	deps := newMaintenanceTestDependencies(t)
	handler := newMaintenanceModeHandler(
		deps.mainDB,
		os.DirFS(t.TempDir()),
		false,
		deps.i18n,
		deps.renderer,
		[]byte("irrelevant"),
		nil,
		false,
		nil,
	)

	testCases := []struct {
		language string
		texts    []string
	}{
		{
			language: "en",
			texts: []string{
				"Application passphrase",
				"Unlock app",
				"Passphrase is required.",
				"Invalid passphrase.",
				"Something went wrong. Please try again.",
				"Too many unlock attempts. Please try again later.",
				"Application unlocked. Starting up.",
			},
		},
		{
			language: "de",
			texts: []string{
				"Anwendungspassphrase",
				"App entsperren",
				"Passphrase ist erforderlich.",
				"Ungültige Passphrase.",
				"Ein Fehler ist aufgetreten. Bitte versuche es erneut.",
				"Zu viele Entsperrversuche. Bitte versuche es später erneut.",
				"Anwendung entsperrt. Start wird fortgesetzt.",
			},
		},
		{
			language: "fr",
			texts: []string{
				"Phrase secrète de l’application",
				"Déverrouiller l'application",
				"La phrase secrète est requise.",
				"Phrase secrète invalide.",
				"Une erreur s'est produite. Veuillez réessayer.",
				"Trop de tentatives de déverrouillage. Veuillez réessayer plus tard.",
				"Application déverrouillée. Démarrage en cours.",
			},
		},
		{
			language: "it",
			texts: []string{
				"Passphrase dell’applicazione",
				"Sblocca app",
				"La passphrase è obbligatoria.",
				"Passphrase non valida.",
				"Qualcosa è andato storto. Riprova per favore.",
				"Troppi tentativi di sblocco. Riprova più tardi.",
				"Applicazione sbloccata. Avvio in corso.",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.language, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/deep/path?unlock=false", nil)
			req.Header.Set("Accept-Language", testCase.language)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
			}
			body := html.UnescapeString(rr.Body.String())
			for _, text := range testCase.texts {
				if !strings.Contains(body, text) {
					t.Errorf("expected %q translation in response", text)
				}
			}
		})
	}
}

func TestMaintenanceStartupStatusTranslations(t *testing.T) {
	deps := newMaintenanceTestDependencies(t)
	for _, locale := range []struct {
		language  string
		unlocking string
		starting  string
	}{
		{
			language:  "en",
			unlocking: "Unlocking application. Please wait.",
			starting:  "Application unlocked. Starting up.",
		},
		{
			language:  "de",
			unlocking: "Anwendung wird entsperrt. Bitte warte.",
			starting:  "Anwendung entsperrt. Start wird fortgesetzt.",
		},
		{
			language:  "fr",
			unlocking: "Déverrouillage de l’application en cours. Veuillez patienter.",
			starting:  "Application déverrouillée. Démarrage en cours.",
		},
		{
			language:  "it",
			unlocking: "Sblocco dell’applicazione in corso. Attendi.",
			starting:  "Applicazione sbloccata. Avvio in corso.",
		},
	} {
		for _, phase := range []struct {
			name    string
			state   int32
			message string
		}{
			{name: "unlocking", state: maintenanceStateUnlocking, message: locale.unlocking},
			{name: "starting", state: maintenanceStateStarting, message: locale.starting},
		} {
			t.Run(locale.language+"/"+phase.name, func(t *testing.T) {
				handler := newMaintenancePageHandler(
					deps.mainDB, deps.i18n, deps.renderer, false,
					func() int32 { return phase.state },
				)
				req := httptest.NewRequest(http.MethodGet, "/?unlock", nil)
				req.Header.Set("Accept-Language", locale.language)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				if rr.Code != http.StatusServiceUnavailable {
					t.Fatalf("expected maintenance status, got %d", rr.Code)
				}
				assertMaintenanceCommandCacheHeaders(t, rr)
				assertMaintenanceFormVisibility(t, rr, false)
				if !strings.Contains(html.UnescapeString(rr.Body.String()), phase.message) {
					t.Errorf("expected visible status %q", phase.message)
				}
			})
		}
	}
}

func assertMaintenanceFormVisibility(
	t *testing.T,
	rr *httptest.ResponseRecorder,
	isVisible bool,
) {
	t.Helper()
	body := rr.Body.String()
	refresh := `<meta http-equiv="refresh" content="60" />`
	formAction := `action="/-/unlock-cmd"`
	if !isVisible {
		if !strings.Contains(body, refresh) {
			t.Error("expected ordinary maintenance refresh metadata")
		}
		if strings.Contains(body, formAction) || strings.Contains(body, `type="password"`) {
			t.Error("expected ordinary maintenance page without unlock form")
		}
		return
	}

	if strings.Contains(body, refresh) {
		t.Error("expected revealed form without refresh metadata")
	}
	assertMaintenanceCommandCacheHeaders(t, rr)
	for _, markup := range []string{
		`method="post"`,
		formAction,
		`hx-boost="false"`,
		`type="password"`,
		`required`,
		`role="status"`,
		`role="alert"`,
		`document.currentScript.previousElementSibling`,
		`fetch("/-/unlock-cmd"`,
		`mode: "same-origin"`,
		`target.searchParams.delete("unlock")`,
		`window.location.replace(target.href)`,
		`response.headers.get("X-SimpleDMS-Maintenance")`,
	} {
		if !strings.Contains(body, markup) {
			t.Errorf("expected form markup %q", markup)
		}
	}
	if strings.Contains(body, "response.text()") || strings.Contains(body, "response.json()") {
		t.Error("expected form not to display command response internals")
	}
	if regexp.MustCompile(`role="(?:status|alert)"[^>]*\shidden`).MatchString(body) {
		t.Error("expected live regions to remain in the accessibility tree")
	}

	inputIDMatch := regexp.MustCompile(`id="([^"]+-passphrase)"`).FindStringSubmatch(body)
	if len(inputIDMatch) != 2 {
		t.Fatal("expected passphrase input id")
	}
	inputID := inputIDMatch[1]
	formID := strings.TrimSuffix(inputID, "-passphrase")
	for _, relationship := range []string{
		`for="` + inputID + `"`,
		`aria-describedby="` + formID + `-status ` + formID + `-error"`,
		`id="` + formID + `-error" role="alert"`,
		`id="` + formID + `-status" role="status"`,
	} {
		if !strings.Contains(body, relationship) {
			t.Errorf("expected accessible relationship %q", relationship)
		}
	}
}

func assertStringAbsent(t *testing.T, surface, value, secret string) {
	t.Helper()
	if strings.Contains(value, secret) {
		t.Errorf("%s exposed sentinel passphrase", surface)
	}
}

func TestMaintenanceUnlockCmdInvalidRequestsDoNotTransition(t *testing.T) {
	const passphrase = "correct-passphrase"
	clearMainIdentity(t)
	encryptedIdentity := mustEncryptIdentityWithPassphrase(t, passphrase)

	testCases := []struct {
		name              string
		body              []byte
		encryptedIdentity []byte
		expectedBody      string
	}{
		{
			name:              "malformed JSON",
			body:              []byte("{"),
			encryptedIdentity: encryptedIdentity,
			expectedBody:      "Invalid request payload",
		},
		{
			name:              "empty passphrase",
			body:              marshalUnlockRequestBody(t, ""),
			encryptedIdentity: encryptedIdentity,
			expectedBody:      "Passphrase is required",
		},
		{
			name:              "wrong passphrase",
			body:              marshalUnlockRequestBody(t, "wrong-passphrase"),
			encryptedIdentity: encryptedIdentity,
			expectedBody:      "Invalid passphrase",
		},
		{
			name: "malformed decrypted identity",
			body: marshalUnlockRequestBody(t, passphrase),
			encryptedIdentity: mustEncryptPlaintextWithPassphrase(
				t,
				passphrase,
				"not an X25519 identity",
			),
			expectedBody: "Invalid passphrase",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var stopCalls atomic.Int32
			handler := newMaintenanceCommandHandler(t, testCase.encryptedIdentity, func() {
				stopCalls.Add(1)
			})
			req := httptest.NewRequest(
				http.MethodPost,
				"/-/unlock-cmd",
				bytes.NewReader(testCase.body),
			)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assertMaintenanceCommandCacheHeaders(t, rr)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
			}
			if rr.Body.String() != testCase.expectedBody {
				t.Fatalf("expected body %q, got %q", testCase.expectedBody, rr.Body.String())
			}
			if encryptor.NilableX25519MainIdentity != nil || stopCalls.Load() != 0 {
				t.Fatal("expected invalid request not to transition maintenance mode")
			}
		})
	}
}

func TestMaintenanceUnlockCmdRateLimitsAttempts(t *testing.T) {
	const passphrase = "rate-limited-passphrase"
	clearMainIdentity(t)
	var stopCalls atomic.Int32
	handler := newMaintenanceCommandHandler(
		t,
		mustEncryptIdentityWithPassphrase(t, passphrase),
		func() {
			stopCalls.Add(1)
		},
	)
	for attempt := range 10 {
		body := []byte("{")
		if attempt%2 == 0 {
			body = marshalUnlockRequestBody(t, "")
		}
		req := httptest.NewRequest(http.MethodPost, "/-/unlock-cmd", bytes.NewReader(body))
		req.RemoteAddr = "192.0.2.10:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("invalid request %d: expected status %d, got %d", attempt, http.StatusBadRequest, rr.Code)
		}
	}

	for attempt := range 5 {
		req := httptest.NewRequest(
			http.MethodPost,
			"/-/unlock-cmd",
			bytes.NewReader(marshalUnlockRequestBody(t, fmt.Sprintf("wrong-%d", attempt))),
		)
		req.RemoteAddr = "192.0.2.10:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("attempt %d: expected status %d, got %d", attempt, http.StatusBadRequest, rr.Code)
		}
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/-/unlock-cmd",
		bytes.NewReader(marshalUnlockRequestBody(t, passphrase)),
	)
	req.RemoteAddr = "192.0.2.10:5678"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assertMaintenanceCommandCacheHeaders(t, rr)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
	if rr.Body.String() != "Too many unlock attempts. Please try again shortly." {
		t.Fatalf("unexpected rate-limit body %q", rr.Body.String())
	}
	if encryptor.NilableX25519MainIdentity != nil || stopCalls.Load() != 0 {
		t.Fatal("expected rate-limited request not to transition maintenance mode")
	}
}

func TestMaintenanceUnlockCmdConcurrentValidRequestsTransitionOnce(t *testing.T) {
	const passphrase = "concurrent-unlock-passphrase"
	clearMainIdentity(t)
	encryptedIdentity := mustEncryptIdentityWithPassphrase(t, passphrase)
	expectedIdentity, err := systemconfigmodel.DecryptMainIdentity(encryptedIdentity, passphrase)
	if err != nil {
		t.Fatalf("decrypt expected identity: %v", err)
	}

	synctest.Test(t, func(t *testing.T) {
		var stopCalls atomic.Int32
		handler := newMaintenanceCommandHandler(t, encryptedIdentity, func() {
			time.Sleep(time.Hour)
			stopCalls.Add(1)
		})

		const requestCount = 4
		body := marshalUnlockRequestBody(t, passphrase)
		start := make(chan struct{})
		responses := make([]*httptest.ResponseRecorder, requestCount)
		var requests sync.WaitGroup
		for index := range requestCount {
			requests.Go(func() {
				<-start
				req := httptest.NewRequest(
					http.MethodPost,
					"/-/unlock-cmd",
					bytes.NewReader(body),
				)
				responses[index] = httptest.NewRecorder()
				handler.ServeHTTP(responses[index], req)
			})
		}
		close(start)
		requests.Wait()
		successes, conflicts := 0, 0
		for _, response := range responses {
			switch response.Code {
			case http.StatusOK:
				successes++
			case http.StatusConflict:
				conflicts++
				assertMaintenanceCommandCacheHeaders(t, response)
			default:
				t.Errorf("expected status 200 or 409, got %d", response.Code)
			}
			if response.Body.Len() != 0 {
				t.Errorf("expected empty response, got %q", response.Body.String())
			}
		}
		if successes != 1 || conflicts != requestCount-1 {
			t.Fatalf(
				"expected one success and %d conflicts, got %d successes and %d conflicts",
				requestCount-1, successes, conflicts,
			)
		}

		time.Sleep(time.Hour)
		synctest.Wait()
		if stopCalls.Load() != 1 {
			t.Fatalf("expected one stop sequence, got %d", stopCalls.Load())
		}
	})

	if encryptor.NilableX25519MainIdentity == nil {
		t.Fatal("expected identity to be loaded")
	}
	if got := encryptor.NilableX25519MainIdentity.String(); got != expectedIdentity.String() {
		t.Fatalf("expected stable identity %q, got %q", expectedIdentity.String(), got)
	}
}

func TestMaintenanceUnlockInFlightHidesFormAndRejectsDuplicateRequests(t *testing.T) {
	const passphrase = "in-flight-unlock-passphrase"
	deps := newMaintenanceTestDependencies(t)
	clearMainIdentity(t)
	var stopCalls atomic.Int32
	callback := make(chan struct{}, 1)
	handler := newMaintenanceModeHandler(
		deps.mainDB,
		os.DirFS(t.TempDir()),
		false,
		deps.i18n,
		deps.renderer,
		mustEncryptIdentityWithPassphrase(t, passphrase),
		nil,
		false,
		func() {
			stopCalls.Add(1)
			callback <- struct{}{}
		},
	)
	assertMaintenance := func(isFormVisible bool, statusMessage string) {
		t.Helper()
		for _, path := range []string{
			"/",
			"/?unlock=anything",
			"/spaces/example?view=list",
			"/spaces/example?view=list&unlock=false",
		} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusServiceUnavailable ||
				rr.Header().Get("X-SimpleDMS-Maintenance") != "true" {
				t.Fatalf("GET %s: expected marked maintenance response, got %d", path, rr.Code)
			}
			assertMaintenanceCommandCacheHeaders(t, rr)
			assertMaintenanceFormVisibility(t, rr, isFormVisible && req.URL.Query().Has("unlock"))
			if !strings.Contains(rr.Body.String(), statusMessage) {
				t.Errorf("GET %s: expected status message %q", path, statusMessage)
			}
		}
	}
	const lockedMessage = "Maintenance mode is enabled. Please wait until the app is ready again."

	// An incomplete body has not reached the decryption guard, so it must not
	// prevent another client from seeing the initial unlock form.
	reader, writer := io.Pipe()
	incompleteRequest := httptest.NewRequest(http.MethodPost, "/-/unlock-cmd", reader)
	incompleteDone := make(chan struct{})
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Errorf("close incomplete unlock body: %v", err)
		}
		<-incompleteDone
	})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), incompleteRequest)
		close(incompleteDone)
	}()
	if _, err := writer.Write([]byte("{")); err != nil {
		t.Fatalf("write partial unlock body: %v", err)
	}
	assertMaintenance(true, lockedMessage)
	if err := writer.Close(); err != nil {
		t.Fatalf("close incomplete unlock body: %v", err)
	}
	select {
	case <-incompleteDone:
	case <-time.After(time.Second):
		t.Fatal("incomplete unlock request did not finish")
	}

	failedWriter := newBlockingStatusWriter(http.StatusBadRequest)
	failedRequest := httptest.NewRequest(
		http.MethodPost,
		"/-/unlock-cmd",
		bytes.NewReader(marshalUnlockRequestBody(t, "wrong-passphrase")),
	)
	failedDone := make(chan struct{})
	failedWriter.cleanup(t, failedDone)
	go func() {
		handler.ServeHTTP(failedWriter, failedRequest)
		close(failedDone)
	}()
	failedWriter.wait(t)
	assertMaintenance(false, "Unlocking application. Please wait.")
	failedWriter.release()
	select {
	case <-failedDone:
	case <-time.After(time.Second):
		t.Fatal("failed unlock request did not finish")
	}
	assertMaintenance(true, lockedMessage)
	if encryptor.NilableX25519MainIdentity != nil || stopCalls.Load() != 0 {
		t.Fatal("failed unlock must not assign identity or invoke callback")
	}

	// Pause at the success response after decryption but before the callback and
	// handoff, which is the public HTTP seam for the in-flight startup state.
	successWriter := newBlockingStatusWriter(http.StatusOK)
	successRequest := httptest.NewRequest(
		http.MethodPost,
		"/-/unlock-cmd",
		bytes.NewReader(marshalUnlockRequestBody(t, passphrase)),
	)
	successDone := make(chan struct{})
	successWriter.cleanup(t, successDone)
	go func() {
		handler.ServeHTTP(successWriter, successRequest)
		close(successDone)
	}()
	successWriter.wait(t)
	assertMaintenance(false, "Application unlocked. Starting up.")
	identityAfterUnlock := encryptor.NilableX25519MainIdentity
	if identityAfterUnlock == nil {
		t.Fatal("expected decrypted identity")
	}

	for _, body := range [][]byte{
		marshalUnlockRequestBody(t, passphrase),
		marshalUnlockRequestBody(t, "wrong-passphrase"),
		[]byte("{"),
	} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(
			http.MethodPost, "/-/unlock-cmd", bytes.NewReader(body),
		))
		if rr.Code != http.StatusConflict {
			t.Fatalf("duplicate unlock: expected status %d, got %d", http.StatusConflict, rr.Code)
		}
		assertMaintenanceCommandCacheHeaders(t, rr)
	}
	if stopCalls.Load() != 0 || encryptor.NilableX25519MainIdentity != identityAfterUnlock {
		t.Fatal("duplicate attempts must not replace identity or invoke callback")
	}
	successWriter.release()
	select {
	case <-successDone:
	case <-time.After(time.Second):
		t.Fatal("unlock request did not finish")
	}
	if successWriter.ResponseRecorder.Code != http.StatusOK {
		t.Fatalf(
			"expected successful unlock, got %d: %s",
			successWriter.ResponseRecorder.Code,
			successWriter.ResponseRecorder.Body.String(),
		)
	}
	select {
	case <-callback:
	case <-time.After(time.Second):
		t.Fatal("unlock callback did not run")
	}
	if stopCalls.Load() != 1 {
		t.Fatalf("expected one callback, got %d", stopCalls.Load())
	}
	if encryptor.NilableX25519MainIdentity != identityAfterUnlock {
		t.Fatal("duplicate attempts replaced the unlocked identity")
	}
	assertMaintenance(false, "Application unlocked. Starting up.")
	for _, body := range [][]byte{
		marshalUnlockRequestBody(t, passphrase),
		marshalUnlockRequestBody(t, "wrong-passphrase"),
		[]byte("{"),
	} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(
			http.MethodPost, "/-/unlock-cmd", bytes.NewReader(body),
		))
		if rr.Code != http.StatusConflict {
			t.Fatalf("post-success duplicate: expected status %d, got %d", http.StatusConflict, rr.Code)
		}
		assertMaintenanceCommandCacheHeaders(t, rr)
	}
	if stopCalls.Load() != 1 || encryptor.NilableX25519MainIdentity != identityAfterUnlock {
		t.Fatal("post-success duplicate attempts must not replace identity or repeat callback")
	}
}

func TestMaintenanceUnlockCmdRejectsCrossOriginAndAllowsCLI(t *testing.T) {
	const passphrase = "cli-compatible-passphrase"
	clearMainIdentity(t)

	stopStarted := make(chan struct{}, 1)
	handler := newMaintenanceCommandHandler(
		t,
		mustEncryptIdentityWithPassphrase(t, passphrase),
		func() {
			stopStarted <- struct{}{}
		},
	)

	crossOriginReq := httptest.NewRequest(
		http.MethodPost,
		"http://simpledms.example/-/unlock-cmd",
		bytes.NewReader(marshalUnlockRequestBody(t, passphrase)),
	)
	crossOriginReq.Header.Set("Origin", "https://attacker.example")
	crossOriginReq.Header.Set("Sec-Fetch-Site", "cross-site")
	crossOriginRR := httptest.NewRecorder()
	handler.ServeHTTP(crossOriginRR, crossOriginReq)

	assertMaintenanceCommandCacheHeaders(t, crossOriginRR)
	if crossOriginRR.Code != http.StatusForbidden {
		t.Fatalf("expected cross-origin status %d, got %d", http.StatusForbidden, crossOriginRR.Code)
	}
	if encryptor.NilableX25519MainIdentity != nil {
		t.Fatal("expected cross-origin request not to load the identity")
	}
	select {
	case <-stopStarted:
		t.Fatal("expected cross-origin request not to stop maintenance mode")
	default:
	}

	cliReq := httptest.NewRequest(
		http.MethodPost,
		"http://simpledms.example/-/unlock-cmd",
		bytes.NewReader(marshalUnlockRequestBody(t, passphrase)),
	)
	cliRR := httptest.NewRecorder()
	handler.ServeHTTP(cliRR, cliReq)

	assertMaintenanceCommandCacheHeaders(t, cliRR)
	if cliRR.Code != http.StatusOK {
		t.Fatalf("expected CLI-shaped status %d, got %d", http.StatusOK, cliRR.Code)
	}
	if cliRR.Body.Len() != 0 {
		t.Fatalf("expected existing empty success response, got %q", cliRR.Body.String())
	}
	select {
	case <-stopStarted:
	case <-time.After(time.Second):
		t.Fatal("expected CLI-shaped request to stop maintenance mode")
	}
}

func TestMaintenanceUnlockCmdDoesNotExposePassphrase(t *testing.T) {
	const passphrase = "sentinel-passphrase-8cd6979d"
	clearMainIdentity(t)

	var logs bytes.Buffer
	previousLogWriter := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() {
		log.SetOutput(previousLogWriter)
	})

	var stopCalls atomic.Int32
	handler := newMaintenanceCommandHandler(
		t,
		mustEncryptIdentityWithPassphrase(t, "different-passphrase"),
		func() {
			stopCalls.Add(1)
		},
	)
	req := httptest.NewRequest(
		http.MethodPost,
		"http://simpledms.example/-/unlock-cmd",
		bytes.NewReader(marshalUnlockRequestBody(t, passphrase)),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assertMaintenanceCommandCacheHeaders(t, rr)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
	if stopCalls.Load() != 0 {
		t.Fatalf("expected no stop calls, got %d", stopCalls.Load())
	}
	if strings.Contains(req.URL.String(), passphrase) {
		t.Fatal("request URL exposed passphrase")
	}
	if strings.Contains(rr.Body.String(), passphrase) {
		t.Fatal("response body exposed passphrase")
	}
	for name, values := range rr.Header() {
		if strings.Contains(name, passphrase) || strings.Contains(strings.Join(values, ","), passphrase) {
			t.Fatalf("response header %q exposed passphrase", name)
		}
	}
	if strings.Contains(logs.String(), passphrase) {
		t.Fatal("logs exposed passphrase")
	}
}

func TestMaintenanceUnlockCmdPreservesPersistedPassphraseProtection(t *testing.T) {
	const passphrase = "persisted-protection-passphrase"
	deps := newMaintenanceTestDependencies(t)

	encryptedIdentity := mustEncryptIdentityWithPassphrase(t, passphrase)
	ctx := context.Background()
	systemConfigID := deps.mainDB.ReadWriteConn.SystemConfig.Query().FirstIDX(ctx)
	deps.mainDB.ReadWriteConn.SystemConfig.UpdateOneID(systemConfigID).
		SetX25519Identity(encryptedIdentity).
		SetIsIdentityEncryptedWithPassphrase(true).
		ExecX(ctx)
	before := deps.mainDB.ReadOnlyConn.SystemConfig.Query().
		Select(
			systemconfig.FieldX25519Identity,
			systemconfig.FieldIsIdentityEncryptedWithPassphrase,
		).
		FirstX(ctx)
	beforeIdentity := bytes.Clone(before.X25519Identity)
	clearMainIdentity(t)

	stopStarted := make(chan struct{}, 1)
	handler := newMaintenanceCommandHandler(t, encryptedIdentity, func() {
		stopStarted <- struct{}{}
	})
	req := httptest.NewRequest(
		http.MethodPost,
		"/-/unlock-cmd",
		bytes.NewReader(marshalUnlockRequestBody(t, passphrase)),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	select {
	case <-stopStarted:
	case <-time.After(time.Second):
		t.Fatal("expected stop sequence to start")
	}

	after := deps.mainDB.ReadOnlyConn.SystemConfig.Query().
		Select(
			systemconfig.FieldX25519Identity,
			systemconfig.FieldIsIdentityEncryptedWithPassphrase,
		).
		FirstX(ctx)
	if !bytes.Equal(after.X25519Identity, beforeIdentity) {
		t.Fatal("expected persisted encrypted identity bytes to remain unchanged")
	}
	if after.IsIdentityEncryptedWithPassphrase != before.IsIdentityEncryptedWithPassphrase {
		t.Fatal("expected persisted passphrase-protection flag to remain unchanged")
	}
	if encryptor.NilableX25519MainIdentity == nil {
		t.Fatal("expected runtime identity to be loaded")
	}
}

func TestStopMaintenanceModeServerForcesCloseAfterDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	requestStarted := make(chan struct{})
	requestFinished := make(chan struct{})
	serverx := &http.Server{
		Handler: http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			close(requestStarted)
			<-req.Context().Done()
			close(requestFinished)
		}),
	}
	t.Cleanup(func() {
		_ = serverx.Close()
	})
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- serverx.Serve(listener)
	}()

	transport := &http.Transport{DisableKeepAlives: true}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}
	requestDone := make(chan error, 1)
	go func() {
		response, err := client.Get("http://" + listener.Addr().String())
		if err == nil {
			_ = response.Body.Close()
		}
		requestDone <- err
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("blocking request did not start")
	}

	var logs bytes.Buffer
	previousLogWriter := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() {
		log.SetOutput(previousLogWriter)
	})

	// This intentionally uses the production timeout; shortening it would require a test-only seam.
	stopDone := make(chan struct{})
	go func() {
		stopMaintenanceModeServer(serverx)
		close(stopDone)
	}()
	select {
	case <-stopDone:
	case <-time.After(12 * time.Second):
		t.Fatal("maintenance stop did not reach its deadline")
	}

	select {
	case <-requestFinished:
	case <-time.After(time.Second):
		t.Fatal("forced close did not cancel the active request")
	}
	select {
	case err := <-requestDone:
		if err == nil {
			t.Fatal("expected forced close to terminate the client connection")
		}
	case <-time.After(time.Second):
		t.Fatal("forced close did not terminate the client connection")
	}
	select {
	case err := <-serveDone:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("expected server closed, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("maintenance server did not stop")
	}
	if !strings.Contains(logs.String(), context.DeadlineExceeded.Error()) {
		t.Fatalf("expected shutdown deadline log, got %q", logs.String())
	}
}

func assertMaintenanceCommandCacheHeaders(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	if value := rr.Header().Get("Cache-Control"); value != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", value)
	}
	if value := rr.Header().Get("Pragma"); value != "no-cache" {
		t.Fatalf("expected Pragma no-cache, got %q", value)
	}
}

func clearMainIdentity(t *testing.T) {
	t.Helper()
	previousIdentity := encryptor.NilableX25519MainIdentity
	encryptor.NilableX25519MainIdentity = nil
	t.Cleanup(func() {
		encryptor.NilableX25519MainIdentity = previousIdentity
	})
}

func newMaintenanceCommandHandler(
	t *testing.T,
	encryptedIdentity []byte,
	stopFn func(),
) http.Handler {
	t.Helper()
	return newMaintenanceModeHandler(
		nil,
		os.DirFS(t.TempDir()),
		false,
		nil,
		nil,
		encryptedIdentity,
		nil,
		false,
		stopFn,
	)
}

func writeTestCertificate(t *testing.T) (string, string) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate certificate key: %v", err)
	}
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("generate certificate serial number: %v", err)
	}
	now := time.Now()
	templatex := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "127.0.0.1",
		},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		templatex,
		templatex,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	privateKeyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal certificate key: %v", err)
	}

	certFile := filepath.Join(t.TempDir(), "certificate.pem")
	keyFile := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificateDER,
	}), 0600); err != nil {
		t.Fatalf("write certificate: %v", err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyDER,
	}), 0600); err != nil {
		t.Fatalf("write certificate key: %v", err)
	}

	return certFile, keyFile
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
	return mustEncryptPlaintextWithPassphrase(t, passphrase, identity.String())
}

func mustEncryptPlaintextWithPassphrase(t *testing.T, passphrase, plaintext string) []byte {
	t.Helper()

	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		t.Fatalf("create scrypt recipient: %v", err)
	}
	recipient.SetWorkFactor(1)

	buf := bytes.NewBuffer(nil)
	enc, err := age.Encrypt(buf, recipient)
	if err != nil {
		t.Fatalf("encrypt identity: %v", err)
	}

	if _, err := io.Copy(enc, strings.NewReader(plaintext)); err != nil {
		t.Fatalf("copy plaintext to encryptor: %v", err)
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

	_ = initSystemConfig(t, mainDB, true, "", "", "")

	tpl := template.New("app")
	tpl.Funcs(ui.TemplateFuncMap(tpl, newEmbeddedAssetVersionsForTest(t)))
	tpl, err = tpl.ParseFS(ui2.WidgetFS, "widget/*.gohtml")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	tpl, err = tpl.ParseFS(partial.TemplateFS, "*.gohtml")
	if err != nil {
		t.Fatalf("parse partial templates: %v", err)
	}

	return &maintenanceTestDependencies{
		mainDB:   mainDB,
		renderer: ui.NewRenderer(tpl),
		i18n:     i18n.NewI18n(),
	}
}
