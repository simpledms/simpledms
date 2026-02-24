package schema

import (
	"context"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/entql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/predicate"
	"github.com/simpledms/simpledms/db/entmain/privacy"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/common/mainrole"
	"github.com/simpledms/simpledms/model/common/tenantrole"
)

// named Account to differantiate from User in enttenant
type Account struct {
	ent.Schema
}

func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("email").GoType(entx.CIText("")),

		// for contact // TODO in main db or tenant db?
		// form_of_address and last_name
		// field.String("form_of_address").Optional(),
		field.String("first_name"), // TODO call_name or nickname?
		field.String("last_name"),
		field.Enum("language").GoType(language.Unknown),
		// TODO phone_number?

		field.Time("subscribed_to_newsletter_at").Nillable().Optional(),

		field.String("password_salt").Default("").Sensitive(),
		field.String("password_hash").Default("").Sensitive(),

		field.String("temporary_password_salt").Default("").Sensitive(),
		field.String("temporary_password_hash").Default("").Sensitive(),
		field.Time("temporary_password_expires_at").Default(time.Time{}).Optional(),

		// require before new users can be setup?
		field.String("temporary_two_factor_auth_key_encrypted").Default("").Sensitive(),
		field.String("two_factory_auth_key_encrypted").Default("").Sensitive(),
		field.String("two_factor_auth_recovery_code_salt").Default("").Sensitive(),
		field.Strings("two_factor_auth_recovery_code_hashes").Default([]string{}).Sensitive(),

		field.Time("last_login_attempt_at").Default(time.Time{}).Optional(),

		// field.Int64("tenant_id").Optional(),
		field.Enum("role").GoType(mainrole.User),
	}
}

func (Account) Edges() []ent.Edge {
	return []ent.Edge{
		// not required because super admin or supporters might not belong to a tenant
		edge.
			To("tenants", Tenant.Type).
			Through("tenant_assignment", TenantAccountAssignment.Type), // TODO unique?
		edge.From("received_mails", Mail.Type).Ref("receiver"),
		edge.From("temporary_files", TemporaryFile.Type).Ref("owner"),
		/*
			edge.From("tenant", Tenant.Type).
				Ref("users").
				Unique().
				Field("tenant_id"),

		*/
	}
}

func (Account) Indexes() []ent.Index {
	return []ent.Index{
		index.
			Fields("email").
			Annotations(entsql.IndexWhere("`deleted_at` is null")).
			Unique(),
	}
}

func (Account) Policy() ent.Policy {
	privacyFn := privacy.FilterFunc(func(untypedCtx context.Context, filterx privacy.Filter) error {
		ctx, ok := ctxx.MainCtx(untypedCtx)
		if !ok {
			return privacy.Skip
		}

		accessFilter, err := accountAccessFilter(untypedCtx, ctx)
		if err != nil {
			return err
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

func accountAccessFilter(untypedCtx context.Context, ctx *ctxx.MainContext) (entql.P, error) {
	now := time.Now()

	accessFilter := entql.Int64EQ(ctx.Account.ID).Field(account.FieldID)

	ownerTenantIDs, err := ownerTenantIDs(ctx, now)
	if err != nil {
		return nil, err
	}
	if len(ownerTenantIDs) > 0 {
		accessFilter = entql.Or(
			accessFilter,
			accountHasTenantAssignmentWith(
				tenantaccountassignment.IsOwningTenant(true),
				tenantaccountassignment.TenantIDIn(ownerTenantIDs...),
				tenantaccountassignment.Or(
					tenantaccountassignment.ExpiresAtIsNil(),
					tenantaccountassignment.ExpiresAtGT(now),
				),
				tenantaccountassignment.HasTenantWith(tenant.DeletedAtIsNil()),
			),
		)
	}

	tenantCtx, ok := ctxx.TenantCtx(untypedCtx)
	if ok {
		accessFilter = entql.And(
			accessFilter,
			accountHasTenantAssignmentWith(
				tenantaccountassignment.TenantID(tenantCtx.Tenant.ID),
				tenantaccountassignment.Or(
					tenantaccountassignment.ExpiresAtIsNil(),
					tenantaccountassignment.ExpiresAtGT(now),
				),
				tenantaccountassignment.HasTenantWith(
					tenant.ID(tenantCtx.Tenant.ID),
					tenant.DeletedAtIsNil(),
				),
			),
		)
	}

	return accessFilter, nil
}

func ownerTenantIDs(ctx *ctxx.MainContext, now time.Time) ([]int64, error) {
	assignments, err := ctx.MainTx.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.AccountID(ctx.Account.ID),
			tenantaccountassignment.RoleEQ(tenantrole.Owner),
			tenantaccountassignment.Or(
				tenantaccountassignment.ExpiresAtIsNil(),
				tenantaccountassignment.ExpiresAtGT(now),
			),
			tenantaccountassignment.HasTenantWith(tenant.DeletedAtIsNil()),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	tenantIDs := make([]int64, 0, len(assignments))
	seenTenantIDs := make(map[int64]struct{}, len(assignments))
	for _, assignment := range assignments {
		if _, ok := seenTenantIDs[assignment.TenantID]; ok {
			continue
		}

		seenTenantIDs[assignment.TenantID] = struct{}{}
		tenantIDs = append(tenantIDs, assignment.TenantID)
	}

	return tenantIDs, nil
}

func accountHasTenantAssignmentWith(preds ...predicate.TenantAccountAssignment) entql.P {
	return entql.HasEdgeWith(account.EdgeTenantAssignment, sqlgraph.WrapFunc(func(selector *sql.Selector) {
		for _, pred := range preds {
			pred(selector)
		}
	}))
}

func (Account) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewCommonMixin(Account.Type),
		NewSoftDeleteMixin(Account.Type),
		entx.NewPublicIDMixin(false),
	}
}
