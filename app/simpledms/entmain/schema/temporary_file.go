package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/app/simpledms/entx"
	"github.com/simpledms/simpledms/app/simpledms/model/common/storagetype"
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

		field.Int64("size").Optional(), // os.FileInfo.Size is int64
		field.Int64("size_in_storage"), // often gzipped

		field.String("sha256").Optional(),
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
		field.Time("expires_at").Optional().Nillable(),
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

func (TemporaryFile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("upload_token"),
	}
}

func (TemporaryFile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		// TODO necessary or not?
		entx.NewPublicIDMixin(true),
		NewCommonMixin(TemporaryFile.Type),
		NewSoftDeleteMixin(TemporaryFile.Type),
	}
}
