package tenantdbs

import (
	"github.com/puzpuzpuz/xsync/v4"

	"github.com/simpledms/simpledms/app/simpledms/sqlx"
)

type TenantDBs struct {
	*xsync.Map[int64, *sqlx.TenantDB]
}

func NewTenantDBs() *TenantDBs {
	return &TenantDBs{
		xsync.NewMap[int64, *sqlx.TenantDB](),
	}
}

/*
func (qq *TenantDBs) Add(id int64, tenantDB *enttenant.Client) {
	if _, found := qq[id]; found {
		// TODO okay?
		panic("tenant db already exists in map, should never happen")
	}
	qq[id] = tenantDB
}

func (qq *TenantDBs) Get(id int64) (*enttenant.Client, bool) {
	tenantDB, found := qq[id]
	return tenantDB, found
}

*/
