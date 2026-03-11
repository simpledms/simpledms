package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/model/common/tenantrole"
)

type TenantAccountAssignment struct {
	ent.Schema
}

func (TenantAccountAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("tenant_id"),
		field.Int64("account_id"),
		field.Bool("is_contact_person").Default(false),
		// for example used for:
		// - redirect after login and
		// - for shared target api and
		// - open with
		field.Bool("is_default").Default(false),
		field.Bool("is_owning_tenant").Default(false),
		field.Enum("role").GoType(tenantrole.User),
		// can be used to invite a supporter
		field.Time("expires_at").Optional().Nillable(), // TODO impl filter similar to deleted at
	}
}

func (TenantAccountAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			To("tenant", Tenant.Type).
			Unique().
			Required().
			Field("tenant_id"),
		edge.
			To("account", Account.Type).
			Unique().
			Required().
			Field("account_id"),
	}
}

func (TenantAccountAssignment) Indexes() []ent.Index {
	return []ent.Index{
		index.
			Fields("tenant_id", "account_id").
			Unique(),
		index.
			Fields("account_id", "is_default").
			Annotations(entsql.IndexWhere("`is_default` = true")).
			Unique(),
		index.
			Fields("account_id", "is_owning_tenant").
			Annotations(entsql.IndexWhere("`is_owning_tenant` = true")).
			Unique(),
	}
}

func (TenantAccountAssignment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewCommonMixin(TenantAccountAssignment.Type),
	}
}
