package entx_test

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"filippo.io/age"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/migrate"
	"github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/encryptor"
)

func TestTurnstileConfigMigrationAddsNullableColumns(t *testing.T) {
	migrations, err := migrate.NewMigrationsMainFS()
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "turnstile.sqlite3")+"?_foreign_keys=0")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var migrationName string
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		if strings.HasSuffix(name, "_turnstile_config.up.sql") {
			migrationName = strings.TrimSuffix(name, ".up.sql")
			break
		}
		script, err := fs.ReadFile(migrations, name)
		if err != nil {
			t.Fatal(err)
		}
		if err := executeSchemaMigration(db, string(script)); err != nil {
			t.Fatalf("replay %s: %v", name, err)
		}
	}
	if migrationName == "" {
		t.Fatal("Turnstile configuration migration is missing")
	}
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	previousIdentity := encryptor.NilableX25519MainIdentity
	encryptor.NilableX25519MainIdentity = identity
	t.Cleanup(func() { encryptor.NilableX25519MainIdentity = previousIdentity })
	secret, err := encryptor.Encrypt([]byte("legacy-s3-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO system_configs
(id, created_at, updated_at, x25519_identity, is_identity_encrypted_with_passphrase,
s3_endpoint, s3_access_key_id, s3_secret_access_key, s3_bucket_name, s3_use_ssl,
tls_enable_autocert, tls_cert_filepath, tls_private_key_filepath, tls_autocert_email,
tls_autocert_hosts, mailer_host, mailer_port, mailer_username, mailer_password,
mailer_from, mailer_insecure_skip_verify, mailer_use_implicit_ssl_tls)
VALUES (1, '2026-01-01', '2026-01-01', X'', false, 'endpoint', 'access', ?, 'bucket', true,
false, '', '', '', '[]', '', 25, '', NULL, '', false, false)`, secret); err != nil {
		t.Fatal(err)
	}
	script, err := fs.ReadFile(migrations, migrationName+".up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(script))
	for _, forbidden := range []string{"CREATE TABLE", "INSERT", "DROP TABLE", "RENAME TO"} {
		if strings.Contains(upper, forbidden) {
			t.Fatalf("migration is not additive; contains %q", forbidden)
		}
	}
	if err := executeSchemaMigration(db, string(script)); err != nil {
		t.Fatal(err)
	}
	var endpoint, bucket string
	var siteKey, turnstileSecret, legacySecret any
	if err := db.QueryRow(`SELECT s3_endpoint, s3_bucket_name, turnstile_site_key,
turnstile_secret, s3_secret_access_key FROM system_configs WHERE id = 1`).Scan(
		&endpoint, &bucket, &siteKey, &turnstileSecret, &legacySecret); err != nil {
		t.Fatal(err)
	}
	if endpoint != "endpoint" || bucket != "bucket" || siteKey != nil || turnstileSecret != nil {
		t.Fatalf("migration changed existing config or failed to leave new fields NULL: endpoint=%v bucket=%v site=%v secret=%v", endpoint, bucket, siteKey, turnstileSecret)
	}
	if string(legacySecret.([]byte)) == "" {
		t.Fatal("migration lost the legacy encrypted secret")
	}
	client := entmain.NewClient(entmain.Driver(entsql.OpenDB("sqlite3", db)))
	config, err := client.SystemConfig.Query().Only(privacy.DecisionContext(context.Background(), privacy.Allow))
	if err != nil {
		t.Fatalf("query upgraded system config through Ent: %v", err)
	}
	if config.TurnstileSiteKey != "" || len(config.TurnstileSecret) != 0 {
		t.Fatalf("nullable Turnstile fields should read empty: site key=%q secret length=%d", config.TurnstileSiteKey, len(config.TurnstileSecret))
	}
	if got := config.S3SecretAccessKey.String(); got != "legacy-s3-secret" {
		t.Fatalf("Ent did not decrypt preserved legacy S3 secret: got %q", got)
	}
}
