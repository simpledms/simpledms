package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLiteQueryParamsPreserveCachedQueryCorrectness(t *testing.T) {
	for _, test := range []struct {
		name   string
		params string
	}{
		{name: "read-write", params: SQLiteQueryParamsReadWrite},
		{name: "read-only", params: SQLiteQueryParamsReadOnly},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "params.sqlite")
			writer := openSQLiteTestDB(t, fmt.Sprintf("file:%s?%s", path, SQLiteQueryParamsReadWrite))
			if _, err := writer.Exec(`CREATE TABLE records (id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Exec(`INSERT INTO records VALUES (1, 'first')`); err != nil {
				t.Fatal(err)
			}

			queryDB := openSQLiteTestDB(t, fmt.Sprintf("file:%s?%s", path, test.params))

			const query = `SELECT value FROM records WHERE id = ?`
			assertCachedQueryValue(t, queryDB, query, 1, "first") // warms the driver's cache

			if _, err := writer.Exec(`INSERT INTO records VALUES (2, 'committed')`); err != nil {
				t.Fatal(err)
			}
			assertCachedQueryValue(t, queryDB, query, 2, "committed")

			tx, err := writer.Begin()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(`UPDATE records SET value = 'rolled back' WHERE id = 1`); err != nil {
				t.Fatal(err)
			}
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			assertCachedQueryValue(t, queryDB, query, 1, "first")

			if _, err := writer.Exec(`ALTER TABLE records ADD COLUMN note TEXT DEFAULT 'added'`); err != nil {
				t.Fatal(err)
			}
			assertCachedQueryValue(t, queryDB, query, 1, "first")
			if _, err := writer.Exec(`UPDATE records SET value = 'after alter' WHERE id = ?`, 1); err != nil {
				t.Fatal(err)
			}
			assertCachedQueryValue(t, queryDB, query, 1, "after alter")
		})
	}
}

func TestSQLiteQueryParamsRefreshCachedTableMetadata(t *testing.T) {
	for _, test := range []struct {
		name   string
		params string
	}{
		{name: "read-write", params: SQLiteQueryParamsReadWrite},
		{name: "read-only", params: SQLiteQueryParamsReadOnly},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "metadata.sqlite")
			writer := openSQLiteTestDB(t, fmt.Sprintf("file:%s?%s", path, SQLiteQueryParamsReadWrite))
			if _, err := writer.Exec(`CREATE TABLE records (id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Exec(`INSERT INTO records VALUES (1, 'first')`); err != nil {
				t.Fatal(err)
			}

			reader := openSQLiteTestDB(t, fmt.Sprintf("file:%s?%s", path, test.params))
			conn, err := reader.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := conn.Close(); err != nil {
					t.Errorf("close reader connection: %v", err)
				}
			})

			const query = `SELECT * FROM records`
			assertSQLiteQueryResult(t, conn, query, []string{"id", "value"}, []string{"1", "first"})
			if _, err := writer.Exec(`ALTER TABLE records ADD COLUMN note TEXT DEFAULT 'added'`); err != nil {
				t.Fatal(err)
			}
			wantColumns := []string{"id", "value", "note"}
			wantValues := []string{"1", "first", "added"}
			assertSQLiteQueryResult(t, conn, query, wantColumns, wantValues)
			assertSQLiteQueryResult(t, conn, query, wantColumns, wantValues)
		})
	}
}

func TestSQLiteQueryParamsRefreshCachedViewMetadata(t *testing.T) {
	for _, test := range []struct {
		name   string
		params string
	}{
		{name: "read-write", params: SQLiteQueryParamsReadWrite},
		{name: "read-only", params: SQLiteQueryParamsReadOnly},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "view-metadata.sqlite")
			writer := openSQLiteTestDB(t, fmt.Sprintf("file:%s?%s", path, SQLiteQueryParamsReadWrite))
			if _, err := writer.Exec(`CREATE TABLE records (id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Exec(`INSERT INTO records VALUES (1, 'first')`); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Exec(`CREATE VIEW record_view AS SELECT id AS old_id, value FROM records`); err != nil {
				t.Fatal(err)
			}

			reader := openSQLiteTestDB(t, fmt.Sprintf("file:%s?%s", path, test.params))
			conn, err := reader.Conn(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := conn.Close(); err != nil {
					t.Errorf("close reader connection: %v", err)
				}
			})

			const query = `SELECT * FROM record_view`
			assertSQLiteQueryResult(t, conn, query, []string{"old_id", "value"}, []string{"1", "first"})
			if _, err := writer.Exec(`DROP VIEW record_view`); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Exec(`CREATE VIEW record_view AS SELECT id AS new_id, value FROM records`); err != nil {
				t.Fatal(err)
			}
			wantColumns := []string{"new_id", "value"}
			wantValues := []string{"1", "first"}
			assertSQLiteQueryResult(t, conn, query, wantColumns, wantValues)
			assertSQLiteQueryResult(t, conn, query, wantColumns, wantValues)
		})
	}
}

func assertSQLiteQueryResult(
	t *testing.T,
	conn *sql.Conn,
	query string,
	wantColumns, wantValues []string,
) {
	t.Helper()
	rows, err := conn.QueryContext(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("close query rows: %v", err)
		}
	}()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(columns, wantColumns) {
		t.Fatalf("columns = %v, want %v", columns, wantColumns)
	}
	values := make([]any, len(columns))
	scanTargets := make([]any, len(columns))
	for i := range values {
		scanTargets[i] = &values[i]
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		t.Fatal("query returned no rows")
	}
	if err := rows.Scan(scanTargets...); err != nil {
		t.Fatal(err)
	}
	gotValues := make([]string, len(values))
	for i, value := range values {
		gotValues[i] = fmt.Sprint(value)
	}
	if rows.Next() {
		t.Fatal("query returned more than one row")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(gotValues, wantValues) {
		t.Fatalf("values = %v, want %v", gotValues, wantValues)
	}
}

func openSQLiteTestDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return db
}

func assertCachedQueryValue(t *testing.T, db *sql.DB, query string, id int, want string) {
	t.Helper()
	var got string
	if err := db.QueryRowContext(context.Background(), query, id).Scan(&got); err != nil {
		t.Fatalf("query id %d: %v", id, err)
	}
	if got != want {
		t.Fatalf("query id %d = %q, want %q", id, got, want)
	}
}
