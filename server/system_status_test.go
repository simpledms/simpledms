package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	systemconfigmodel "github.com/simpledms/simpledms/model/main/systemconfig"
	"github.com/simpledms/simpledms/model/main/systemstatus"
	"github.com/simpledms/simpledms/ui/uix/route"
)

func TestSystemStatusPageAndRefreshRenderForAdmin(t *testing.T) {
	harness := newActionTestHarness(t)
	harness.router.RegisterPage(
		route.SystemStatusRoute(),
		harness.actions.Dashboard.SystemStatusPage.Handler,
	)

	email := "system-status-admin@example.com"
	password := "supersecret"
	createAccountWithRole(t, harness.mainDB, email, password, mainrole.Admin)
	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	pageReq := httptest.NewRequest(http.MethodGet, route.SystemStatus(), nil)
	pageReq.AddCookie(sessionCookie)
	pageRR := httptest.NewRecorder()
	harness.router.ServeHTTP(pageRR, pageReq)

	if pageRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, pageRR.Code)
	}
	// the harness has no storage and no mail server configured
	for _, want := range []string{
		"System status",
		"No storage is configured. Files cannot be uploaded.",
		"No mail server is configured.",
	} {
		if !strings.Contains(pageRR.Body.String(), want) {
			t.Fatalf("expected page to contain %q, body was: %s", want, pageRR.Body.String())
		}
	}

	partialReq := httptest.NewRequest(
		http.MethodPost,
		harness.actions.Dashboard.SystemStatusPartial.Endpoint(),
		nil,
	)
	partialReq.AddCookie(sessionCookie)
	partialReq.Header.Set("HX-Request", "true")
	partialRR := httptest.NewRecorder()
	harness.router.ServeHTTP(partialRR, partialReq)

	if partialRR.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, partialRR.Code)
	}
	if !strings.Contains(partialRR.Body.String(), `id="systemStatus"`) {
		t.Fatalf("expected refreshed status island, body was: %s", partialRR.Body.String())
	}
}

func TestSystemStatusIsForbiddenForNonAdmins(t *testing.T) {
	harness := newActionTestHarness(t)
	harness.router.RegisterPage(
		route.SystemStatusRoute(),
		harness.actions.Dashboard.SystemStatusPage.Handler,
	)

	email := "system-status-user@example.com"
	password := "supersecret"
	createAccountWithRole(t, harness.mainDB, email, password, mainrole.User)
	sessionCookie := signInAndGetSessionCookie(t, harness, email, password)

	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, route.SystemStatus(), nil),
		httptest.NewRequest(
			http.MethodPost,
			harness.actions.Dashboard.SystemStatusPartial.Endpoint(),
			nil,
		),
	} {
		req.AddCookie(sessionCookie)
		req.Header.Set("HX-Request", "true")
		rr := httptest.NewRecorder()
		harness.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected status %d for %s, got %d", http.StatusForbidden, req.URL, rr.Code)
		}
	}
}

func TestSystemStatusReportsServiceConnectionsAndHidesSecrets(t *testing.T) {
	harness := newActionTestHarness(t)

	tikaServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/version" {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = rw.Write([]byte("Apache Tika 3.0.0"))
		},
	))
	defer tikaServer.Close()
	xbergServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/health" {
				rw.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = rw.Write([]byte(`{"status":"healthy","version":"4.1.0"}`))
		},
	))
	defer xbergServer.Close()
	gotenbergServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusServiceUnavailable)
		},
	))
	defer gotenbergServer.Close()

	checker := systemstatus.NewSystemStatusChecker(
		systemconfigmodel.NewSystemConfig(
			&entmain.SystemConfig{
				S3Endpoint:        "s3.example.com",
				S3AccessKeyID:     "access-key-id",
				S3SecretAccessKey: entx.EncryptedString("s3-secret-value"),
				MailerPassword:    entx.EncryptedString("mail-secret-value"),
				OcrTikaURL:        tikaServer.URL,
				OcrXbergURL:       xbergServer.URL,
				GotenbergURL:      gotenbergServer.URL,
			},
			false,
			false,
			false,
			"https://dms.example.com",
			"example.org",
			"",
		),
		nil,
		false,
		false,
		false,
	)

	status := checkSystemStatus(t, harness, checker)

	ocr := findSystemStatusComponent(t, status, "OCR")
	if ocr.Level() != systemstatus.StatusLevelOK {
		t.Fatalf("expected OCR to be OK, got findings %v", status.texts(ocr.Findings()))
	}
	assertSystemStatusSetting(t, status, ocr, "Tika connection", "Connected (Apache Tika 3.0.0)")
	assertSystemStatusSetting(t, status, ocr, "Xberg connection", "Connected (4.1.0)")

	previews := findSystemStatusComponent(t, status, "PDF previews")
	if previews.Level() != systemstatus.StatusLevelError {
		t.Fatalf("expected unreachable Gotenberg to be an error, got %v", previews.Level())
	}
	assertSystemStatusFinding(t, status, previews, "Gotenberg is not reachable")

	general := findSystemStatusComponent(t, status, "General")
	assertSystemStatusFinding(t, status, general, "relying party ID does not match")

	for _, component := range status.Components() {
		for _, setting := range component.Settings() {
			value := setting.Value().String(status.ctx)
			if strings.Contains(value, "secret-value") {
				t.Fatalf("secret shown as value of %q", setting.Label().String(status.ctx))
			}
		}
	}
}

