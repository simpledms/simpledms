package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/simpledms/simpledms/db/entmain/systemconfig"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/encryptor"
	appmodel "github.com/simpledms/simpledms/model/main/app"
)

func TestTurnstileStartupStoresProductionKeysInDevelopment(t *testing.T) {
	db := newTurnstileConfigDB(t)
	serverx := &Server{devMode: true}
	serverx.initializeMainConfig(context.Background(), db, false)
	_, config := serverx.loadRuntimeSystemConfig(context.Background(), db)
	stored := config.Turnstile()
	if stored.TurnstileSiteKey != "stored-site" || stored.TurnstileSecret != "stored-secret" {
		t.Fatal("startup did not preserve the supplied production keys")
	}
	if stored.TurnstileSiteKey == appmodel.TurnstileTestSiteKey ||
		stored.TurnstileSecret == appmodel.TurnstileTestSecret {
		t.Fatal("development test keys were persisted over production keys")
	}
}

func TestTurnstileStoredConfigurationHonoursEnvironmentOverrides(t *testing.T) {
	for _, test := range []struct {
		name       string
		override   bool
		setSite    bool
		setSecret  bool
		site       string
		secret     string
		wantSite   string
		wantSecret string
	}{
		{
			name:       "disabled overrides preserve stored values",
			setSite:    true,
			setSecret:  true,
			site:       "replacement-site",
			secret:     "replacement-secret",
			wantSite:   "stored-site",
			wantSecret: "stored-secret",
		},
		{
			name:       "present variables replace stored values",
			override:   true,
			setSite:    true,
			setSecret:  true,
			site:       " replacement-site ",
			secret:     " replacement-secret ",
			wantSite:   "replacement-site",
			wantSecret: "replacement-secret",
		},
		{
			name:       "absent variables preserve stored values",
			override:   true,
			wantSite:   "stored-site",
			wantSecret: "stored-secret",
		},
		{
			name:       "one absent variable preserves its stored value",
			override:   true,
			setSite:    true,
			site:       "replacement-site",
			wantSite:   "replacement-site",
			wantSecret: "stored-secret",
		},
		{
			name:      "explicit empty secret clears protection",
			override:  true,
			setSecret: true,
			secret:    " \t ",
			wantSite:  "stored-site",
		},
		{
			name:      "explicit empty keys clear stored values",
			override:  true,
			setSite:   true,
			setSecret: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := newTurnstileConfigDB(t)
			serverx := &Server{}
			ctx := context.Background()
			serverx.initializeMainConfig(ctx, db, false)
			for _, key := range []string{
				"SIMPLEDMS_TURNSTILE_SITE_KEY", "SIMPLEDMS_TURNSTILE_SECRET",
			} {
				if err := os.Unsetenv(key); err != nil {
					t.Fatal(err)
				}
			}
			if test.setSite {
				t.Setenv("SIMPLEDMS_TURNSTILE_SITE_KEY", test.site)
			}
			if test.setSecret {
				t.Setenv("SIMPLEDMS_TURNSTILE_SECRET", test.secret)
			}
			serverx.applyOverrideDBConfigAfterIdentity(ctx, db, test.override)
			_, config := serverx.loadRuntimeSystemConfig(ctx, db)
			stored := config.Turnstile()
			if stored.TurnstileSiteKey != test.wantSite ||
				stored.TurnstileSecret != test.wantSecret {
				t.Fatal("effective stored Turnstile configuration does not match override rules")
			}
		})
	}
}

func TestTurnstileOverridesWaitUntilIdentityIsUnlocked(t *testing.T) {
	db := newTurnstileConfigDB(t)
	serverx := &Server{}
	ctx := context.Background()
	serverx.initializeMainConfig(ctx, db, false)
	t.Setenv("SIMPLEDMS_TURNSTILE_SITE_KEY", "replacement-site")
	t.Setenv("SIMPLEDMS_TURNSTILE_SECRET", "replacement-secret")
	identity := encryptor.NilableX25519MainIdentity
	encryptor.NilableX25519MainIdentity = nil
	serverx.initializeMainConfig(ctx, db, true)
	bootstrap := db.ReadOnlyConn.SystemConfig.Query().
		Select(systemconfig.FieldTurnstileSiteKey).OnlyX(ctx)
	if bootstrap.TurnstileSiteKey != "stored-site" {
		t.Fatal("Turnstile configuration was changed before unlock")
	}
	encryptor.NilableX25519MainIdentity = identity
	stored := db.ReadOnlyConn.SystemConfig.Query().OnlyX(ctx)
	if stored.TurnstileSecret.String() != "stored-secret" {
		t.Fatal("Turnstile secret was changed before unlock")
	}
}

func newTurnstileConfigDB(t *testing.T) *sqlx.MainDB {
	t.Helper()
	previousIdentity := encryptor.NilableX25519MainIdentity
	t.Cleanup(func() { encryptor.NilableX25519MainIdentity = previousIdentity })
	t.Setenv("SIMPLEDMS_INITIAL_ACCOUNT_EMAIL", "")
	t.Setenv("SIMPLEDMS_INITIAL_TENANT_NAME", "")
	t.Setenv("SIMPLEDMS_TURNSTILE_SITE_KEY", " stored-site ")
	t.Setenv("SIMPLEDMS_TURNSTILE_SECRET", " stored-secret ")
	db := sqlx.NewMainDB(filepath.Join(t.TempDir(), "main.sqlite3"))
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.ReadWriteConn.Schema.Create(context.Background()); err != nil {
		t.Fatal(err)
	}
	return db
}
