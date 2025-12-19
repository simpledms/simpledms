package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"ariga.io/atlas/sql/sqltool"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	_ "github.com/mattn/go-sqlite3"

	migratemain "github.com/simpledms/simpledms/entmain/migrate"
	migratetenant "github.com/simpledms/simpledms/enttenant/migrate"
	"github.com/simpledms/simpledms/sqlx"
)

// run with:
// CGO_ENABLED=1 go run ./cmd/migrate/main.go v1.0.0-beta.x
// cgo just necessary as long as using old driver
//
// there are some issues with .down. files, which can lead to a `checksum mismatch` error;
// a workaround for these errors is to:
// 1. delete all down files
// 2. calculate new hash with atlas-update-hash.sh
// 3. run this script
// 4. restore .down. files
func main() {
	if len(os.Args) != 2 {
		log.Fatalln("migration name is required. Use: 'go run cmd/migrate/main.go <name>'")
	}

	ctx := context.Background()

	// Create a local migration directory able to understand Atlas migration file format for replay.
	// dir, err := atlas.NewLocalDir("ent/migrate/migrations")
	// TODO write directly to migrationsFS? possible?
	tenantDir, err := sqltool.NewGolangMigrateDir("enttenant/migrate/migrations")
	if err != nil {
		log.Fatalf("failed creating atlas migration directory: %v", err)
	}

	optsTenant := []schema.MigrateOption{
		schema.WithDir(tenantDir),                   // provide migration directory
		schema.WithMigrationMode(schema.ModeReplay), // provide migration mode
		schema.WithDialect(dialect.SQLite),          // Ent dialect to use
		schema.WithDropIndex(true),
		schema.WithDropColumn(true),
		// important that disabled when GolangMigrateDir is used:
		// schema.WithFormatter(atlas.DefaultFormatter),
	}
	err = migratetenant.NamedDiff(
		ctx,
		fmt.Sprintf("sqlite://migrate.sqlite3?mode=memory%s", sqlx.SQLiteQueryParamsReadWrite),
		os.Args[1],
		optsTenant...,
	)
	if err != nil {
		log.Fatalf("failed generating enttenant migration file: %v", err)
	}

	mainDir, err := sqltool.NewGolangMigrateDir("entmain/migrate/migrations")
	// dir2, err := sqltool.NewGolangMigrateDir("sql/migrations-main")
	if err != nil {
		log.Fatalf("failed creating atlas migration directory: %v", err)
	}
	optsMain := []schema.MigrateOption{
		schema.WithDir(mainDir),                     // provide migration directory
		schema.WithMigrationMode(schema.ModeReplay), // provide migration mode
		schema.WithDialect(dialect.SQLite),          // Ent dialect to use
		schema.WithDropIndex(true),
		schema.WithDropColumn(true),
		// important that disabled when GolangMigrateDir is used:
		// schema.WithFormatter(atlas.DefaultFormatter),
	}
	err = migratemain.NamedDiff(
		ctx,
		fmt.Sprintf("sqlite://migrate.sqlite3?mode=memory%s", sqlx.SQLiteQueryParamsReadWrite),
		os.Args[1],
		optsMain...,
	)
	if err != nil {
		log.Fatalf("failed generating entmain migration file: %v", err)
	}
}