func TestSystemStatusReportsMailsThatCouldNotBeSent(t *testing.T) {
	harness := newActionTestHarness(t)

	email := "system-status-mail@example.com"
	createAccountWithRole(t, harness.mainDB, email, "supersecret", mainrole.User)
	receiver := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())
	for _, retryCount := range []int{0, 3} {
		harness.mainDB.ReadWriteConn.Mail.Create().
			SetSubject("Subject").
			SetBody("Body").
			SetReceiver(receiver).
			SetRetryCount(retryCount).
			SaveX(context.Background())
	}

	checker := systemstatus.NewSystemStatusChecker(
		harness.infra.SystemConfig(),
		nil,
		false,
		false,
		false,
	)
	status := checkSystemStatus(t, harness, checker)

	mail := findSystemStatusComponent(t, status, "Email")
	assertSystemStatusSetting(t, status, mail, "Emails waiting to be sent", "1")
	assertSystemStatusSetting(t, status, mail, "Emails that could not be sent", "1")
	assertSystemStatusFinding(t, status, mail, "1 emails could not be sent after 3 attempts.")
}

func TestSystemStatusWarnsBeforeCertificateExpires(t *testing.T) {
	harness := newActionTestHarness(t)
	// valid for one hour
	certFilepath, keyFilepath := writeTestCertificate(t)

	checker := systemstatus.NewSystemStatusChecker(
		systemconfigmodel.NewSystemConfig(
			&entmain.SystemConfig{
				TLSCertFilepath:       certFilepath,
				TLSPrivateKeyFilepath: keyFilepath,
			},
			false,
			false,
			false,
			"",
			"",
			"",
		),
		nil,
		false,
		false,
		false,
	)

	status := checkSystemStatus(t, harness, checker)

	tlsStatus := findSystemStatusComponent(t, status, "TLS")
	if tlsStatus.Level() != systemstatus.StatusLevelWarning {
		t.Fatalf("expected certificate expiring soon to be a warning, got %v", tlsStatus.Level())
	}
	assertSystemStatusFinding(t, status, tlsStatus, "The certificate expires on")
}

// checkedSystemStatus keeps the context for translating the reported texts.
type checkedSystemStatus struct {
	*systemstatus.SystemStatus
	ctx ctxx.Context
}

func (qq *checkedSystemStatus) texts(findings []*systemstatus.Finding) []string {
	var texts []string
	for _, finding := range findings {
		texts = append(texts, finding.Message().String(qq.ctx))
	}
	return texts
}

func checkSystemStatus(
	t *testing.T,
	harness *actionTestHarness,
	checker *systemstatus.SystemStatusChecker,
) *checkedSystemStatus {
	t.Helper()

	email := "system-status-checker@example.com"
	createAccountWithRole(t, harness.mainDB, email, "supersecret", mainrole.Admin)
	adminx := harness.mainDB.ReadWriteConn.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		OnlyX(context.Background())

	var checked *checkedSystemStatus
	err := withMainContext(
		t,
		harness,
		adminx,
		func(_ *entmain.Tx, mainCtx *ctxx.MainContext) error {
			status, err := checker.Check(mainCtx)
			if err != nil {
				return err
			}
			checked = &checkedSystemStatus{
				SystemStatus: status,
				ctx:          mainCtx,
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	return checked
}

func findSystemStatusComponent(
	t *testing.T,
	status *checkedSystemStatus,
	title string,
) *systemstatus.ComponentStatus {
	t.Helper()

	for _, component := range status.Components() {
		if component.Title().String(status.ctx) == title {
			return component
		}
	}
	t.Fatalf("component %q not found", title)
	return nil
}

func assertSystemStatusSetting(
	t *testing.T,
	status *checkedSystemStatus,
	component *systemstatus.ComponentStatus,
	label string,
	want string,
) {
	t.Helper()

	for _, setting := range component.Settings() {
		if setting.Label().String(status.ctx) != label {
			continue
		}
		if got := setting.Value().String(status.ctx); got != want {
			t.Fatalf("expected %q to be %q, got %q", label, want, got)
		}
		return
	}
	t.Fatalf("setting %q not found", label)
}

func assertSystemStatusFinding(
	t *testing.T,
	status *checkedSystemStatus,
	component *systemstatus.ComponentStatus,
	wantSubstring string,
) {
	t.Helper()

	texts := status.texts(component.Findings())
	for _, text := range texts {
		if strings.Contains(text, wantSubstring) {
			return
		}
	}
	t.Fatalf("expected finding containing %q, got %v", wantSubstring, texts)
}
