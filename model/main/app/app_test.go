package app

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/db/entmain/enttest"
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
