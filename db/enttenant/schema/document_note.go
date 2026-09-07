package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/db/entx"
)

// DocumentNote retains current notes and their deleted or replaced history.
type DocumentNote struct {
	ent.Schema
}

func (DocumentNote) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("file_id"),
		// Empty titles preserve existing and legacy notes without inventing content.
		field.String("title").Optional(),
		field.Text("body"),
		field.Int64("author_id").Optional(),
		field.Time("authored_at").Optional().Nillable(),
		field.Time("edited_at").Optional().Nillable(),
		field.Int64("editor_id").Optional(),
		field.Time("deleted_at").Optional().Nillable(),
		field.Int64("replaced_by_id").Optional(),
	}
}

func (DocumentNote) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("file", File.Type).Unique().Required().Field("file_id"),
		edge.To("author", User.Type).Unique().Field("author_id"),
		edge.To("editor", User.Type).Unique().Field("editor_id"),
		edge.To("predecessor", DocumentNote.Type).Unique().
			From("replacement").Unique().Field("replaced_by_id"),
	}
}

func (DocumentNote) Mixin() []ent.Mixin {
	// Explicit attribution and history fields avoid common author and soft-delete hooks.
	return []ent.Mixin{entx.NewPublicIDMixin(true), NewSpaceMixin()}
}

func (DocumentNote) Indexes() []ent.Index {
	return []ent.Index{index.Fields("file_id")}
}
