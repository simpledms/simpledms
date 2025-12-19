package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"

	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/enttenant/file"
	"github.com/simpledms/simpledms/app/simpledms/enttenant/predicate"
	"github.com/simpledms/simpledms/app/simpledms/enttenant/privacy"
	"github.com/simpledms/simpledms/app/simpledms/enttenant/space"
	"github.com/simpledms/simpledms/app/simpledms/model/common/storagetype"
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
			Ref("versions"),
		// Field("file_id").
		// Unique().
		// Required(),
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
	}
}

func (StoredFile) Policy() ent.Policy {
	type SpacesFilter interface {
		WhereHasFilesWith(...predicate.File)
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
		spacesFilter.WhereHasFilesWith(
			file.HasSpaceWith(space.ID(ctx.SpaceCtx().Space.ID)),
			// entql.Int64EQ(ctx.SpaceCtx().Space.ID),
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
