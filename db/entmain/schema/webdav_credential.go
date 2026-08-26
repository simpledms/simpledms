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

type WebDAVCredential struct {
	ent.Schema
}

func (WebDAVCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("account_id").Immutable(),
		field.Int64("tenant_id").Immutable(),
		field.String("space_public_id").
			GoType(entx.CIText("")).
			Immutable(),
		field.String("label").NotEmpty(),
		field.String("username").Unique().Immutable(),
		field.String("secret_salt").Sensitive().Immutable(),
		field.String("secret_hash").Sensitive().Immutable(),
		field.Time("last_used_at").Optional().Nillable(),
		field.Time("revoked_at").Optional().Nillable(),
		field.Int64("revoked_by_account_id").Optional().Nillable(),
	}
}

func (WebDAVCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("account", Account.Type).
			Field("account_id").
			Required().
			Immutable().
			Unique(),
		edge.To("tenant", Tenant.Type).
			Field("tenant_id").
			Required().
			Immutable().
			Unique(),
		edge.To("revoked_by_account", Account.Type).
			Field("revoked_by_account_id").
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Unique(),
	}
}

func (WebDAVCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id", "tenant_id", "space_public_id", "revoked_at"),
		index.Fields("tenant_id", "account_id", "revoked_at"),
		index.Fields("tenant_id", "space_public_id", "label").Unique(),
	}
}

func (WebDAVCredential) Policy() ent.Policy {
	privacyFn := privacy.FilterFunc(func(untypedCtx context.Context, filterx privacy.Filter) error {
		ctx, ok := ctxx.MainCtx(untypedCtx)
		if !ok {
			return privacy.Skip
		}

		accessFilter := entql.FieldEQ("account_id", ctx.Account.ID)
		ownerTenantIDs, err := ownerTenantIDs(ctx, time.Now())
		if err != nil {
			return err
		}
		if len(ownerTenantIDs) > 0 {
			values := make([]any, 0, len(ownerTenantIDs))
			for _, tenantID := range ownerTenantIDs {
				values = append(values, tenantID)
			}
			accessFilter = entql.Or(accessFilter, entql.FieldIn("tenant_id", values...))
		}

		filterx.Where(accessFilter)

		return privacy.Skip
	})

	return privacy.Policy{
		Mutation: privacy.MutationPolicy{
			privacyFn,
		},
		Query: privacy.QueryPolicy{
			privacyFn,
		},
	}
}

func (WebDAVCredential) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewCommonMixin(WebDAVCredential.Type),
		entx.NewPublicIDMixin(true),
	}
}
