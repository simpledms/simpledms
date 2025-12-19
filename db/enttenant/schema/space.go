package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/predicate"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/common/tenantrole"
)

type Space struct {
	ent.Schema
}

// TODO is a space just a file?
func (Space) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("name"),
		field.String("icon").Optional(),
		field.String("description").Optional(),
		field.Bool("is_folder_mode").Default(false), // TODO or has_folder_mode_enabled
		// TODO storage_backend? s3, file system, FTP, etc.
		//		FTP probably just live read, otherwise we have to sync...
	}
}

func (Space) Edges() []ent.Edge {
	// TODO inbox?
	// TODO users/accounts and permissions
	// TODO temporary tokens to access the space
	return []ent.Edge{
		/*edge.
		From("files", File.Type).
		Ref("spaces").
		Through("file_assignment", SpaceFileAssignment.Type),*/
		edge.
			From("files", File.Type).
			Ref("space"),
		edge.
			From("users", User.Type).
			Ref("spaces").
			Through("user_assignment", SpaceUserAssignment.Type),
		edge.From("tags", Tag.Type).
			Ref("space"),
		edge.From("document_types", DocumentType.Type).
			Ref("space"),
		edge.From("properties", Property.Type).
			Ref("space"),
	}
}

func (Space) Indexes() []ent.Index {
	return []ent.Index{
		index.
			Fields("name").
			Annotations(entsql.IndexWhere("`deleted_at` is null")).
			Unique(),
	}
}

func (Space) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entx.NewPublicIDMixin(false), // spaces can be renamed
		NewSoftDeleteMixin(Space.Type),
	}
}

func (Space) Policy() ent.Policy {
	type Filter interface {
		WhereHasUserAssignmentWith(...predicate.SpaceUserAssignment)
	}
	mutationPrivacyFn := privacy.FilterFunc(func(untypedCtx context.Context, filterx privacy.Filter) error {
		ctx, ok := ctxx.TenantCtx(untypedCtx)
		if !ok {
			return privacy.Denyf("unexpected context type %T", untypedCtx)
		}

		if ctx.User.Role != tenantrole.Owner {
			return privacy.Denyf("only the tenant owner can create and edit spaces")
		}

		return privacy.Skip
	})

	queryPrivacyFn := privacy.FilterFunc(func(untypedCtx context.Context, filterx privacy.Filter) error {
		ctx, ok := ctxx.TenantCtx(untypedCtx)
		if !ok {
			return privacy.Denyf("unexpected context type %T", untypedCtx)
		}

		// the owner must get a full list of spaces on the Dashboard
		if ctx.User.Role == tenantrole.Owner {
			return privacy.Skip
		}

		filter, ok := filterx.(Filter)
		if !ok {
			return privacy.Denyf("unexpected filter type %T", filterx)
		}
		filter.WhereHasUserAssignmentWith(
			spaceuserassignment.HasUserWith(user.ID(ctx.User.ID)),
		)

		return privacy.Skip
	})

	return privacy.Policy{
		Mutation: privacy.MutationPolicy{
			mutationPrivacyFn,
		},
		Query: privacy.QueryPolicy{
			queryPrivacyFn,
		},
	}
}
