package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type TenantDataMigration struct {
	ent.Schema
}

func (TenantDataMigration) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").Immutable().NotEmpty(),
		field.Int64("cursor").Default(0),
		field.Time("first_started_at").Immutable(),
		field.Time("completed_at").Optional().Nillable(),
		field.Time("last_attempted_at").Optional().Nillable(),
		field.Time("failed_at").Optional().Nillable(),
		field.String("last_error").Optional().Nillable(),
		field.Int("retry_count").Default(0),
		field.String("lease_token").Optional().Nillable(),
		field.Time("lease_expires_at").Optional().Nillable(),
	}
}

func (TenantDataMigration) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("key").Unique(),
	}
}
