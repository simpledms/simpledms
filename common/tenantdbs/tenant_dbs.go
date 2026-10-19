package tenantdbs

import (
	"log"

	"github.com/puzpuzpuz/xsync/v4"

	"github.com/simpledms/simpledms/db/sqlx"
)

type TenantDBs struct {
	*xsync.Map[int64, *sqlx.TenantDB]
}

func NewTenantDBs() *TenantDBs {
	return &TenantDBs{
		xsync.NewMap[int64, *sqlx.TenantDB](),
	}
}

// LoadOrOpen returns the open DB of a tenant and opens it at most once, even when concurrent
// first requests for the same tenant arrive. A failed open is not stored and can be retried.
func (qq *TenantDBs) LoadOrOpen(
	tenantID int64,
	open func() (*sqlx.TenantDB, error),
) (*sqlx.TenantDB, error) {
	var openErr error
	tenantDB, _ := qq.LoadOrCompute(tenantID, func() (*sqlx.TenantDB, bool) {
		openedDB, err := open()
		if err != nil {
			openErr = err
			return nil, true
		}
		return openedDB, false
	})
	if openErr != nil {
		return nil, openErr
	}
	return tenantDB, nil
}

// StoreOrClose registers a DB opened outside of LoadOrOpen. If another caller registered a DB
// for the tenant in the meantime, the given DB is closed and the registered one is returned,
// so no connection pool is leaked or replaced while in use.
func (qq *TenantDBs) StoreOrClose(tenantID int64, tenantDB *sqlx.TenantDB) *sqlx.TenantDB {
	actual, loaded := qq.LoadOrStore(tenantID, tenantDB)
	if loaded && actual != tenantDB {
		if err := tenantDB.Close(); err != nil {
			log.Println(err)
		}
	}
	return actual
}
