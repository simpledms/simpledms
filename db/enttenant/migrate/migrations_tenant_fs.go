package migrate

import (
	"embed"
	"io/fs"
)

//go:embed migrations
var migrationsTenantFS embed.FS

func NewMigrationsTenantFS() (fs.FS, error) {
	return fs.Sub(migrationsTenantFS, "migrations")
}
