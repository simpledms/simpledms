package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/enttest"
	"github.com/simpledms/simpledms/db/entx"
)

func TestInitAppStoresGotenbergURL(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:app-test?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()
	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = InitAppWithoutCustomContext(
		ctx,
		tx,
		"",
		true,
		S3Config{},
		TLSConfig{},
		MailerConfig{},
		OCRConfig{GotenbergURL: "http://gotenberg:3000"},
		TurnstileConfig{},
	)
	if err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	config := client.SystemConfig.Query().OnlyX(ctx)
	if config.GotenbergURL != "http://gotenberg:3000" {
		t.Fatalf("expected Gotenberg URL to be stored, got %q", config.GotenbergURL)
	}
}

func TestInitAppStoresTrimmedTurnstileConfiguration(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:turnstile-app-test?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	client := entmain.NewClient(entmain.Driver(entsql.OpenDB("sqlite3", db)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()
	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = InitAppWithoutCustomContext(
		ctx,
		tx,
		"",
		true,
		S3Config{},
		TLSConfig{},
		MailerConfig{},
		OCRConfig{},
		TurnstileConfig{
			TurnstileSiteKey: " site-key ",
			TurnstileSecret:  " secret-key ",
		},
	)
	if err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	config := client.SystemConfig.Query().OnlyX(ctx)
	if config.TurnstileSiteKey != "site-key" {
		t.Errorf("expected trimmed Turnstile site key, got %q", config.TurnstileSiteKey)
	}
	if got := config.TurnstileSecret.String(); got != "secret-key" {
		t.Errorf("expected decrypted Turnstile secret, got %q", got)
	}
	var stored []byte
	if err := db.QueryRow("SELECT turnstile_secret FROM system_configs").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stored), "secret-key") {
		t.Fatal("database stores the Turnstile secret in plaintext")
	}
	var roundTrip entx.EncryptedString
	if err := roundTrip.Scan(stored); err != nil || roundTrip.String() != "secret-key" {
		t.Fatalf("stored secret failed decryption roundtrip: value=%q error=%v", roundTrip.String(), err)
	}
	jsonValue, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(jsonValue), "secret-key") || strings.Contains(config.String(), "secret-key") {
		t.Fatal("formatted or JSON config leaked the Turnstile secret")
	}
}
