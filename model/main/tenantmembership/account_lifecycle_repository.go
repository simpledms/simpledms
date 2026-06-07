package tenantmembership

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
)

type AccountLifecycleRepository interface {
	TenantAccountAssignmentByTenantAndAccount(ctx ctxx.Context, tenantID int64, accountID int64) (*entmain.TenantAccountAssignment, error)
	ActiveAccountByID(ctx ctxx.Context, accountID int64) (*entmain.Account, error)
}

type EntAccountLifecycleRepository struct{}

var _ AccountLifecycleRepository = (*EntAccountLifecycleRepository)(nil)

func NewEntAccountLifecycleRepository() *EntAccountLifecycleRepository {
	return &EntAccountLifecycleRepository{}
}

func (qq *EntAccountLifecycleRepository) TenantAccountAssignmentByTenantAndAccount(
	ctx ctxx.Context,
	tenantID int64,
	accountID int64,
) (*entmain.TenantAccountAssignment, error) {
	return ctx.MainCtx().MainTx.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.TenantID(tenantID),
			tenantaccountassignment.AccountID(accountID),
		).
		Only(ctx)
}

func (qq *EntAccountLifecycleRepository) ActiveAccountByID(ctx ctxx.Context, accountID int64) (*entmain.Account, error) {
	return ctx.MainCtx().MainTx.Account.Query().
		Where(
			account.ID(accountID),
			account.DeletedAtIsNil(),
		).
		Only(ctx)
}
