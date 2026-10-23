package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
)

// TODO rename to Account? User seems better for the moment...
type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),

		field.Int64("account_id").
			Unique().
			Immutable().
			Comment("Account ID in main database."), // in main db
		// derived from mainDB.Account
		field.Enum("role").
			GoType(tenantrole.User),

		field.String("email").
			GoType(entx.CIText("")),
		field.String("first_name"),
		field.String("last_name").Optional(),

		field.String("avatar").Optional(),
		field.String("description").Optional(),

		// TODO deleted?
		// field.Bool("is_disabled").Default(false),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("spaces", Space.Type).
			Through("space_assignment", SpaceUserAssignment.Type), // TODO unique?
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("email").
			Annotations(entsql.IndexWhere("`deleted_at` is null")).
			Unique(),
	}
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		NewCommonMixin(User.Type),
		NewSoftDeleteMixin(User.Type),
		entx.NewPublicIDMixin(false),
	}
}
