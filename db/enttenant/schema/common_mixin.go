package schema

import (
	"github.com/marcobeierer/go-core/db/entx"
)

type CommonMixin struct {
	entx.CommonMixin
}

func NewCommonMixin(entityType any) CommonMixin {
	return CommonMixin{
		CommonMixin: entx.NewUnsafeCommonMixin(entityType, User.Type),
	}
}
