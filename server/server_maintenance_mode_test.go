package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"html/template"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestMaintenanceRootReturnsServiceUnavailable(t *testing.T) {
	deps := newMaintenanceTestDependencies(t)

	handler := newMaintenanceModeHandler(
		deps.mainDB,
		os.DirFS(t.TempDir()),
		false,
		deps.i18n,
		deps.renderer,
		[]byte("irrelevant"),
		false,
		nil,
	)
	clearMainIdentity(t)

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
		for _, response := range responses {
			if response.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, response.Code)
			}
			if response.Body.Len() != 0 {
				t.Errorf("expected empty success response, got %q", response.Body.String())
			}
		}

		synctest.Sleep(time.Hour)
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

func TestMaintenanceUnlockListenerLifecycle(t *testing.T) {
	testCases := []struct {
		name            string
		useTLS          bool
		shutdownFailure bool
	}{
		{name: "HTTP"},
		{name: "certificate TLS", useTLS: true},
		{name: "shutdown error", shutdownFailure: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testMaintenanceListenerLifecycle(
				t,
				testCase.useTLS,
				testCase.shutdownFailure,
			)
		})
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
		false,
		stopFn,
	)
}

func testMaintenanceListenerLifecycle(t *testing.T, useTLS, shutdownFailure bool) {
	t.Helper()
	const passphrase = "listener-lifecycle-sentinel-passphrase"
	clearMainIdentity(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().String()
	if shutdownFailure {
		listener = newCloseErrorListener(listener)
	}

	certFile, keyFile := "", ""
	if useTLS {
		certFile, keyFile = writeTestCertificate(t)
	}

	var logs bytes.Buffer
	previousLogWriter := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() {
		log.SetOutput(previousLogWriter)
	})

	serverx := &http.Server{}
	stopDone := make(chan struct{})
	serverx.Handler = newMaintenanceCommandHandler(
		t,
		mustEncryptIdentityWithPassphrase(t, passphrase),
		func() {
			stopMaintenanceModeServer(serverx)
			close(stopDone)
		},
	)
	serveDone := make(chan error, 1)
	go func() {
		if useTLS {
			serveDone <- serverx.ServeTLS(listener, certFile, keyFile)
			return
		}
		serveDone <- serverx.Serve(listener)
	}()

	transport := &http.Transport{DisableKeepAlives: true}
	scheme := "http"
	if useTLS {
		scheme = "https"
		transport.TLSClientConfig = &tls.Config{ // #nosec G402 -- generated loopback test certificate.
			InsecureSkipVerify: true,
		}
	}
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}
	t.Cleanup(transport.CloseIdleConnections)

	startedAt := time.Now()
	response, err := client.Post(
		scheme+"://"+address+"/-/unlock-cmd",
		"application/json",
		bytes.NewReader(marshalUnlockRequestBody(t, passphrase)),
	)
	if err != nil {
		t.Fatalf("unlock request: %v", err)
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		_ = response.Body.Close()
		t.Fatalf("read unlock response: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close unlock response: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed >= 5*time.Second {
		t.Fatalf("expected prompt unlock response, took %s", elapsed)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
	if len(responseBody) != 0 {
		t.Fatalf("expected existing empty response, got %q", responseBody)
	}
	if response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", response.Header.Get("Cache-Control"))
	}
	if response.Header.Get("Pragma") != "no-cache" {
		t.Fatalf("expected Pragma no-cache, got %q", response.Header.Get("Pragma"))
	}

	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("maintenance stop did not complete")
	}
	select {
	case err := <-serveDone:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("expected server closed, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("maintenance server did not stop")
	}
	if shutdownFailure && !strings.Contains(logs.String(), closeErrorListenerMessage) {
		t.Fatalf("expected graceful shutdown error log, got %q", logs.String())
	}
	if strings.Contains(logs.String(), passphrase) {
		t.Fatal("maintenance lifecycle logs exposed passphrase")
	}

	replacementListener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("rebind maintenance address: %v", err)
	}
	replacementServer := &http.Server{
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			_, _ = rw.Write([]byte("replacement"))
		}),
	}
	replacementDone := make(chan error, 1)
	go func() {
		if useTLS {
			replacementDone <- replacementServer.ServeTLS(
				replacementListener,
				certFile,
				keyFile,
			)
			return
		}
		replacementDone <- replacementServer.Serve(replacementListener)
	}()

	replacementResponse, err := client.Get(scheme + "://" + address + "/")
	if err != nil {
		_ = replacementServer.Close()
		t.Fatalf("replacement request: %v", err)
	}
	replacementBody, err := io.ReadAll(replacementResponse.Body)
	if err != nil {
		_ = replacementResponse.Body.Close()
		_ = replacementServer.Close()
		t.Fatalf("read replacement response: %v", err)
	}
	if err := replacementResponse.Body.Close(); err != nil {
		_ = replacementServer.Close()
		t.Fatalf("close replacement response: %v", err)
	}
	if string(replacementBody) != "replacement" {
		_ = replacementServer.Close()
		t.Fatalf("expected replacement response, got %q", replacementBody)
	}
	if err := replacementServer.Close(); err != nil {
		t.Fatalf("close replacement server: %v", err)
	}
	select {
	case err := <-replacementDone:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("expected replacement server closed, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("replacement server did not stop")
	}
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
	tpl.Funcs(ui.TemplateFuncMap(tpl))
	tpl, err = tpl.ParseFS(ui2.WidgetFS, "widget/*.gohtml")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	return &maintenanceTestDependencies{
		mainDB:   mainDB,
		renderer: ui.NewRenderer(tpl),
		i18n:     i18n.NewI18n(),
	}
}
