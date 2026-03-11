package server

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	migratemain "github.com/simpledms/simpledms/db/entmain/migrate"
	"github.com/simpledms/simpledms/db/sqlx"
)

func TestNormalizeMainDBBeforeDevSchemaCreateHandlesNullPasskeyRecoveryCodes(t *testing.T) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "main.sqlite3")
	rawDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer func() {
		_ = rawDB.Close()
	}()

	_, err = rawDB.Exec(`
		CREATE TABLE accounts (
			id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
			created_at datetime NOT NULL,
			created_by integer NULL,
			updated_at datetime NOT NULL,
			updated_by integer NULL,
			deleted_by integer NULL,
			deleted_at datetime NULL,
			public_id text NOT NULL,
			email text NOT NULL,
			first_name text NOT NULL,
			last_name text NOT NULL,
			language text NOT NULL,
			subscribed_to_newsletter_at datetime NULL,
			password_salt text NOT NULL DEFAULT (''),
			password_hash text NOT NULL DEFAULT (''),
			temporary_password_salt text NOT NULL DEFAULT (''),
			temporary_password_hash text NOT NULL DEFAULT (''),
			temporary_password_expires_at datetime NULL,
			temporary_two_factor_auth_key_encrypted text NOT NULL DEFAULT (''),
			two_factory_auth_key_encrypted text NOT NULL DEFAULT (''),
			two_factor_auth_recovery_code_salt text NOT NULL DEFAULT (''),
			two_factor_auth_recovery_code_hashes json NOT NULL,
			last_login_attempt_at datetime NULL,
			passkey_login_enabled bool NULL DEFAULT (false),
			passkey_recovery_code_salt text NULL DEFAULT (''),
			passkey_recovery_code_hashes json NULL,
			role text NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("create legacy accounts table: %v", err)
	}

	_, err = rawDB.Exec(`
		INSERT INTO accounts (
			created_at,
			updated_at,
			public_id,
			email,
			first_name,
			last_name,
			language,
			password_salt,
			password_hash,
			temporary_password_salt,
			temporary_password_hash,
			temporary_two_factor_auth_key_encrypted,
			two_factory_auth_key_encrypted,
			two_factor_auth_recovery_code_salt,
			two_factor_auth_recovery_code_hashes,
			role,
			passkey_login_enabled,
			passkey_recovery_code_salt,
			passkey_recovery_code_hashes
		) VALUES (
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP,
			'acc_test_1',
			'test@example.com',
			'Test',
			'User',
			'English',
			'',
			'',
			'',
			'',
			'',
			'',
			'',
			'[]',
			'User',
			NULL,
			NULL,
			NULL
		);
	`)
	if err != nil {
		t.Fatalf("insert legacy account row: %v", err)
	}

	mainDB := sqlx.NewMainDB(dbPath)
	defer func() {
		_ = mainDB.Close()
	}()

	ctx := context.Background()
	err = normalizeMainDBBeforeDevSchemaCreate(mainDB)
	if err != nil {
		t.Fatalf("normalize main db: %v", err)
	}

	err = mainDB.ReadWriteConn.Schema.Create(
		ctx,
		migratemain.WithDropIndex(true),
		migratemain.WithDropColumn(true),
	)
	if err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	accountx := mainDB.ReadWriteConn.Account.Query().OnlyX(ctx)
	if accountx.PasskeyLoginEnabled {
		t.Fatal("expected null passkey_login_enabled to be normalized to false")
	}
	if accountx.PasskeyRecoveryCodeSalt != "" {
		t.Fatalf("expected empty passkey recovery code salt, got %q", accountx.PasskeyRecoveryCodeSalt)
	}
	if len(accountx.PasskeyRecoveryCodeHashes) != 0 {
		t.Fatalf("expected empty passkey recovery code hashes, got %d", len(accountx.PasskeyRecoveryCodeHashes))
	}
}

func TestNormalizeMainDBBeforeDevSchemaCreateHandlesMissingPasskeyColumns(t *testing.T) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "main.sqlite3")
	rawDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer func() {
		_ = rawDB.Close()
	}()

	_, err = rawDB.Exec(`
		CREATE TABLE accounts (
			id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
			created_at datetime NOT NULL,
			created_by integer NULL,
			updated_at datetime NOT NULL,
			updated_by integer NULL,
			deleted_by integer NULL,
			deleted_at datetime NULL,
			public_id text NOT NULL,
			email text NOT NULL,
			first_name text NOT NULL,
			last_name text NOT NULL,
			language text NOT NULL,
			subscribed_to_newsletter_at datetime NULL,
			password_salt text NOT NULL DEFAULT (''),
			password_hash text NOT NULL DEFAULT (''),
			temporary_password_salt text NOT NULL DEFAULT (''),
			temporary_password_hash text NOT NULL DEFAULT (''),
			temporary_password_expires_at datetime NULL,
			temporary_two_factor_auth_key_encrypted text NOT NULL DEFAULT (''),
			two_factory_auth_key_encrypted text NOT NULL DEFAULT (''),
			two_factor_auth_recovery_code_salt text NOT NULL DEFAULT (''),
			two_factor_auth_recovery_code_hashes json NOT NULL,
			last_login_attempt_at datetime NULL,
			role text NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("create legacy accounts table without passkey columns: %v", err)
	}

	_, err = rawDB.Exec(`
		INSERT INTO accounts (
			created_at,
			updated_at,
			public_id,
			email,
			first_name,
			last_name,
			language,
			password_salt,
			password_hash,
			temporary_password_salt,
			temporary_password_hash,
			temporary_two_factor_auth_key_encrypted,
			two_factory_auth_key_encrypted,
			two_factor_auth_recovery_code_salt,
			two_factor_auth_recovery_code_hashes,
			role
		) VALUES (
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP,
			'acc_test_2',
			'test2@example.com',
			'Test',
			'User',
			'English',
			'',
			'',
			'',
			'',
			'',
			'',
			'',
			'[]',
			'User'
		);
	`)
	if err != nil {
		t.Fatalf("insert legacy account row without passkey columns: %v", err)
	}

	mainDB := sqlx.NewMainDB(dbPath)
	defer func() {
		_ = mainDB.Close()
	}()

	ctx := context.Background()
	err = normalizeMainDBBeforeDevSchemaCreate(mainDB)
	if err != nil {
		t.Fatalf("normalize main db: %v", err)
	}

	err = mainDB.ReadWriteConn.Schema.Create(
		ctx,
		migratemain.WithDropIndex(true),
		migratemain.WithDropColumn(true),
	)
	if err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	accountx := mainDB.ReadWriteConn.Account.Query().OnlyX(ctx)
	if accountx.PasskeyLoginEnabled {
		t.Fatal("expected missing passkey_login_enabled to be initialized to false")
	}
	if accountx.PasskeyRecoveryCodeSalt != "" {
		t.Fatalf("expected empty passkey recovery code salt, got %q", accountx.PasskeyRecoveryCodeSalt)
	}
	if len(accountx.PasskeyRecoveryCodeHashes) != 0 {
		t.Fatalf("expected empty passkey recovery code hashes, got %d", len(accountx.PasskeyRecoveryCodeHashes))
	}
}
