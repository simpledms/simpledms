package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// TagAssignment holds the schema definition for the TagAssignment entity.
// TODO rename to FileTagAssignment
type TagAssignment struct {
	ent.Schema
}

// Fields of the TagAssignment.
func (TagAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("file_id"),
		field.Int64("tag_id"),
		// TODO type string? or multi type including bool?
		//  	with Int is easier to work to in the beginning;
		//		bool can be simulated with tags, string probably too?
		//		ints can also be simulated, but can have a to width range,
		//		for example invoice amounts
		// field.Int("attribute_value").Optional(),
		// when a assigned to directory, do all files inherit tag
		// TODO recursive or just one layer? limiting to direct children
		//		makes it easier to implement and also easier to understand
		//		why a tag is assigned
		// TODO not sure if it makes sense; can also be solved when filtering:
		//		`all files from folders tagged with x`
		// field.Bool("is_inherited").Default(false),
		// TODO document property type? or implicit? what if reselected via tags?

		// TODO add property_id? difficult to set if set via tags... try implicit first
	}
}

// Edges of the TagAssignment.
func (TagAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			To("tag", Tag.Type).
			Unique().
			Required().
			Field("tag_id"),
		edge.
			To("file", File.Type).
			Unique().
			Required().
			Field("file_id"),
	}
}

func (TagAssignment) Indexes() []ent.Index {
	return []ent.Index{}
}

func (TagAssignment) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewSpaceMixin(),
	}
}
