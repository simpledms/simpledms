package schema

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/predicate"
	"github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	previewconversion "github.com/simpledms/simpledms/model/tenant/previewconversion"
)

type PreviewConversion struct {
	ent.Schema
}

func (PreviewConversion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("source_stored_file_id"),
		field.Int64("preview_stored_file_id").Optional().Nillable(),
		field.Enum("status").GoType(previewconversion.Pending),
		field.Int("retry_count").Default(0),
		field.Time("last_attempted_at").Optional().Nillable(),
		field.Time("next_attempt_at").Optional().Nillable(),
		field.Time("processing_started_at").Optional().Nillable(),
		field.String("failure_category").Optional().Nillable(),
	}
}

func (PreviewConversion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("source", StoredFile.Type).
			Unique().Required().Field("source_stored_file_id"),
		edge.To("preview", StoredFile.Type).
			Unique().Field("preview_stored_file_id"),
	}
}

func (PreviewConversion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source_stored_file_id").Unique(),
		index.Fields("status", "next_attempt_at", "id"),
	}
}

func (PreviewConversion) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewCommonMixin(PreviewConversion.Type),
	}
}

func (PreviewConversion) Policy() ent.Policy {
	type sourceFilter interface {
		WhereHasSourceWith(...predicate.StoredFile)
	}

	privacyFn := privacy.FilterFunc(func(untypedCtx context.Context, filterx privacy.Filter) error {
		ctx, ok := ctxx.SpaceCtx(untypedCtx)
		if !ok {
			return privacy.Denyf("unexpected context type %T", untypedCtx)
		}

		filter, ok := filterx.(sourceFilter)
		if !ok {
			return privacy.Denyf("unexpected filter type %T", filterx)
		}

		filter.WhereHasSourceWith(
			storedfile.HasFilesWith(file.SpaceID(ctx.SpaceCtx().Space.ID)),
		)
		return privacy.Skip
	})

	return privacy.Policy{
		Mutation: privacy.MutationPolicy{privacyFn},
		Query:    privacy.QueryPolicy{privacyFn},
	}
}
