package server

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/simpledms/simpledms/db/entmain/systemconfig"
	"github.com/simpledms/simpledms/encryptor"
)

const startupTestTimeout = 5 * time.Second

func TestEnsureMainIdentityKeepsUnlockListenerUntilApplicationHandoff(t *testing.T) {
	const passphrase = "startup-handoff-passphrase"
	for _, useTLS := range []bool{false, true} {
		name := "HTTP"
		if useTLS {
			name = "certificate TLS"
		}
		t.Run(name, func(t *testing.T) {
			clearMainIdentity(t)
			deps := newMaintenanceTestDependencies(t)
			port := reserveTestPort(t)
			config := deps.mainDB.ReadOnlyConn.SystemConfig.Query().FirstX(context.Background())
			update := deps.mainDB.ReadWriteConn.SystemConfig.Update().
				Where(systemconfig.ID(config.ID)).
				SetX25519Identity(mustEncryptIdentityWithPassphrase(t, passphrase)).
				SetIsIdentityEncryptedWithPassphrase(true)
			if useTLS {
				cert, key := writeTestCertificate(t)
				update.SetTLSCertFilepath(cert).SetTLSPrivateKeyFilepath(key)
			}
			update.SaveX(context.Background())
			encryptor.NilableX25519MainIdentity = nil
			identityServer := &Server{unsafePort: port, assetsFS: os.DirFS(t.TempDir())}
			config = identityServer.loadBootstrapSystemConfig(context.Background(), deps.mainDB)
			ensureResult := make(chan ensureMainIdentityResult, 1)
			go func() {
				startup, err := identityServer.ensureMainIdentity(
					deps.mainDB, deps.i18n, deps.renderer, config, nil, false, nil,
				)
				ensureResult <- ensureMainIdentityResult{startup: startup, err: err}
			}()

			scheme := "http"
			transport := &http.Transport{}
			if useTLS {
				scheme = "https"
				transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402
			}
			client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
			t.Cleanup(transport.CloseIdleConnections)
			url := scheme + "://127.0.0.1:" + strconv.Itoa(port)
			var startup *startupServer
			startResult := make(chan error, 1)
			startCalled, startReturned := false, false
			t.Cleanup(func() {
				if startup == nil {
					select {
					case ensured := <-ensureResult:
						if ensured.err != nil {
							t.Errorf("ensureMainIdentity during cleanup: %v", ensured.err)
						}
						startup = ensured.startup
					default:
						startup = unlockAndCollectStartup(
							t, client, url, passphrase, ensureResult,
						)
					}
				}
				if startup != nil {
					if err := startup.server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
						t.Errorf("close startup server: %v", err)
					}
					if startCalled && !startReturned {
						select {
						case <-startResult:
						case <-time.After(startupTestTimeout):
							t.Error("PreparedServer.Start did not finish during cleanup")
						}
					} else if !startCalled {
						select {
						case <-startup.done:
						case <-time.After(startupTestTimeout):
							t.Error("startup server did not stop during cleanup")
						}
					}
				}
			})
			waitForListener(t, client, url)
			appCalls := make(chan struct{}, 1)
			mux := http.NewServeMux()
			mux.HandleFunc("GET /registered", func(rw http.ResponseWriter, _ *http.Request) {
				select {
				case appCalls <- struct{}{}:
				default:
				}
				_, _ = io.WriteString(rw, "application")
			})
			response := getResponse(t, client, url+"/")
			assertMaintenanceResponse(t, response)

			unlock, err := client.Post(
				url+"/-/unlock-cmd",
				"application/json",
				strings.NewReader(string(marshalUnlockRequestBody(t, passphrase))),
			)
			if err != nil {
				t.Fatalf("unlock request: %v", err)
			}
			closeResponseBody(t, unlock)
			if unlock.StatusCode != http.StatusOK {
				t.Fatalf("expected unlock status 200, got %d", unlock.StatusCode)
			}

			var ensured ensureMainIdentityResult
			select {
			case ensured = <-ensureResult:
			case <-time.After(startupTestTimeout):
				t.Fatal("ensureMainIdentity did not return after unlock")
			}
			if ensured.err != nil {
				t.Fatalf("ensure main identity: %v", ensured.err)
			}
			startup = ensured.startup
			if startup == nil {
				t.Fatal("ensureMainIdentity returned no startup server")
			}

			// The listener must remain on the maintenance handler until PreparedServer.Start.
			response = getResponse(t, client, url+"/still-locked")
			assertMaintenanceResponse(t, response)
			requestReused := false
			trace := &httptrace.ClientTrace{
				GotConn: func(info httptrace.GotConnInfo) { requestReused = info.Reused },
			}
			request := mustNewRequest(t, url+"/registered", trace)
			response = doRequest(t, client, request)
			assertMaintenanceResponse(t, response)
			if !requestReused {
				t.Fatal("expected post-unlock request to reuse persistent connection")
			}

			freshTransport := newStartupTestTransport(useTLS)
			freshClient := &http.Client{Transport: freshTransport, Timeout: 2 * time.Second}
			t.Cleanup(freshTransport.CloseIdleConnections)
			freshResponse := getResponse(t, freshClient, url+"/fresh-still-locked")
			assertMaintenanceResponse(t, freshResponse)
			select {
			case <-appCalls:
				t.Fatal("application handler ran before PreparedServer.Start")
			default:
			}
			prepared := &PreparedServer{
				nilableStartupServer: startup,
				handler:              mux,
			}
			startCalled = true
			go func() { startResult <- prepared.Start() }()

			response = waitForApplication(t, client, url+"/registered", appCalls)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("expected registered app response 200, got %d", response.StatusCode)
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read application response: %v", err)
			}
			if err := response.Body.Close(); err != nil {
				t.Fatalf("close application response: %v", err)
			}
			if string(body) != "application" {
				t.Fatalf("expected registered app body, got %q", body)
			}
			postHandoffReused := false
			postHandoffTrace := &httptrace.ClientTrace{
				GotConn: func(info httptrace.GotConnInfo) { postHandoffReused = info.Reused },
			}
			response = doRequest(
				t,
				client,
				mustNewRequest(t, url+"/registered", postHandoffTrace),
			)
			if !postHandoffReused {
				t.Fatal("expected post-handoff request to reuse persistent connection")
			}
			assertApplicationResponse(t, response)
			freshResponse = getResponse(t, freshClient, url+"/registered")
			assertApplicationResponse(t, freshResponse)

			if err := startup.server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				t.Fatalf("close startup server: %v", err)
			}
			select {
			case err := <-startResult:
				startReturned = true
				if !errors.Is(err, http.ErrServerClosed) {
					t.Fatalf("expected Start to return ErrServerClosed, got %v", err)
				}
			case <-time.After(startupTestTimeout):
				t.Fatal("PreparedServer.Start did not return after server close")
			}
		})
	}
}

