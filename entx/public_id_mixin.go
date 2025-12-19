package entx

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"

	"github.com/simpledms/simpledms/util"
)

type PublicIDMixin struct {
	mixin.Schema
	isImmutable bool
}

func NewPublicIDMixin(isImmutable bool) *PublicIDMixin {
	return &PublicIDMixin{
		isImmutable: isImmutable,
	}
}

func (qq PublicIDMixin) Fields() []ent.Field {
	publicIDField := field.String("public_id").
		Unique().           // is okay in this situation, even if soft delete is enabled
		GoType(CIText("")). // just to be safe...
		DefaultFunc(func() CIText {
			return NewCIText(util.NewPublicID())
		})

	if qq.isImmutable {
		// for example tenant/org id is immutable
		publicIDField = publicIDField.Immutable()
	}

	return []ent.Field{
		publicIDField,
	}
}

/*
func (qq PublicIDMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("public_id"),
	}
}

*/
