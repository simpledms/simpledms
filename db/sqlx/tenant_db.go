package sqlx

import (
	"log"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/db/enttenant"
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
	readOnlyDrv.DB().SetMaxOpenConns(readOnlyMaxOpenConns()) // TODO enough?
	readOnlyConn := enttenant.NewClient(enttenant.Driver(newTimingDriver(readOnlyDrv)))

	// read write
	readWriteDrv, err := sql.Open(dialect.SQLite, readWriteDataSourceURL)
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	readWriteDrv.DB().SetMaxIdleConns(0)
	readWriteDrv.DB().SetMaxOpenConns(1)
	readWriteConn := enttenant.NewClient(enttenant.Driver(newTimingDriver(readWriteDrv)))

	return &TenantDB{
		DB: newDB(readOnlyConn, readWriteConn, readWriteDataSourceURL),
	}
}
