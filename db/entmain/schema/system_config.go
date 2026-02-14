package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"

	"github.com/simpledms/simpledms/db/entx"
)

// Config conflicts with ent, thus SystemConfig
type SystemConfig struct {
	ent.Schema
}

func (SystemConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		// field.Bool("is_dev_mode").Default(false), // TODO not sure if good idea? is it permanent or per start? thus a cli flag?

		field.Bytes("x25519_identity").Sensitive(), // usually passphrase encrypted, but not in dev and maybe local use
		field.Bool("is_identity_encrypted_with_passphrase"),

		field.String("s3_endpoint"),
		field.String("s3_access_key_id"),
		// TODO String or Bytes?
		field.Bytes("s3_secret_access_key").Sensitive().GoType(entx.EncryptedString("")),
		field.String("s3_bucket_name"),
		field.Bool("s3_use_ssl").Default(true),

		field.Bool("tls_enable_autocert"),
		// can be defined by user because sometimes it is shared and thus located outside of simpledms directory
		// and should also not be within metaPath
		field.String("tls_cert_filepath"),
		field.String("tls_private_key_filepath"),
		field.String("tls_autocert_email"),
		field.Strings("tls_autocert_hosts"),

		field.String("mailer_host").Default(""),
		field.Int("mailer_port").Default(25),
		field.String("mailer_username").Default(""),
		field.Bytes("mailer_password").Optional().Sensitive().GoType(entx.EncryptedString("")), // FIXME encrypt; also not encrypted in config file
		field.String("mailer_from").Default(""),                                                // TODO name okay?
		field.Bool("mailer_insecure_skip_verify").Default(false),
		// true would be recommended as default, but set to false because this was the default
		// before implicit SSL/TLS support was implemented
		field.Bool("mailer_use_implicit_ssl_tls").Default(false),

		field.String("ocr_tika_url").Default(""),
		field.Int64("ocr_max_file_size_mib").Default(25),

		field.Time("initialized_at").Optional().Nillable(),
	}
}

func (SystemConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewCommonMixin(SystemConfig.Type),
	}
}
