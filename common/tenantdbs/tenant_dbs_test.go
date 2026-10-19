package tenantdbs

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/db/sqlx"
)

func newTestTenantDB(t *testing.T) *sqlx.TenantDB {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "tenant.sqlite3")
	tenantDB := sqlx.NewTenantDB(dsn, dsn)
	t.Cleanup(func() {
		_ = tenantDB.Close()
	})
	return tenantDB
}

func isClosed(tenantDB *sqlx.TenantDB) bool {
	tx, err := tenantDB.Tx(context.Background(), false)
	if err != nil {
		return true
	}
	_ = tx.Rollback()
	return false
}

func TestLoadOrOpenOpensTenantDBOnceForConcurrentFirstRequests(t *testing.T) {
	tenantDBs := NewTenantDBs()
	opened := newTestTenantDB(t)
	var openCalls atomic.Int32

	const requests = 32
	results := make([]*sqlx.TenantDB, requests)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range requests {
		wg.Go(func() {
			<-start
			tenantDB, err := tenantDBs.LoadOrOpen(1, func() (*sqlx.TenantDB, error) {
				openCalls.Add(1)
				return opened, nil
			})
			if err != nil {
				t.Errorf("LoadOrOpen: %v", err)
			}
			results[i] = tenantDB
		})
	}
	close(start)
	wg.Wait()

	if got := openCalls.Load(); got != 1 {
		t.Fatalf("open calls = %d, want 1", got)
	}
	for i, tenantDB := range results {
		if tenantDB != opened {
			t.Fatalf("request %d got a different tenant DB", i)
		}
	}
}

func TestLoadOrOpenDoesNotStoreFailedOpenAndRetries(t *testing.T) {
	tenantDBs := NewTenantDBs()
	openErr := errors.New("open failed")

	tenantDB, err := tenantDBs.LoadOrOpen(1, func() (*sqlx.TenantDB, error) {
		return nil, openErr
	})
	if !errors.Is(err, openErr) || tenantDB != nil {
		t.Fatalf("LoadOrOpen = (%v, %v), want (nil, %v)", tenantDB, err, openErr)
	}
	if _, found := tenantDBs.Load(1); found {
		t.Fatal("failed open must not be stored")
	}

	opened := newTestTenantDB(t)
	tenantDB, err = tenantDBs.LoadOrOpen(1, func() (*sqlx.TenantDB, error) {
		return opened, nil
	})
	if err != nil || tenantDB != opened {
		t.Fatalf("retry LoadOrOpen = (%v, %v), want opened DB", tenantDB, err)
	}
}

func TestStoreOrCloseKeepsRegisteredTenantDBAndClosesDuplicate(t *testing.T) {
	tenantDBs := NewTenantDBs()
	registered := newTestTenantDB(t)
	duplicate := newTestTenantDB(t)

	if got := tenantDBs.StoreOrClose(1, registered); got != registered {
		t.Fatal("first StoreOrClose must register the given DB")
	}
	if got := tenantDBs.StoreOrClose(1, duplicate); got != registered {
		t.Fatal("StoreOrClose must return the already registered DB")
	}
	if !isClosed(duplicate) {
		t.Fatal("duplicate DB must be closed")
	}
	if isClosed(registered) {
		t.Fatal("registered DB must stay open")
	}
}
