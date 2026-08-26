package entx

import (
	atlasschema "ariga.io/atlas/sql/schema"
	"entgo.io/ent/dialect/sql/schema"
)

// WithFileSourceDefault works around Ent v0.14.6 generating invalid Go for
// defaults on integer-backed custom enums.
func WithFileSourceDefault() schema.MigrateOption {
	return schema.WithDiffHook(func(next schema.Differ) schema.Differ {
		return schema.DiffFunc(func(current, desired *atlasschema.Schema) ([]atlasschema.Change, error) {
			setFileSourceDefaults(desired)
			return next.Diff(current, desired)
		})
	})
}

func setFileSourceDefaults(desired *atlasschema.Schema) {
	for _, table := range desired.Tables {
		if table.Name != "files" && table.Name != "temporary_files" {
			continue
		}
		for _, column := range table.Columns {
			if column.Name == "source" {
				column.Default = &atlasschema.Literal{V: "UnknownLegacy"}
			}
		}
	}
}
