package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/predicate"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/model/main/common/storagetype"
)

// similar to entmain.TemporaryFile
type StoredFile struct {
	ent.Schema
}

func (StoredFile) Fields() []ent.Field {
	// TODO add UNIQUE indexes?
	return []ent.Field{
		field.Int64("id"),

		// M2M, not O2M
		// field.Int64("file_id"),
		field.String("filename").Immutable(),

		// TODO everything immutable?
		field.Int64("size").Optional(), // os.FileInfo.Size is int64
		field.Int64("size_in_storage"), // often gzipped

		field.String("sha256").Optional(),
		field.String("content_sha256").Optional(),
		field.String("mime_type").Optional(), // was media_type

		field.Enum("storage_type").GoType(storagetype.Unknown),
		field.String("bucket_name").Optional(),

		// always set on creation, so that scheduler can be stupid and just copy files around
		field.String("storage_path"),
		field.String("storage_filename"),

		// all new files are uploaded to a temporary location and then moved to the final
		// destination by a scheduler. this is done to make cleanup easier if a transaction fails
		field.String("temporary_storage_path"),
		field.String("temporary_storage_filename"),

		field.Time("copied_to_final_destination_at").Optional().Nillable(),
		field.Time("deleted_temporary_file_at").Optional().Nillable(),
	}
}

func (StoredFile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.
			// TODO required or not? orphans might have no files linked and are ready for cleanup?
			From("files", File.Type).
			Ref("versions").
			Through("file_versions", FileVersion.Type),
		// Field("file_id").
		// Unique().
		// Required(),
	}
}

func (StoredFile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("content_sha256"),
		index.
			Fields("id").
			StorageKey("storedfile_content_hash_pending").
			Annotations(entsql.IndexWhere(
				"`content_sha256` is null and " +
					"`upload_succeeded_at` is not null and " +
					"`copied_to_final_destination_at` is not null",
			)),
		index.
			Fields("copied_to_final_destination_at", "id").
			StorageKey("storedfile_copy_pending").
			Annotations(entsql.IndexWhere(
				"`copied_to_final_destination_at` is null and `deleted_temporary_file_at` is null",
			)),
		index.
			Fields("copied_to_final_destination_at", "id").
			StorageKey("storedfile_temp_delete_pending").
			Annotations(entsql.IndexWhere(
				"`copied_to_final_destination_at` is not null and `deleted_temporary_file_at` is null",
			)),
	}
}

func (StoredFile) Mixin() []ent.Mixin {
	// no space mixin because stored files can be shared between spaces;
	// File is responsible for space selection
	return []ent.Mixin{
		// PublicID removed on 30.06.2025 because it was only used for naming the file in storage
		// and enforced double bookkeeping on user (keep publicID and filename in sync)
		// entx.NewPublicIDMixin(true),
		NewCommonMixin(StoredFile.Type),
		NewUploadStatusMixin(),
	}
}

func (StoredFile) Policy() ent.Policy {
	type SpacesFilter interface {
		WhereHasFileVersionsWith(...predicate.FileVersion)
	}
	privacyFn := privacy.FilterFunc(func(untypedCtx context.Context, filterx privacy.Filter) error {
		ctx, ok := ctxx.SpaceCtx(untypedCtx)
		if !ok {
			return privacy.Denyf("unexpected context type %T", untypedCtx)
		}

		spacesFilter, ok := filterx.(SpacesFilter)
		if !ok {
			return privacy.Denyf("unexpected filter type %T", filterx)
		}
		// changed on 12.02.2026 to fix performance issue;
		// query is more efficient than
		// spacesFilter.WhereHasFilesWith(file.SpaceID(ctx.SpaceCtx().Space.ID))
		// because it doesn't JOIN files table and thus prevents a
		// full SCAN on files table
		spacesFilter.WhereHasFileVersionsWith(
			fileversion.HasFileWith(file.SpaceID(ctx.SpaceCtx().Space.ID)),
		)

		return privacy.Skip
	})

	return privacy.Policy{
		Mutation: privacy.MutationPolicy{
			privacyFn,
		},
		Query: privacy.QueryPolicy{
			privacyFn,
		},
	}
}
