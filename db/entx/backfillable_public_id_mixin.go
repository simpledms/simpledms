package entx

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"

	"github.com/simpledms/simpledms/util"
)

// BackfillablePublicIDMixin keeps the column nullable for additive SQLite upgrades. Consumers
// must gate external use on backfill completion and reject empty IDs at projection boundaries.
// Replace it with the required PublicIDMixin in a future schema-tightening migration once that
// transition can be made without rebuilding existing tables.
type BackfillablePublicIDMixin struct {
	mixin.Schema
}

func NewBackfillablePublicIDMixin() *BackfillablePublicIDMixin {
	return &BackfillablePublicIDMixin{}
}

func (BackfillablePublicIDMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("public_id").
			Optional().
			Unique().
			Immutable().
			GoType(CIText("")).
			DefaultFunc(func() CIText {
				return NewCIText(util.NewPublicID())
			}),
	}
}
