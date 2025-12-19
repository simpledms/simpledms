package sqlx

import (
	"log"
	"runtime"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/app/simpledms/enttenant"
)

type TenantDB struct {
	*DB[*enttenant.Client, *enttenant.Tx]
}

func NewTenantDB(readOnlyDataSourceURL, readWriteDataSourceURL string) *TenantDB {
	// read only
	readOnlyDrv, err := sql.Open(dialect.SQLite, readOnlyDataSourceURL)
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	readOnlyDrv.DB().SetMaxIdleConns(0)
	// TODO related to number of cpus? runtime.NumCPU
	//		if in doubt, set it low to prevent out of memory issue?
	readOnlyDrv.DB().SetMaxOpenConns(runtime.NumCPU()) // TODO enough?
	readOnlyConn := enttenant.NewClient(enttenant.Driver(readOnlyDrv))

	// read write
	readWriteDrv, err := sql.Open(dialect.SQLite, readWriteDataSourceURL)
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	readWriteDrv.DB().SetMaxIdleConns(0)
	readWriteDrv.DB().SetMaxOpenConns(1)
	readWriteConn := enttenant.NewClient(enttenant.Driver(readWriteDrv))

	return &TenantDB{
		DB: newDB(readOnlyConn, readWriteConn, readWriteDataSourceURL),
	}
}
