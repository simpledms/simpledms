package main

import (
	"errors"
	"flag"
	"log"
	"os"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"

	// necessary to prevent circular dependencies when interceptors, hooks or privacy policies
	// are used; not tested, just taken from ent docs
	_ "github.com/simpledms/simpledms/db/entmain/runtime"
	_ "github.com/simpledms/simpledms/db/enttenant/runtime"
	"github.com/simpledms/simpledms/server"
	"github.com/simpledms/simpledms/ui/uix"
)

/*//go:generate ent generate simpledms/db/entx/schema/ --feature intercept,schema/snapshot,sql/versioned-migration,sql/modifier,sql/execquery --template ./enttmpl*/
//go:generate ent generate ./db/enttenant/schema/ --target ./db/enttenant --feature intercept,entql,privacy,schema/snapshot,sql/versioned-migration,sql/modifier,sql/execquery --template ./db/enttmpl
//go:generate ent generate ./db/entmain/schema/ --target ./db/entmain/ --feature intercept,entql,privacy,schema/snapshot,sql/versioned-migration,sql/modifier,sql/execquery --template ./db/enttmpl
func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	var port int
	// var dbPath string
	var devMode bool
	var metaPath string
	flag.IntVar(&port, "port", 443, "Port to listen on")
	flag.StringVar(&metaPath, "meta", "simpledms", "Path to the data directory for simpledms, for example the database is stored in there and all files if a local storage driver is used. Relative to the served directory.")
	flag.BoolVar(&devMode, "dev", false, "Run in development mode")
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Println("no .env file found, using default values")
		} else {
			log.Fatalln(err)
		}
	}

	assetsFS, err := uix.NewAssetsFS()
	if err != nil {
		log.Fatalln(err)
	}

	serverx := server.NewServer(
		metaPath,
		devMode,
		port,
		assetsFS,
		false,
	)
	err = serverx.Start()
	if err != nil {
		log.Println(err)
	}
}
