package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/db/entmain/hook"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
)

// similar to enttenant.StoredFile
type TemporaryFile struct {
	ent.Schema
}

func (TemporaryFile) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),

		field.Int64("owner_id").Immutable(),
		field.String("filename").Immutable(),
		field.Enum("source").
			GoType(filesource.FileSource(0)).
			Immutable().
			Annotations(entsql.Default(filesource.UnknownLegacy.String())),

		field.Int64("size").Optional(), // os.FileInfo.Size is int64
		field.Int64("size_in_storage"), // often gzipped

		field.String("sha256").Optional(),
		field.String("content_sha256").Optional(),
		field.String("storage_crc32c").Optional().Nillable(),
		field.String("mime_type").Optional(), // TODO necessary?

		field.Enum("storage_type").GoType(storagetype.Unknown),
		field.String("bucket_name").Optional(),

		field.String("storage_path"),
		field.String("storage_filename"),

		field.String("upload_token"),

		// using a bool like can_be_deleted is not a good idea because it may lead to early deletion before
		// the file was processed by the scheduler and thus moved to the assigned location

		// if this date is set, normal file processing (moving tmp files to final destination) is responsible
		// for deletion and expires_at can be ignored; ideally the client should set expires_at to NULL if
		// this value is set
		field.Time("converted_to_stored_file_at").Optional().Nillable(),
		field.String("persistence_claim_token").Optional().Nillable(),
		field.Int64("persistence_tenant_id").Optional().Nillable(),
		field.Time("persistence_last_progress_at").Optional().Nillable(),
		field.Time("expires_at").Optional().Nillable(),
	}
}

func (TemporaryFile) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(setDefaultFileSource(filesource.UnknownLegacy), ent.OpCreate),
	}
}

func (TemporaryFile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("owner", Account.Type).
			Field("owner_id").
			Required().
			Immutable().
			Unique(),
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

func (TemporaryFile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("upload_token"),
		index.
			Fields("expires_at", "id").
			StorageKey("temporaryfile_delete_pending").
			Annotations(entsql.IndexWhere(
				"`converted_to_stored_file_at` is null and `deleted_at` is null",
			)),
	}
}

func (TemporaryFile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		// TODO necessary or not?
		entx.NewPublicIDMixin(true),
		NewCommonMixin(TemporaryFile.Type),
		NewSoftDeleteMixin(TemporaryFile.Type),
		NewUploadStatusMixin(),
	}
}
