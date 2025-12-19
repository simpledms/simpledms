package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/entx"
)

// File holds the schema definition for the File entity.
type File struct {
	ent.Schema
}

// Fields of the File.
func (File) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		// TODO move to StoredFile? what if it contains a date or version string?
		//		but shouldn't get changed all the time either;
		//		what if extension changes because new version has another file type?
		field.String("name"), // is filename // TODO rename to filename?
		field.Bool("is_directory"),

		field.String("notes").Optional(), // notes because description should be reserved for AI generated content

		// try to persist original date // TODO necessary or updated_at enough?
		field.Time("modified_at").Optional().Nillable(), // TODO modification_time or modified_at?
		field.Time("indexed_at"),
		field.Time("indexing_completed_at").Optional().Nillable(), // TODO name

		field.Int64("parent_id").Optional(),
		field.Int64("document_type_id").Optional(),

		field.Bool("is_in_inbox").Default(false),
		field.Bool("is_root_dir").Default(false),

		field.Text("ocr_content").Default(""),
		field.Time("ocr_success_at").Optional().Nillable(),
		field.Int("ocr_retry_count").Default(0),
		field.Time("ocr_last_tried_at").Default(time.Time{}), // zero value instead of nil makes logic easier
		// TODO ocr error message?

		// field.String("crc32"), // maybe for quick checks
	}
}

// Edges of the File.
func (File) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			To("versions", StoredFile.Type), // not required because directories have no version...
		edge.
			To("parent", File.Type).
			Field("parent_id").
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Unique().
			From("children"),
		edge.
			To("document_type", DocumentType.Type).
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Field("document_type_id").
			Unique(),
		edge.
			To("tags", Tag.Type).
			Through("tag_assignment", TagAssignment.Type),
		edge.
			To("properties", Property.Type).
			Through("property_assignment", FilePropertyAssignment.Type),
		/*edge.
		To("spaces", Space.Type).
		Through("space_assignment", SpaceFileAssignment.Type).
		Required(),*/
		// edge.
		// To("file_info", FileInfo.Type).Field("id").Unique(),

		// edge.To("tags", Tag.Type),
		// edge.From("document", Document.Type).Ref("file").Unique(),
		// edge.From("image", Image.Type).Ref("file").Unique(),
		// edge.From("video", Video.Type).Ref("file").Unique(),
	}
}

func (File) Indexes() []ent.Index {
	// TODO seems not to work when parent_id is NULL
	return []ent.Index{
		// TODO makes no sense for non-folder mode, only in folder mode but there we can check manually...
		//		constraint not possible because is_folder_mode is defined in another table;
		//
		// 		can be solved by auto-renaming in non-folder mode, for example suffixing public ID
		index.
			Fields("space_id", "name", "parent_id").
			Annotations(entsql.IndexWhere("`deleted_at` is null and `is_in_inbox` = false")).
			Unique(),
		index.
			Fields("space_id", "is_root_dir").
			Annotations(entsql.IndexWhere("`is_root_dir` = true")).
			Unique(),
	}
}

func (File) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewSoftDeleteMixin(File.Type), // TODO necessary?
		entx.NewPublicIDMixin(true),
		NewCommonMixin(File.Type),
		NewSpaceMixin(),
	}
}
