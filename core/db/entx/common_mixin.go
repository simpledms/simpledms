package entx

import (
	"reflect"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

type CommonMixin struct {
	mixin.Schema
	entityType any
	userType   any
}

// Unsafe because it should not be used directly in entities, needs a wrapper per schema
func NewUnsafeCommonMixin(entityType, userType any) CommonMixin {
	return CommonMixin{
		entityType: entityType,
		userType:   userType,
	}
}

func (CommonMixin) Fields() []ent.Field {
	return []ent.Field{
		field.
			Time("created_at").
			Immutable().
			Default(time.Now),
		field.
			Int64("created_by").
			Immutable().
			Optional(),
		field.
			Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
		field.
			Int64("updated_by").
			Optional(),
	}
}

func (qq CommonMixin) Edges() []ent.Edge {
	if reflect.TypeOf(qq.entityType).In(0).Name() == reflect.TypeOf(qq.userType).In(0).Name() {
		// edge defined below creates a UNIQUE index when referencing itself; found no way to fix this,
		// thus this workaround for Account and User table
		return []ent.Edge{}
	}
	return []ent.Edge{
		// TODO are the names good?
		edge.To("creator", qq.userType).
			Field("created_by").
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Unique().
			Immutable(),
		edge.To("updater", qq.userType).
			Field("updated_by").
			Annotations(entsql.OnDelete(entsql.NoAction)).
			Unique(),
	}
}
