package tenant

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/sqlx"
)

func dropTenantViews(ctx context.Context, tenantDB *sqlx.TenantDB) error {
	// SQLite validates dependent views while Atlas replaces tables. FTS is an external
	// content table, not a view: retain it unless its content table loses its triggers.
	_, err := tenantDB.ReadWriteConn.ExecContext(ctx, `
DROP VIEW IF EXISTS file_infos;
DROP VIEW IF EXISTS resolved_tag_assignments;`)
	return err
}

func ensureTenantViews(ctx context.Context, tenantDB *sqlx.TenantDB) error {
	tx, err := tenantDB.ReadWriteConn.Tx(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				log.Println(err)
			}
		}
	}()
	client := tx.Client()
	definitions, err := tenantViewDefinitions(ctx, client)
	if err != nil {
		return err
	}
	resolved := schema.ResolvedTagAssignment{}
	if normalizedSQL(definitions["resolved_tag_assignments"]) != normalizedSQL(resolved.CreateSQL()) {
		if _, err := client.ExecContext(ctx, resolved.SQL()); err != nil {
			return fmt.Errorf("restore resolved Tag view: %w", err)
		}
	}

	search := schema.FileSearch{}
	triggers := search.TriggersSQL()
	shouldRebuild := normalizedSQL(definitions["file_searches"]) != normalizedSQL(search.TableSQL())
	for name := range triggers {
		// A missing trigger may have missed writes, including during a table rebuild.
		shouldRebuild = shouldRebuild || definitions[name] == ""
	}
	if shouldRebuild {
		if _, err := client.ExecContext(ctx, search.SQL()); err != nil {
			return fmt.Errorf("rebuild file search index: %w", err)
		}
	} else {
		for _, name := range []string{"files_ai", "files_ad", "files_au"} {
			if normalizedSQL(definitions[name]) == normalizedSQL(triggers[name]) {
				continue
			}
			// Upgrade existing triggers without rebuilding an already synchronized index.
			if _, err := client.ExecContext(ctx, "DROP TRIGGER "+name+";\n"+triggers[name]); err != nil {
				return fmt.Errorf("update file search trigger %s: %w", name, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func tenantViewDefinitions(ctx context.Context, client *enttenant.Client) (map[string]string, error) {
	rows, err := client.QueryContext(ctx, `SELECT name, sql FROM sqlite_master
WHERE name IN ('resolved_tag_assignments', 'file_searches', 'files_ai', 'files_ad', 'files_au')`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
		}
	}()
	definitions := make(map[string]string)
	for rows.Next() {
		var name, definition string
		if err := rows.Scan(&name, &definition); err != nil {
			return nil, err
		}
		definitions[name] = definition
	}
	return definitions, rows.Err()
}

func normalizedSQL(query string) string {
	return strings.TrimSuffix(strings.Join(strings.Fields(query), " "), ";")
}
