package modelmain

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/enttest"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/encryptor"
	"github.com/simpledms/simpledms/model/main/common/country"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/model/main/common/plan"
	"github.com/simpledms/simpledms/util/e"
)

func TestUploadLimitServiceSetGlobalUploadLimit(t *testing.T) {
	service := NewUploadLimitService()
	env := newUploadLimitServiceTestEnv(t, mainrole.Admin)
	defer env.cleanup()

	uploadLimit, err := service.SetGlobalUploadLimit(env.mainCtx, env.systemConfig, false, 5)
	if err != nil {
		t.Fatalf("set global upload limit: %v", err)
	}
	if uploadLimit.MiB() != 5 {
		t.Fatalf("expected 5 MiB, got %d", uploadLimit.MiB())
	}

	systemConfigx := env.mainCtx.MainTx.SystemConfig.Query().OnlyX(env.mainCtx)
	if systemConfigx.MaxUploadSizeMib != 5 {
		t.Fatalf("expected persisted limit 5 MiB, got %d", systemConfigx.MaxUploadSizeMib)
	}

	uploadLimit, err = service.SetGlobalUploadLimit(env.mainCtx, env.systemConfig, true, 999)
	if err != nil {
		t.Fatalf("set global unlimited upload limit: %v", err)
	}
	if !uploadLimit.IsUnlimited() {
		t.Fatal("expected unlimited upload limit")
	}

	systemConfigx = env.mainCtx.MainTx.SystemConfig.Query().OnlyX(env.mainCtx)
	if systemConfigx.MaxUploadSizeMib != 0 {
		t.Fatalf("expected persisted unlimited limit 0 MiB, got %d", systemConfigx.MaxUploadSizeMib)
	}
}

func TestUploadLimitServiceSetGlobalUploadLimitRequiresAdmin(t *testing.T) {
	service := NewUploadLimitService()
	env := newUploadLimitServiceTestEnv(t, mainrole.User)
	defer env.cleanup()

	_, err := service.SetGlobalUploadLimit(env.mainCtx, env.systemConfig, false, 1)
	_ = requireHTTPErrorStatus(t, err, http.StatusForbidden)
}

func TestUploadLimitServiceSetGlobalUploadLimitRequiresMainContext(t *testing.T) {
	service := NewUploadLimitService()
	visitorCtx := &ctxx.VisitorContext{
		Context: context.Background(),
	}

	_, err := service.SetGlobalUploadLimit(visitorCtx, nil, false, 1)
	_ = requireHTTPErrorStatus(t, err, http.StatusUnauthorized)
}

func TestUploadLimitServiceSetTenantUploadLimitOverride(t *testing.T) {
	service := NewUploadLimitService()
	env := newUploadLimitServiceTestEnv(t, mainrole.Admin)
	defer env.cleanup()

	tenantx := env.mainCtx.MainTx.Tenant.Create().
		SetName("Tenant A").
		SetCountry(country.Unknown).
		SetPlan(plan.Trial).
		SetTermsOfServiceAccepted(time.Now()).
		SetPrivacyPolicyAccepted(time.Now()).
		SaveX(env.mainCtx)

	uploadLimit, err := service.SetTenantUploadLimitOverride(
		env.mainCtx,
		tenantx.PublicID.String(),
		false,
		false,
		7,
	)
	if err != nil {
		t.Fatalf("set tenant upload limit override: %v", err)
	}
	if uploadLimit == nil || uploadLimit.MiB() != 7 {
		t.Fatalf("expected override 7 MiB, got %#v", uploadLimit)
	}

	tenantx = env.mainCtx.MainTx.Tenant.GetX(env.mainCtx, tenantx.ID)
	if tenantx.MaxUploadSizeMibOverride == nil || *tenantx.MaxUploadSizeMibOverride != 7 {
		t.Fatalf("expected persisted tenant override 7 MiB, got %#v", tenantx.MaxUploadSizeMibOverride)
	}

	uploadLimit, err = service.SetTenantUploadLimitOverride(
		env.mainCtx,
		tenantx.PublicID.String(),
		false,
		true,
		7,
	)
	if err != nil {
		t.Fatalf("set tenant unlimited upload limit override: %v", err)
	}
	if uploadLimit == nil || !uploadLimit.IsUnlimited() {
		t.Fatalf("expected unlimited override, got %#v", uploadLimit)
	}

	tenantx = env.mainCtx.MainTx.Tenant.GetX(env.mainCtx, tenantx.ID)
	if tenantx.MaxUploadSizeMibOverride == nil || *tenantx.MaxUploadSizeMibOverride != 0 {
		t.Fatalf("expected persisted tenant override 0 MiB, got %#v", tenantx.MaxUploadSizeMibOverride)
	}

	uploadLimit, err = service.SetTenantUploadLimitOverride(
		env.mainCtx,
		tenantx.PublicID.String(),
		true,
		false,
		7,
	)
	if err != nil {
		t.Fatalf("clear tenant upload limit override: %v", err)
	}
	if uploadLimit != nil {
		t.Fatalf("expected nil override for global default, got %#v", uploadLimit)
	}

	tenantx = env.mainCtx.MainTx.Tenant.GetX(env.mainCtx, tenantx.ID)
	if tenantx.MaxUploadSizeMibOverride != nil {
		t.Fatalf("expected nil persisted tenant override, got %#v", tenantx.MaxUploadSizeMibOverride)
	}
}

