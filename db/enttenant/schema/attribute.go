package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/model/common/attributetype"
)

type Attribute struct {
	ent.Schema
}

func (Attribute) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("document_type_id"),

		// not necessary if property attribute
		field.String("name").Default(""),

		field.Bool("is_name_giving").Default(false),

		field.Bool("is_protected").Default(false),
		field.Bool("is_disabled").Default(false), // mainly if protected
		field.Bool("is_required").Default(false),
		// TODO is_multiselect?
		// TODO is used as identifier? primary value

		field.Enum("type").
			GoType(attributetype.AttributeType(0)),

		// TODO enforce that one is not null?
		field.Int64("tag_id").Optional(),
		field.Int64("property_id").Optional(),
	}
}

func (Attribute) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			From("document_type", DocumentType.Type).
			Ref("attributes").
			Unique().
			Field("document_type_id").
			Required(),
		edge.
			To("tag", Tag.Type).
			Field("tag_id").
			// necessary because field is optional and ent uses
			// `ON DELETE SET NULL` as default for optional fields
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Unique(),
		edge.To("property", Property.Type).
			Field("property_id").
			// necessary because field is optional and ent uses
			// `ON DELETE SET NULL` as default for optional fields
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Unique(),
	}
}

func (Attribute) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("space_id", "document_type_id", "name").
			Unique().
			Annotations(entsql.IndexWhere("`name` != ''")),
		index.Fields("space_id", "document_type_id", "tag_id"). // ignores Null values
			// Annotations(entsql.IndexWhere("`tag_id` is not null")). // necessary?
			Unique(),
		index.Fields("space_id", "document_type_id", "property_id"). // ignores Null values
			// Annotations(entsql.IndexWhere("`property_id` is not null")). // necessary?
			Unique(),
	}
}

func (Attribute) Mixin() []ent.Mixin {
	return []ent.Mixin{
		// entcommon.NewPublicIDMixin(true),
		NewSpaceMixin(),
	}
}
