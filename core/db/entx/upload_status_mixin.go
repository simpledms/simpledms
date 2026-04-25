package entx

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

type UploadStatusMixin struct {
	mixin.Schema
}

// Unsafe because it should not be used directly in entities, needs a wrapper per schema
func NewUnsafeUploadStatusMixin() UploadStatusMixin {
	return UploadStatusMixin{}
}

func (UploadStatusMixin) Fields() []ent.Field {
	return []ent.Field{
		// Optional() for legacy reason, old files have no upload_started_at value
		field.Time("upload_started_at").Optional().Default(time.Now),
		field.Time("upload_failed_at").Optional().Nillable(),
		field.Time("upload_succeeded_at").Optional().Nillable(),
	}
}

// P adds a storage-level predicate to the queries and mutations.
func (qq UploadStatusMixin) P(w interface{ WhereP(...func(*sql.Selector)) }) {
	w.WhereP(
		sql.OrPredicates(
			// condition is for legacy files (before v1.8.0), that have
			// no upload state information
			sql.FieldIsNull("upload_started_at"),
			// this condition is for newer files (v1.8.0 and later)
			sql.FieldNotNull("upload_succeeded_at"),
		),
	)
}
