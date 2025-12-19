package schema

import (
	"github.com/simpledms/simpledms/app/simpledms/entx"
)

type CommonMixin struct {
	entx.CommonMixin
}

func NewCommonMixin(entityType any) CommonMixin {
	return CommonMixin{
		CommonMixin: entx.NewUnsafeCommonMixin(entityType, Account.Type),
	}
}
