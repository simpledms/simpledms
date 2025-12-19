package entx

import (
	"reflect"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// mostly copied from docs
// https://entgo.io/docs/interceptors#soft-delete
type SoftDeleteMixin struct {
	mixin.Schema
	entityType any
	userType   any
}

// Unsafe because it should not be used directly in entities, needs a wrapper per schema
func NewUnsafeSoftDeleteMixin(entityType, userType any) SoftDeleteMixin {
	return SoftDeleteMixin{
		entityType: entityType,
		userType:   userType,
	}
}

func (SoftDeleteMixin) Fields() []ent.Field {
	return []ent.Field{
		field.
			Int64("deleted_by").
			Optional(),
		field.
			Time("deleted_at").
			Optional(),
	}
}

func (qq SoftDeleteMixin) Edges() []ent.Edge {
	if reflect.TypeOf(qq.entityType).In(0).Name() == reflect.TypeOf(qq.userType).In(0).Name() {
		// edge defined below creates a UNIQUE index when referencing itself; found no way to fix this,
		// thus this workaround for Account and User table
		return []ent.Edge{}
	}
	return []ent.Edge{
		// TODO name?
		edge.To("deleter", qq.userType).
			Field("deleted_by").
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Unique(),
	}
}

// P adds a storage-level predicate to the queries and mutations.
func (qq SoftDeleteMixin) P(w interface{ WhereP(...func(*sql.Selector)) }) {
	w.WhereP(
		sql.FieldIsNull(qq.Fields()[0].Descriptor().Name),
	)
}
