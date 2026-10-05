package sqlx

import (
	"fmt"
	"log"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/db/entmain"
)

type MainDB struct {
	*DB[*entmain.Client, *entmain.Tx]
}

func NewMainDB(dbPath string) *MainDB {
	// read only
	readOnlyDataSourceURL := fmt.Sprintf("file:%s?%s", dbPath, SQLiteQueryParamsReadOnly)
	readOnlyDrv, err := sql.Open(dialect.SQLite, readOnlyDataSourceURL)
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	configureReadOnlyPool(readOnlyDrv.DB())
	readOnlyConn := entmain.NewClient(entmain.Driver(newTimingDriver(readOnlyDrv)))

	// read write
	readWriteDataSourceURL := fmt.Sprintf("file:%s?%s", dbPath, SQLiteQueryParamsReadWrite)
	readWriteDrv, err := sql.Open(dialect.SQLite, readWriteDataSourceURL)
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	configureReadWritePool(readWriteDrv.DB())
	readWriteConn := entmain.NewClient(entmain.Driver(newTimingDriver(readWriteDrv)))

	return &MainDB{
		DB: newDB(readOnlyConn, readWriteConn, readWriteDataSourceURL),
	}
}