func TestUploadLimitServiceSetTenantUploadLimitOverrideErrors(t *testing.T) {
	service := NewUploadLimitService()
	env := newUploadLimitServiceTestEnv(t, mainrole.Admin)
	defer env.cleanup()

	_, err := service.SetTenantUploadLimitOverride(env.mainCtx, "", false, false, 1)
	_ = requireHTTPErrorStatus(t, err, http.StatusBadRequest)

	_, err = service.SetTenantUploadLimitOverride(env.mainCtx, "missing", false, false, 1)
	_ = requireHTTPErrorStatus(t, err, http.StatusNotFound)

	tenantx := env.mainCtx.MainTx.Tenant.Create().
		SetName("Tenant B").
		SetCountry(country.Unknown).
		SetPlan(plan.Trial).
		SetTermsOfServiceAccepted(time.Now()).
		SetPrivacyPolicyAccepted(time.Now()).
		SaveX(env.mainCtx)

	_, err = service.SetTenantUploadLimitOverride(env.mainCtx, tenantx.PublicID.String(), false, false, 0)
	_ = requireHTTPErrorStatus(t, err, http.StatusBadRequest)
}

type uploadLimitServiceTestEnv struct {
	mainCtx      *ctxx.MainContext
	systemConfig *SystemConfig
	cleanup      func()
}

func newUploadLimitServiceTestEnv(t *testing.T, role mainrole.MainRole) *uploadLimitServiceTestEnv {
	t.Helper()

	oldIdentity := encryptor.NilableX25519MainIdentity
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate test identity: %v", err)
	}
	encryptor.NilableX25519MainIdentity = identity

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", strings.ReplaceAll(t.Name(), "/", "_"))
	mainDB := enttest.Open(t, "sqlite3", dsn)

	mainDB.SystemConfig.Create().
		SetX25519Identity([]byte(identity.String())).
		SetIsIdentityEncryptedWithPassphrase(false).
		SetS3Endpoint("http://localhost").
		SetS3AccessKeyID("access-key").
		SetS3SecretAccessKey(entx.NewEncryptedString("secret")).
		SetS3BucketName("bucket").
		SetMailerPassword(entx.NewEncryptedString("")).
		SetTLSEnableAutocert(false).
		SetTLSCertFilepath("").
		SetTLSPrivateKeyFilepath("").
		SetTLSAutocertEmail("").
		SetTLSAutocertHosts([]string{}).
		SaveX(context.Background())

	mainTx, err := mainDB.Tx(context.Background())
	if err != nil {
		t.Fatalf("start main transaction: %v", err)
	}

	mainCtx := &ctxx.MainContext{
		VisitorContext: &ctxx.VisitorContext{
			Context: context.Background(),
			MainTx:  mainTx,
		},
		Account: &entmain.Account{
			Role: role,
		},
	}

	systemConfigx := mainCtx.MainTx.SystemConfig.Query().OnlyX(mainCtx)
	systemConfig := NewSystemConfig(
		systemConfigx,
		false,
		false,
		true,
		"",
		"",
		"",
	)

	cleanup := func() {
		if err := mainTx.Rollback(); err != nil {
			t.Fatalf("rollback main transaction: %v", err)
		}
		if err := mainDB.Close(); err != nil {
			t.Fatalf("close main db: %v", err)
		}
		encryptor.NilableX25519MainIdentity = oldIdentity
	}

	return &uploadLimitServiceTestEnv{
		mainCtx:      mainCtx,
		systemConfig: systemConfig,
		cleanup:      cleanup,
	}
}

func requireHTTPErrorStatus(t *testing.T, err error, expectedStatus int) *e.HTTPError {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with status %d", expectedStatus)
	}

	httpErr, ok := err.(*e.HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %T", err)
	}
	if httpErr.StatusCode() != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, httpErr.StatusCode())
	}

	return httpErr
}
