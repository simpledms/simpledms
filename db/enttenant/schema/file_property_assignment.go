package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"

	"github.com/simpledms/simpledms/util/timex"
)

type FilePropertyAssignment struct {
	ent.Schema
}

func (FilePropertyAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("file_id"),
		field.Int64("property_id"),

		//	with multiple values instead of one string, we can do calculations (number) or
		//	filtering (for example date) directly in DB
		field.String("text_value").Optional(),
		field.Int("number_value").Optional(),
		field.Time("date_value").GoType(timex.Date{}).Optional(),
		field.Bool("bool_value").Optional(),
	}
}

func (FilePropertyAssignment) Annotations() []schema.Annotation {
	// Keep legacy unset values valid, but never store conflicting typed values.
	singleValue := "((`text_value` is not null) + (`number_value` is not null) + " +
		"(`date_value` is not null) + (`bool_value` is not null)) <= 1"
	return []schema.Annotation{
		entsql.Checks(map[string]string{
			"file_property_assignment_single_value": singleValue,
		}),
	}
}

func (FilePropertyAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("file", File.Type).Unique().Required().Field("file_id"),
		edge.To("property", Property.Type).Unique().Required().Field("property_id"),
	}
}

func (FilePropertyAssignment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewSpaceMixin(),
	}
}
