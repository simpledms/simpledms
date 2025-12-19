package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"

	"github.com/simpledms/simpledms/app/simpledms/entx"
	"github.com/simpledms/simpledms/app/simpledms/model/common/country"
)

// Tenant holds the schema definition for the Tenant entity.
type Tenant struct {
	ent.Schema
}

// Fields of the Tenant.
func (Tenant) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		// Shown in app if multiple tenants
		field.String("name"), // TODO instance_name or name? // Unique or not? not possible if it could be empty on registration; just needed if others get invited...
		// field.Enum("language").Values("English", "German"),

		// invoicing details
		field.String("first_name").Default(""),
		field.String("last_name").Default(""),
		// field.String("company_name").Default(""), // name on invoice // TODO or company_name?
		field.String("street").Default(""),
		field.String("house_number").Default(""),
		field.String("additional_address_info").Default(""),
		field.String("postal_code").Default(""),
		field.String("city").Default(""),
		// TODO state?
		field.Enum("country").GoType(country.Unknown),
		field.String("vat_id").Default(""),

		field.Time("terms_of_service_accepted"),
		field.Time("privacy_policy_accepted"),

		field.Bool("two_factor_auth_enforced").Default(false),

		// optional because done during in initialization
		field.Bytes("x25519_identity_encrypted").Optional().Sensitive().GoType(entx.EncryptedX25519Identity{}),
		field.Time("maintenance_mode_enabled_at").Optional().Nillable(),

		field.Time("initialized_at").Optional().Nillable(),
		// TODO add retries, last try, etc.?

		/*
			field.String("s3_custom_backup_endpoint"),
			field.String("s3_custom_backup_access_key_id"),
			field.String("s3_custom_backup_secret_access_key").Sensitive(),
			field.String("s3_custom_backup_bucket_name"),
			field.Bool("s3_custom_backup_use_ssl"),
		*/
	}
}

// Edges of the Tenant.
func (Tenant) Edges() []ent.Edge {
	return []ent.Edge{
		// TODO or just user_assignment?
		edge.
			From("accounts", Account.Type).
			Ref("tenants").
			Through("account_assignment", TenantAccountAssignment.Type), // TODO unique?
	}
}

func (Tenant) Mixin() []ent.Mixin {
	// TODO
	return []ent.Mixin{
		NewCommonMixin(Tenant.Type),
		NewSoftDeleteMixin(Tenant.Type),
		entx.NewPublicIDMixin(false),
	}
}
