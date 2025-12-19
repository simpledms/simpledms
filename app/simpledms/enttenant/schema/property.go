package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/app/simpledms/model/common/fieldtype"
)

// TODO Rename to Field
type Property struct {
	ent.Schema
}

func (Property) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),

		field.String("name"),
		field.Enum("type").GoType(fieldtype.Unknown),

		field.String("unit").Default(""),
	}
}

func (Property) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			From("files", File.Type).
			Ref("properties").
			Through("file_assignments", FilePropertyAssignment.Type),
	}
}

func (Property) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("space_id", "name").Unique(),
	}
}

func (Property) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewSpaceMixin(),
	}
}
