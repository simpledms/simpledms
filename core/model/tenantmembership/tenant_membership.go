package tenantmembership

import (
	"github.com/simpledms/simpledms/core/db/entmain"
	"github.com/simpledms/simpledms/ctxx"
)

type TenantMembership struct {
	Data *entmain.TenantAccountAssignment
}

func NewTenantMembership(data *entmain.TenantAccountAssignment) *TenantMembership {
	return &TenantMembership{
		Data: data,
	}
}

func (qq *TenantMembership) IsOwningTenant() bool {
	return qq.Data.IsOwningTenant
}

func (qq *TenantMembership) Remove(ctx ctxx.Context) error {
	return ctx.MainCtx().MainTx.TenantAccountAssignment.DeleteOneID(qq.Data.ID).Exec(ctx)
}
