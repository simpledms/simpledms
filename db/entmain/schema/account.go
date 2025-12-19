package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/common/mainrole"
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

func (Account) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewCommonMixin(Account.Type),
		NewSoftDeleteMixin(Account.Type),
		entx.NewPublicIDMixin(false),
	}
}