func unlockAndCollectStartup(
	t *testing.T,
	client *http.Client,
	url, passphrase string,
	result <-chan ensureMainIdentityResult,
) *startupServer {
	t.Helper()
	deadline := time.NewTimer(startupTestTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case ensured := <-result:
			if ensured.err != nil {
				t.Errorf("ensureMainIdentity during cleanup: %v", ensured.err)
			}
			return ensured.startup
		case <-deadline.C:
			t.Error("ensureMainIdentity did not finish during cleanup")
			return nil
		case <-ticker.C:
			unlockResponse, err := client.Post(
				url+"/-/unlock-cmd",
				"application/json",
				strings.NewReader(string(marshalUnlockRequestBody(t, passphrase))),
			)
			if err == nil {
				closeResponseBody(t, unlockResponse)
			}
		}
	}
}

type ensureMainIdentityResult struct {
	startup *startupServer
	err     error
}

func TestEnsureMainIdentityReturnsBindFailure(t *testing.T) {
	clearMainIdentity(t)
	deps := newMaintenanceTestDependencies(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close occupied listener: %v", err)
		}
	}()
	config := deps.mainDB.ReadOnlyConn.SystemConfig.Query().FirstX(context.Background())
	deps.mainDB.ReadWriteConn.SystemConfig.UpdateOneID(config.ID).
		SetX25519Identity(mustEncryptIdentityWithPassphrase(t, "bind-failure")).
		SetIsIdentityEncryptedWithPassphrase(true).SaveX(context.Background())
	encryptor.NilableX25519MainIdentity = nil
	identityServer := &Server{
		unsafePort: listener.Addr().(*net.TCPAddr).Port,
		assetsFS:   os.DirFS(t.TempDir()),
	}
	config = identityServer.loadBootstrapSystemConfig(context.Background(), deps.mainDB)
	result := make(chan error, 1)
	go func() {
		_, err := identityServer.ensureMainIdentity(
			deps.mainDB, deps.i18n, deps.renderer, config, nil, false, nil,
		)
		result <- err
	}()
	select {
	case err := <-result:
		if err == nil || !strings.Contains(err.Error(), "maintenance server stopped before unlock") {
			t.Fatalf("expected listener bind failure, got %v", err)
		}
	case <-time.After(startupTestTimeout):
		t.Fatal("ensureMainIdentity did not return after listener bind failure")
	}
}

