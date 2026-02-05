package schema

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/db/enttenant/hook"
	"github.com/simpledms/simpledms/db/enttenant/intercept"
	"github.com/simpledms/simpledms/db/entx"
)

// IMPORTANT
// shares most code with entmain.SoftDeleteMixin; cannot merge because hook and
// intercept dependencies belong to enttenant/entmain
type SoftDeleteMixin struct {
	entx.SoftDeleteMixin
}

func NewSoftDeleteMixin(entityType any) SoftDeleteMixin {
	return SoftDeleteMixin{
		SoftDeleteMixin: entx.NewUnsafeSoftDeleteMixin(entityType, User.Type),
	}
}

type softDeleteKey struct{}

func SkipSoftDelete(parent context.Context) context.Context {
	return context.WithValue(parent, softDeleteKey{}, true)
}

func (qq SoftDeleteMixin) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{
		intercept.TraverseFunc(func(ctx context.Context, q intercept.Query) error {
			// Skip soft-delete, means include soft-deleted entities.
			if skip, _ := ctx.Value(softDeleteKey{}).(bool); skip {
				return nil
			}
			qq.P(q)
			return nil
		}),
	}
}

func (qq SoftDeleteMixin) Hooks() []ent.Hook {
	return []ent.Hook{
		hook.On(
			func(next ent.Mutator) ent.Mutator {
				return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
					// Skip soft-delete, means delete the entity permanently.
					if skip, _ := ctx.Value(softDeleteKey{}).(bool); skip {
						return next.Mutate(ctx, m)
					}
					mx, ok := m.(interface {
						SetOp(ent.Op)
						// Client() *gen.Client
						SetDeleteTime(time.Time)
						WhereP(...func(*sql.Selector))
					})
					if !ok {
						return nil, fmt.Errorf("unexpected mutation type %T", m)
					}
					qq.P(mx)
					mx.SetOp(ent.OpUpdate)
					mx.SetDeleteTime(time.Now())
					return next.Mutate(ctx, m)
				})
			},
			ent.OpDeleteOne|ent.OpDelete,
		),
	}
}
