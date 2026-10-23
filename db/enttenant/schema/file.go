package schema

import (
	"context"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/db/enttenant/hook"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/filesource"
)

// FileOCRPendingCondition is the condition of the file_ocr_pending_with_markdown_content partial
// index. Queries must contain it verbatim: SQLite only uses a partial index if it can prove that
// the query implies the index condition, which fails for bound parameters like
// `ocr_retry_count` < ?. Content derived from Xberg markdown is still pending for Tika.
const FileOCRPendingCondition = "(`ocr_success_at` is null or " +
	"`is_ocr_content_from_markdown` = true) and `ocr_retry_count` < 3 and `is_directory` = false"

// FileOCRMarkdownPendingCondition is the condition of the file_ocr_markdown_pending partial
// index, see FileOCRPendingCondition. The retry count is NULL for files created before the
// column was added.
const FileOCRMarkdownPendingCondition = "`ocr_markdown_success_at` is null and " +
	"coalesce(`ocr_markdown_retry_count`, 0) < 3 and `is_directory` = false"

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
		// TODO get rid of name? and always use filename from StoredFile?
		field.String("name"), // is filename // TODO rename to filename?
		field.Enum("source").
			GoType(filesource.FileSource(0)).
			Immutable().
			Annotations(entsql.Default(filesource.UnknownLegacy.String())),
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

		// Xberg markdown with tables for LLMs, tracked separately from the Tika OCR above. Optional
		// without defaults so that SQLite can add the columns without rebuilding the table; NULL
		// reads as empty string and 0.
		field.Text("ocr_markdown").Optional(),
		field.Time("ocr_markdown_success_at").Optional().Nillable(),
		field.Int("ocr_markdown_retry_count").Optional(),
		field.Time("ocr_markdown_last_tried_at").Optional().Nillable(),
		// true if ocr_content was derived from ocr_markdown because Tika was not configured; Tika
		// replaces it once configured
		field.Bool("is_ocr_content_from_markdown").Optional(),

		// field.String("crc32"), // maybe for quick checks
	}
}

func (File) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(setDefaultFileSource(filesource.UnknownLegacy), ent.OpCreate),
	}
}

// Edges of the File.
func (File) Edges() []ent.Edge {
	return []ent.Edge{
		// not required because directories have no version...
		edge.
			To("versions", StoredFile.Type).
			Through("file_versions", FileVersion.Type),
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

func setDefaultFileSource(source filesource.FileSource) ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			sourceMutation, ok := mutation.(interface {
				Source() (filesource.FileSource, bool)
				SetSource(filesource.FileSource)
			})
			if ok {
				if _, exists := sourceMutation.Source(); !exists {
					sourceMutation.SetSource(source)
				}
			}

			return next.Mutate(ctx, mutation)
		})
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
		index.Fields("parent_id"),
		index.Fields("is_directory"),
		index.Fields("is_in_inbox"),
		index.Fields("space_id", "parent_id"),
		// Full indexes cover both bound and literal boolean filters used by listing queries.
		index.
			Fields("space_id", "parent_id", "is_in_inbox", "deleted_at", "is_directory", "name").
			StorageKey("file_browse_name_fast").
			Annotations(
				entsql.DescColumns("is_directory"),
			),
		index.
			Fields("space_id", "parent_id", "is_in_inbox", "deleted_at", "is_directory", "created_at", "name").
			StorageKey("file_browse_created_fast").
			Annotations(
				entsql.DescColumns("is_directory", "created_at"),
			),
		index.
			Fields("space_id", "parent_id", "is_in_inbox", "deleted_at", "is_directory", "created_at", "name").
			StorageKey("file_browse_created_oldest_fast").
			Annotations(
				entsql.DescColumns("is_directory"),
			),
		index.
			Fields("space_id", "is_in_inbox", "is_directory", "deleted_at", "created_at").
			StorageKey("file_inbox_created_fast").
			Annotations(
				entsql.DescColumns("created_at"),
			),
		index.
			Fields("space_id", "is_in_inbox", "is_directory", "deleted_at", "name").
			StorageKey("file_inbox_name_fast"),
		index.
			Fields("space_id", "is_in_inbox", "deleted_at", "is_directory", "source", "created_at").
			StorageKey("file_inbox_source_created_fast").
			Annotations(
				entsql.DescColumns("created_at"),
			),
		index.
			Fields("space_id", "is_in_inbox", "deleted_at", "is_directory", "source", "created_at").
			StorageKey("file_inbox_source_created_oldest_fast"),
		index.
			Fields("space_id", "is_in_inbox", "deleted_at", "is_directory", "source", "name").
			StorageKey("file_inbox_source_name_fast"),
		index.
			Fields("ocr_last_tried_at", "id").
			// renamed when the condition changed; Atlas rebuilds the table to modify an index in
			// place on SQLite, but only drops and creates an index with a new name
			StorageKey("file_ocr_pending_with_markdown_content").
			Annotations(entsql.IndexWhere(FileOCRPendingCondition)),
		index.
			Fields("ocr_markdown_last_tried_at", "id").
			StorageKey("file_ocr_markdown_pending").
			Annotations(entsql.IndexWhere(FileOCRMarkdownPendingCondition)),
	}
}

func (File) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewSoftDeleteMixin(File.Type), // TODO necessary?
		entx.NewPublicIDMixin(true),
		NewCommonMixin(File.Type),
		NewMutableSpaceMixin(),
	}
}
