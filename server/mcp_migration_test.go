package server

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	migratex "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/simpledms/simpledms/db/entmain"
	migratemain "github.com/simpledms/simpledms/db/entmain/migrate"
	"github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/language"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
)

func TestMCPConnectionMigrationFreshAndPopulated(t *testing.T) {
	const version = uint(20260918225936)
	migrations, err := migratemain.NewMigrationsMainFS()
	if err != nil {
		t.Fatal(err)
	}
	script, err := fs.ReadFile(migrations, "20260918225936_mcp_credentials.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"ALTER TABLE", "DROP TABLE", "INSERT INTO", "web_dav"} {
		if strings.Contains(strings.ToUpper(string(script)), strings.ToUpper(unwanted)) {
			t.Fatalf("MCP migration changes unrelated existing data: %s", unwanted)
		}
	}
	for _, populated := range []bool{false, true} {
		name := "fresh"
		if populated {
			name = "populated"
		}
		t.Run(name, func(t *testing.T) {
			source, err := iofs.New(migrations, ".")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "main.sqlite3")
			migration, err := migratex.NewWithSourceInstance("migrationsMainFS", source,
				"sqlite3://"+path+"?_foreign_keys=0")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _, _ = migration.Close() })
			client, err := entmain.Open("sqlite3", path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = client.Close() })
			ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
			var actor *entmain.Account
			if populated {
				previous, err := source.Prev(version)
				if err != nil {
					t.Fatal(err)
				}
				if err := migration.Migrate(previous); err != nil {
					t.Fatal(err)
				}
				actor = client.Account.Create().SetEmail(entx.NewCIText("existing@example.com")).
					SetFirstName("Existing").SetLastName("Account").
					SetLanguage(language.English).SetRole(mainrole.User).SaveX(ctx)
			}
			if err := migration.Migrate(version); err != nil {
				t.Fatal(err)
			}
			if client.MCPCredential.Query().CountX(ctx) != 0 {
				t.Fatal("migration fabricated credentials")
			}
			if actor != nil && client.Account.GetX(ctx, actor.ID).Email != actor.Email {
				t.Fatal("migration changed existing account")
			}
		})
	}
}
