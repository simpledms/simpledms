package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type DocumentType struct {
	ent.Schema
}

func (DocumentType) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),

		field.String("name"),
		field.String("icon").Optional(),

		field.Bool("is_protected").Default(false),
		field.Bool("is_disabled").Default(false), // mainly if protected
	}
}

func (DocumentType) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			To("attributes", Attribute.Type),
	}
}

func (DocumentType) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewSpaceMixin(),
		// entcommon.NewPublicIDMixin(true),
	}
}
