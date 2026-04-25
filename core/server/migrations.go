package server

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"strings"

	"entgo.io/ent/dialect"
	"github.com/cyphar/filepath-securejoin"
	migrate2 "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/simpledms/simpledms/core/db/entmain/migrate"

	"github.com/simpledms/simpledms/core/db/sqlx"
)

// caller has to close db
func DBMigrationsMainDB(isDevMode bool, metaPath string, migrationsMainFS fs.FS) *sqlx.MainDB {
	return dbMigrationsMainDB(isDevMode, metaPath, migrationsMainFS)
}

func dbMigrationsMainDB(isDevMode bool, metaPath string, migrationsMainFS fs.FS) *sqlx.MainDB {
	mainDBPath, err := securejoin.SecureJoin(metaPath, "main.sqlite3")
	if err != nil {
		log.Fatalln(err)
	}
	mainDB := sqlx.NewMainDB(mainDBPath)

	if isDevMode {
		mainDB.Debug()
		log.Println("Running in development mode")

		// if auto migration fails because of foreign key constraint violation,
		// just create migration scripts and execute them manually
		//
		// see longer comment in model.Tenant.ExecuteDBMigrations

		/*if err := normalizeMainDBBeforeDevSchemaCreate(mainDB); err != nil {
			log.Fatalf("failed normalizing main db before schema migration: %v", err)
		}*/

		if err := mainDB.ReadWriteConn.Schema.Create(
			context.Background(),
			migrate.WithDropIndex(true),
			migrate.WithDropColumn(true),
		); err != nil {
			log.Fatalf("failed creating schema resources: %v", err)
		}

	} else {
		mainDrv, err := iofs.New(migrationsMainFS, ".")
		if err != nil {
			log.Fatalf("failed reading migration filesystem: %v", err)
		}
		defer func() {
			if err = mainDrv.Close(); err != nil {
				log.Println(err)
			}
		}()

		readWriteDataSourceURL := mainDB.ReadWriteDataSourceURL()
		// necessary because `PRAGMA foreign_keys = off` doesn't work in many
		// circumstances
		readWriteDataSourceURL = strings.Replace(
			readWriteDataSourceURL,
			"_foreign_keys=1",
			"_foreign_keys=0",
			1,
		)

		mainMigx, err := migrate2.NewWithSourceInstance(
			"migrationsMainFS",
			mainDrv,
			dialect.SQLite+"://"+readWriteDataSourceURL,
		)
		if err != nil {
			log.Fatalf("failed loading migration instance: %v", err)
		}

		err = mainMigx.Up()
		if err != nil && !errors.Is(err, migrate2.ErrNoChange) {
			log.Fatalf("failed running migrations up: %v", err)
		}

		srcErr, dbErr := mainMigx.Close()
		if srcErr != nil || dbErr != nil {
			log.Fatalf("failed closing migration instance: %v, %v", srcErr, dbErr)
		}
	}

	return mainDB
}
