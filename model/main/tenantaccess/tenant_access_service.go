package tenantaccess

import (
	"context"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/predicate"
	"github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
)

type TenantAccessService struct {
	now func() time.Time
}

func NewTenantAccessService() *TenantAccessService {
	return &TenantAccessService{
		now: time.Now,
	}
}

func (qq *TenantAccessService) activeTenantAssignmentPredicates(
	now time.Time,
	predicates ...predicate.TenantAccountAssignment,
) []predicate.TenantAccountAssignment {
	activePredicates := []predicate.TenantAccountAssignment{
		tenantaccountassignment.Or(
			tenantaccountassignment.ExpiresAtIsNil(),
			tenantaccountassignment.ExpiresAtGT(now),
		),
		tenantaccountassignment.HasAccountWith(account.DeletedAtIsNil()),
		tenantaccountassignment.HasTenantWith(tenant.DeletedAtIsNil()),
	}

	return append(predicates, activePredicates...)
}

func (qq *TenantAccessService) HasActiveTenantAssignment(
	ctx context.Context,
	mainTx *entmain.Tx,
	accountID int64,
) (bool, error) {
	now := qq.now()

	return mainTx.TenantAccountAssignment.Query().
		Where(qq.activeTenantAssignmentPredicates(
			now,
			tenantaccountassignment.AccountID(accountID),
		)...).
		Exist(ctx)
}

func (qq *TenantAccessService) HasActiveTenantAssignmentForTenant(
	ctx context.Context,
	mainTx *entmain.Tx,
	accountID int64,
	tenantID int64,
) (bool, error) {
	now := qq.now()

	return mainTx.TenantAccountAssignment.Query().
		Where(qq.activeTenantAssignmentPredicates(
			now,
			tenantaccountassignment.AccountID(accountID),
			tenantaccountassignment.TenantID(tenantID),
		)...).
		Exist(ctx)
}

func (qq *TenantAccessService) IsActiveTenantOwner(
	ctx *ctxx.MainContext,
	accountID int64,
	tenantID int64,
) (bool, error) {
	now := qq.now()

	return ctx.MainCtx().MainTx.TenantAccountAssignment.Query().
		Where(qq.activeTenantAssignmentPredicates(
			now,
			tenantaccountassignment.AccountID(accountID),
			tenantaccountassignment.TenantID(tenantID),
			tenantaccountassignment.RoleEQ(tenantrole.Owner),
		)...).
		Exist(ctx)
}