func waitForListener(t *testing.T, client *http.Client, url string) {
	t.Helper()
	deadline := time.Now().Add(startupTestTimeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		response, err := client.Get(url + "/")
		if err == nil {
			if response.StatusCode == http.StatusServiceUnavailable &&
				response.Header.Get("X-SimpleDMS-Maintenance") == "true" {
				closeResponseBody(t, response)
				return
			}
			closeResponseBody(t, response)
		}
		if time.Now().After(deadline) {
			t.Fatalf("startup listener did not become ready before deadline (last error: %v)", err)
		}
		<-ticker.C
	}
}

func waitForApplication(
	t *testing.T,
	client *http.Client,
	url string,
	appCalls <-chan struct{},
) *http.Response {
	t.Helper()
	deadline := time.Now().Add(startupTestTimeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		response, err := client.Get(url)
		if err != nil {
			t.Fatalf("application readiness request: %v", err)
		}
		if response.StatusCode != http.StatusServiceUnavailable ||
			response.Header.Get("X-SimpleDMS-Maintenance") != "true" {
			return response
		}
		closeResponseBody(t, response)
		select {
		case <-appCalls:
		case <-ticker.C:
		}
		if time.Now().After(deadline) {
			t.Fatalf("application handler did not become ready before deadline (last error: %v)", err)
		}
	}
}

func getResponse(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()
	response, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return response
}

func mustNewRequest(t *testing.T, url string, trace *httptrace.ClientTrace) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(context.Background(), trace),
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	return request
}

func doRequest(t *testing.T, client *http.Client, request *http.Request) *http.Response {
	t.Helper()
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("request %s: %v", request.URL, err)
	}
	return response
}

func assertMaintenanceResponse(t *testing.T, response *http.Response) {
	t.Helper()
	closeResponseBody(t, response)
	if response.StatusCode != http.StatusServiceUnavailable ||
		response.Header.Get("X-SimpleDMS-Maintenance") != "true" {
		t.Fatalf("expected marked maintenance response, got %d %v", response.StatusCode, response.Header)
	}
}

func assertApplicationResponse(t *testing.T, response *http.Response) {
	t.Helper()
	if response.StatusCode != http.StatusOK {
		closeResponseBody(t, response)
		t.Fatalf("expected application response 200, got %d", response.StatusCode)
	}
	if response.Header.Get("X-SimpleDMS-Maintenance") != "" {
		closeResponseBody(t, response)
		t.Fatal("application response retained maintenance marker")
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read application response: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close application response: %v", err)
	}
	if string(body) != "application" {
		t.Fatalf("expected registered app body, got %q", body)
	}
}

func closeResponseBody(t *testing.T, response *http.Response) {
	t.Helper()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Errorf("read response body: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
}

func newStartupTestTransport(useTLS bool) *http.Transport {
	transport := &http.Transport{DisableKeepAlives: true}
	if useTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402
	}
	return transport
}

func reserveTestPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}
