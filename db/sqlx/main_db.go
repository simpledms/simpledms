package sqlx

import (
	"fmt"
	"log"
	"runtime"

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
	readOnlyDrv.DB().SetMaxIdleConns(0)
	readOnlyDrv.DB().SetMaxOpenConns(runtime.NumCPU()) // TODO enough?
	readOnlyConn := entmain.NewClient(entmain.Driver(newTimingDriver(readOnlyDrv)))

	// read write
	readWriteDataSourceURL := fmt.Sprintf("file:%s?%s", dbPath, SQLiteQueryParamsReadWrite)
	readWriteDrv, err := sql.Open(dialect.SQLite, readWriteDataSourceURL)
	if err != nil {
		log.Fatalf("failed opening connection to sqlite: %v", err)
	}
	readWriteDrv.DB().SetMaxIdleConns(0)
	readWriteDrv.DB().SetMaxOpenConns(1)
	readWriteConn := entmain.NewClient(entmain.Driver(newTimingDriver(readWriteDrv)))

	return &MainDB{
		DB: newDB(readOnlyConn, readWriteConn, readWriteDataSourceURL),
	}
}
