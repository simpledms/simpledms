package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/model/tagging/tagtype"
)

type Tag struct {
	ent.Schema
}

// detailed tagging system description can be found in videoserver sql file
//
// many2many is necessary implicit tags, for example if we want to describe persons
// with tags, the tag for each hair color might be added to multiple persons...
func (Tag) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),

		// TODO not sure if unique works nicely with groups?
		//		may make things simpler, but could also limit options, for example
		//		if the same tag name should exist with different colors or different weights...
		field.String("name"), // .Unique(),
		field.String("color").Optional(),
		field.String("icon").Optional(), // TODO or emoji

		// TODO set default, seems not possible?
		field.
			Enum("type").
			GoType(tagtype.Simple),

		// field.Enum("attribute_type").GoType(attributetype.Unknown),
		// field.String("attribute_unit").Optional(), // for example % or CHF or €

		// for sorting and filtering // TODO or level?
		// for example rating (1 star, 2 star, 3 star) within group
		// field.Int("weight").Optional(),
		field.Int64("group_id").Optional(),

		// TODO should probably be an enum
		// field.Bool("assignable_to_file"),
		// field.Bool("assignable_to_directory"),

		// TODO add searchable (only for group tags)
	}
}

func (Tag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			To("group", Tag.Type).
			Field("group_id").
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Unique().
			From("children"),

		// in the join table tag_sub_tags the second column is named
		// super_tag_id, but should be sub_tag_id; that is a bug in ent
		// https://github.com/ent/ent/issues/957
		//
		// TODO use Through(SubTag), but couldn't get it working,
		// 		problems seems that it is always bidirectional and thus for every
		//		subtag two rows (both directions) get added to table
		// 		!! main disadvantage of current solution is that we have no index...
		edge.
			To("sub_tags", Tag.Type).
			From("super_tags"),
		// Through("tag_compositions", TagComposition.Type),

		edge.
			From("files", File.Type).
			Ref("tags").
			Through("tag_assignment", TagAssignment.Type),
	}
}

func (Tag) Indexes() []ent.Index {
	return []ent.Index{
		index.
			Fields("space_id", "name").
			Annotations(entsql.IndexWhere("`group_id` is null")).
			Unique(),
		index.
			Fields("space_id", "name", "group_id").
			// Annotations(entsql.IndexWhere("`deleted_at` is null")).
			Unique(),
	}
	/*return []ent.Index{
		index.
			Fields("name").
			Edges("parents").
			Unique(),
	}*/
}

func (Tag) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewSpaceMixin(),
		// TODO necessary or not?
		// entcommon.NewPublicIDMixin(true),
	}
}
