package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type WebAuthnChallenge struct {
	ent.Schema
}

func (WebAuthnChallenge) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("challenge_id").Unique(),
		field.Int64("account_id").Optional().Nillable(),
		field.String("client_key").Optional().Nillable(),
		field.String("ceremony"),
		field.Bytes("session_data_json").Sensitive(),
		field.Time("expires_at"),
		field.Time("used_at").Optional().Nillable(),
		field.Time("created_at").Immutable().Default(time.Now),
	}
}

func (WebAuthnChallenge) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("account", Account.Type).
			Unique().
			Field("account_id"),
	}
}

func (WebAuthnChallenge) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("expires_at"),
		index.Fields("account_id", "ceremony"),
		index.Fields("client_key", "ceremony"),
	}
}
