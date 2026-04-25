package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/core/db/entx"
)

type PasskeyCredential struct {
	ent.Schema
}

func (PasskeyCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("account_id").Immutable(),
		field.Bytes("credential_id").Unique().Sensitive(),
		field.Bytes("credential_json").Sensitive(),
		field.String("name").Default(""),
		field.Time("last_used_at").Optional().Nillable(),
	}
}

func (PasskeyCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("account", Account.Type).
			Unique().
			Required().
			Immutable().
			Field("account_id"),
	}
}

func (PasskeyCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id"),
	}
}

func (PasskeyCredential) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewCommonMixin(PasskeyCredential.Type),
		entx.NewPublicIDMixin(false),
	}
}
