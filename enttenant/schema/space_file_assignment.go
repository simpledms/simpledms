package schema

/*
import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TODO rename to SpaceFileAssignment
type SpaceFileAssignment struct {
	ent.Schema
}

// can later be extended with data like:
// - who has assigned/shared with space
// - how long should the file belong to space, etc.
func (SpaceFileAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("space_id"),
		field.Bool("is_root_dir").Default(false),
		field.Int64("file_id"),

		// field.Int64("parent_id").Optional(),
		field.Bool("is_in_inbox").Default(false),
	}
}

func (SpaceFileAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			To("space", Space.Type).
			Unique().
			Required().
			Field("space_id"),
		edge.To("file", File.Type).
			Unique().
			Required().
			Field("file_id"),
	}
}

func (SpaceFileAssignment) Indexes() []ent.Index {
	return []ent.Index{
		index.
			Fields("space_id", "is_root_dir").
			Annotations(entsql.IndexWhere("`is_root_dir` = true")).
			Unique(),
	}
}


*/
