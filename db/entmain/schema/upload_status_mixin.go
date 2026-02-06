package schema

import (
	"context"

	"entgo.io/ent"

	"github.com/simpledms/simpledms/db/entmain/intercept"
	"github.com/simpledms/simpledms/db/entx"
)

// IMPORTANT
// shares most code with entmain.UploadStatusMixin; cannot merge because hook and
// intercept dependencies belong to enttenant/entmain
type UploadStatusMixin struct {
	entx.UploadStatusMixin
}

func NewUploadStatusMixin() UploadStatusMixin {
	return UploadStatusMixin{
		UploadStatusMixin: entx.NewUnsafeUploadStatusMixin(),
	}
}

type withIncompleteStoredFilesKey struct{}

func WithUnfinishedUploads(parent context.Context) context.Context {
	return context.WithValue(parent, withIncompleteStoredFilesKey{}, true)
}

func (qq UploadStatusMixin) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{
		intercept.TraverseFunc(func(ctx context.Context, q intercept.Query) error {
			// Skip soft-delete, means include soft-deleted entities.
			if skip, _ := ctx.Value(withIncompleteStoredFilesKey{}).(bool); skip {
				return nil
			}
			qq.P(q)
			return nil
		}),
	}
}
