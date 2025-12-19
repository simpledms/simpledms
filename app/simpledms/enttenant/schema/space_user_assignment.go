package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/app/simpledms/model/common/spacerole"
)

type SpaceUserAssignment struct {
	ent.Schema
}

func (SpaceUserAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		// field.Int64("space_id"),
		field.Int64("user_id"),
		// https://chatgpt.com/c/67ca9aba-4698-8000-8f36-7a6ae65e2914
		field.Enum("role").GoType(spacerole.User), // TODO add Guest?

		field.Bool("is_default").Default(false),
	}
}

func (SpaceUserAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		/*edge.
		To("space", Space.Type).
		Unique().
		Required().
		Field("space_id"),*/
		edge.
			To("user", User.Type).
			Unique().
			Required().
			Field("user_id"),
	}
}

func (SpaceUserAssignment) Indexes() []ent.Index {
	return []ent.Index{
		index.
			Fields("space_id", "user_id").
			Unique(),
		index.
			Fields("user_id", "is_default").
			Annotations(entsql.IndexWhere("`is_default` = true")).
			Unique(),
	}
}

func (SpaceUserAssignment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewSpaceMixin(),
		NewCommonMixin(SpaceUserAssignment.Type),
	}
}
