package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Session struct {
	ent.Schema
}

func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("value").Unique().Sensitive(),
		field.Int64("account_id").Immutable(),

		field.Bool("is_temporary_session"),
		field.Time("expires_at"), // can be zero value for temporary sessions

		// necessary to cleanup database
		field.Time("deletable_at"),

		field.Time("created_at").Immutable().Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("account", Account.Type).
			Unique().
			Required().
			Immutable().
			Field("account_id"),
	}
}
