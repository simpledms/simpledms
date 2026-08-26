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
	webdavresourcemodel "github.com/simpledms/simpledms/model/tenant/webdavresource"
)

type WebDAVResource struct {
	ent.Schema
}

func (WebDAVResource) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("credential_public_id").GoType(entx.CIText("")).Immutable(),
		field.Int64("file_id").Optional().Nillable(),
		field.Int64("stored_file_id").Optional().Nillable(),
		field.String("dav_path"),
		field.Enum("state").
			GoType(webdavresourcemodel.State(0)).
			Annotations(entsql.Default(webdavresourcemodel.Uploading.String())),
		field.Time("last_progress_at").Default(time.Now),
		field.Time("finalized_at").Optional().Nillable(),
	}
}

func (WebDAVResource) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(setDefaultWebDAVResourceState(webdavresourcemodel.Uploading), ent.OpCreate),
	}
}

func (WebDAVResource) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("file", File.Type).
			Field("file_id").
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Unique(),
		edge.To("stored_file", StoredFile.Type).
			Field("stored_file_id").
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Unique(),
	}
}

func setDefaultWebDAVResourceState(state webdavresourcemodel.State) ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (ent.Value, error) {
			stateMutation, ok := mutation.(interface {
				State() (webdavresourcemodel.State, bool)
				SetState(webdavresourcemodel.State)
			})
			if ok {
				if _, exists := stateMutation.State(); !exists {
					stateMutation.SetState(state)
				}
			}

			return next.Mutate(ctx, mutation)
		})
	}
}

func (WebDAVResource) Indexes() []ent.Index {
	return []ent.Index{
		index.
			Fields("credential_public_id", "space_id", "dav_path").
			StorageKey("webdavresource_active_path").
			Annotations(entsql.IndexWhere("`state` in ('Uploading', 'Active')")).
			Unique(),
		index.Fields("state", "last_progress_at", "id"),
		index.Fields("file_id"),
		index.Fields("stored_file_id"),
	}
}

func (WebDAVResource) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewCommonMixin(WebDAVResource.Type),
		NewSpaceMixin(),
	}
}
