package entx

import (
	atlasschema "ariga.io/atlas/sql/schema"
	"entgo.io/ent/dialect/sql/schema"
)

// WithInboxTransferDefault keeps the new opt-in column additive on SQLite.
// Ent v0.14.6 otherwise represents false as an expression, forcing a table rebuild.
func WithInboxTransferDefault() schema.MigrateOption {
	return schema.WithDiffHook(func(next schema.Differ) schema.Differ {
		return schema.DiffFunc(func(current, desired *atlasschema.Schema) ([]atlasschema.Change, error) {
			for _, table := range desired.Tables {
				if table.Name != "spaces" {
					continue
				}
				for _, column := range table.Columns {
					if column.Name == "accepts_inbox_transfers" {
						column.Default = &atlasschema.Literal{V: "false"}
					}
				}
			}
			return next.Diff(current, desired)
		})
	})
}
