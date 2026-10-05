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
	configureReadOnlyPool(readOnlyDrv.DB())
	readOnlyConn := enttenant.NewClient(enttenant.Driver(newTimingDriver(readOnlyDrv)))

	// read write
	readWriteDrv, err := sql.Open(dialect.SQLite, readWriteDataSourceURL)
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	configureReadWritePool(readWriteDrv.DB())
	readWriteConn := enttenant.NewClient(enttenant.Driver(newTimingDriver(readWriteDrv)))

	return &TenantDB{
		DB: newDB(readOnlyConn, readWriteConn, readWriteDataSourceURL),
	}
}
