package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// cannot name TagAssignmentResolved because view name must be plural and ent uses
// tag_assignment_resolveds
type ResolvedTagAssignment struct {
	ent.View
}

// manually executed in main.go
func (ResolvedTagAssignment) SQL() string {
	// language=sqlite
	return `
		DROP VIEW IF EXISTS resolved_tag_assignments; 
		CREATE VIEW resolved_tag_assignments (tag_id, file_id, space_id) AS 
			-- select all directly assigned tags
			SELECT tag_id, file_id, space_id FROM tag_assignments
			UNION
			-- select all sub-tags of assigned composed tags
			-- super_tag_id is not named clearly, should be sub_tag_id, but is a bug in ent, see
			-- https://github.com/ent/ent/issues/957
			SELECT st.super_tag_id AS tag_id, ta.file_id AS file_id, ta.space_id AS space_id FROM tag_assignments ta
			LEFT JOIN tag_sub_tags st ON ta.tag_id = st.tag_id
			WHERE st.tag_id IS NOT NULL;
	`
}

func (ResolvedTagAssignment) Annotations() []schema.Annotation {
	return []schema.Annotation{}
}

// Fields of the TagAssignmentResolved.
func (ResolvedTagAssignment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("tag_id"), // .StorageKey("tag_id"),
		field.Int64("file_id"),
		field.Int64("space_id"),
	}
}

func (ResolvedTagAssignment) Policy() ent.Policy {
	// cannot use Mixin because edge seems not supported in views
	return NewSpaceMixin().Policy()
}

// TODO how to define edges
func (ResolvedTagAssignment) Edges() []ent.Edge {
	return []ent.Edge{
		// edge.From("tags", Tag.Type).Field("tag_id").Ref("tags"),
		// edge.To("tag", Tag.Type).Required().Unique().Field("tag_id"), // .Unique().Required(), // .Field("tag_id"),
		// edge.To("tag", Tag.Type).StorageKey(edge.Column("tag_id")).Required(),
	}
}

func (ResolvedTagAssignment) Mixin() []ent.Mixin {
	return []ent.Mixin{}
}
