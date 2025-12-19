package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/entql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
)

type SpaceMixin struct {
	mixin.Schema
}

func NewSpaceMixin() SpaceMixin {
	return SpaceMixin{}
}

func (SpaceMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("space_id").Immutable(),
	}
}

func (SpaceMixin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("space", Space.Type).
			Field("space_id").
			Immutable().
			Unique().
			Required(),
	}
}

func (SpaceMixin) Policy() ent.Policy {
	type SpacesFilter interface {
		WhereSpaceID(entql.Int64P)
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
		spacesFilter.WhereSpaceID(
			entql.Int64EQ(ctx.SpaceCtx().Space.ID),
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
