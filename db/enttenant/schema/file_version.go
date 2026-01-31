package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FileVersion holds the schema definition for the FileVersion entity.
type FileVersion struct {
	ent.Schema
}

// Fields of the FileVersion.
func (FileVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("file_id"),
		field.Int64("stored_file_id"),
		// TODO remove Default, only for auto migration
		field.Int("version_number").Default(1),
		// field.Int64("merged_from_file_id").Optional(),
		// field.Time("merged_at").Optional().Nillable(),
		// field.Int64("merged_by").Optional(),
		field.Text("note").Optional(),
	}
}

// Edges of the FileVersion.
func (FileVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("file", File.Type).Unique().Required().Field("file_id"),
		edge.To("stored_file", StoredFile.Type).Unique().Required().Field("stored_file_id"),
	}
}

func (FileVersion) Indexes() []ent.Index {
	return []ent.Index{
		// TODO necessary or implicit?
		index.Fields("file_id", "stored_file_id").Unique(),
		index.Fields("file_id", "version_number").Unique(),
	}
}
