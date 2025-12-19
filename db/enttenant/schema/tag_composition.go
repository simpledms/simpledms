package schema

/*
import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type TagComposition struct {
	ent.Schema
}

func (TagComposition) Annotations() []schema.Annotation {
	return []schema.Annotation{
		field.ID("tag_id", "sub_tag_id"),
	}
}

func (TagComposition) Fields() []ent.Field {
	return []ent.Field{
		// field.Int64("id"),
		field.Int64("tag_id"),
		// field.Int64("super_tag_id"),
		field.Int64("sub_tag_id"),
	}
}

func (TagComposition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			To("tag", Tag.Type).
			Unique().
			Required().
			Field("tag_id"),
		edge.
			To("sub_tag", Tag.Type).
			Unique().
			Required().
			Field("sub_tag_id"),
	}
}


*/
