package migrate

import (
	"embed"
	"io/fs"
)

//go:embed migrations
var migrationsMainFS embed.FS

func NewMigrationsMainFS() (fs.FS, error) {
	return fs.Sub(migrationsMainFS, "migrations")
}
