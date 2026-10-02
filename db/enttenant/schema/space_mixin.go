package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/entql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
)

type SpaceMixin struct {
	mixin.Schema
	isMutable bool
}

func NewSpaceMixin() SpaceMixin {
	return SpaceMixin{}
}

// NewMutableSpaceMixin permits aggregate transfers while retaining Space privacy filters.
func NewMutableSpaceMixin() SpaceMixin {
	return SpaceMixin{
		isMutable: true,
	}
}

func (qq SpaceMixin) Fields() []ent.Field {
	spaceID := field.Int64("space_id")
	if !qq.isMutable {
		spaceID.Immutable()
	}
	return []ent.Field{spaceID}
}

func (qq SpaceMixin) Edges() []ent.Edge {
	spaceEdge := edge.To("space", Space.Type).
		Field("space_id").
		Unique().
		Required()
	if !qq.isMutable {
		spaceEdge.Immutable()
	}
	return []ent.Edge{spaceEdge}
}

func (SpaceMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("space_id"),
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
