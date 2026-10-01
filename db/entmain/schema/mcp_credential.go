package schema

import (
	"context"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/entql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/db/entx"
)

type MCPCredential struct {
	ent.Schema
}

func (MCPCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("account_id").Immutable(),
		field.Int64("tenant_id").Immutable(),
		field.String("space_public_id").GoType(entx.CIText("")).Immutable(),
		field.String("label").NotEmpty().MaxLen(100),
		field.Bool("is_read_only").Default(true).Immutable(),
		field.String("secret_hash").Sensitive().Immutable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("revoked_at").Optional().Nillable(),
	}
}

func (MCPCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("account", Account.Type).Field("account_id").Required().Immutable().Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("tenant", Tenant.Type).Field("tenant_id").Required().Immutable().Unique().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (MCPCredential) Indexes() []ent.Index {
	return []ent.Index{index.Fields("account_id", "created_at")}
}

func (MCPCredential) Mixin() []ent.Mixin {
	return []ent.Mixin{entx.NewPublicIDMixin(true)}
}

func (MCPCredential) Policy() ent.Policy {
	owner := privacy.FilterFunc(func(ctx context.Context, filter privacy.Filter) error {
		mainCtx, ok := ctxx.MainCtx(ctx)
		if !ok || mainCtx.Account == nil || mainCtx.IsTemporarySession {
			return privacy.Denyf("MCP credentials require a full account context")
		}
		filter.Where(entql.FieldEQ("account_id", mainCtx.Account.ID))
		return privacy.Allow
	})
	return privacy.Policy{
		Query: privacy.QueryPolicy{owner},
		Mutation: privacy.MutationPolicy{
			privacy.MutationRuleFunc(func(ctx context.Context, mutation ent.Mutation) error {
				if !mutation.Op().Is(ent.OpCreate) {
					return privacy.Skip
				}
				mainCtx, ok := ctxx.MainCtx(ctx)
				accountID, exists := mutation.Field("account_id")
				if !ok || mainCtx.Account == nil || mainCtx.IsTemporarySession ||
					!exists || accountID != mainCtx.Account.ID {
					return privacy.Denyf("cannot create another account's MCP credential")
				}
				return privacy.Allow
			}),
			owner,
		},
	}
}
