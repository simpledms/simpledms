package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Mail struct {
	ent.Schema
}

func (Mail) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),

		field.String("subject"),
		field.String("body"),
		field.String("html_body").Optional(),

		field.Time("sent_at").Optional().Nillable(),

		field.Time("last_tried_at").Default(time.Time{}),
		field.Int("retry_count").Default(0),

		field.Int64("receiver_id"),
	}
}

func (Mail) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("receiver", Account.Type).
			Unique().
			Required().
			Field("receiver_id"),
	}
}

func (Mail) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewCommonMixin(Mail.Type),
	}
}
