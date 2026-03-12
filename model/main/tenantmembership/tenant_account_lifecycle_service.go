package tenantmembership

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	account2 "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/model/main/tenantaccess"
)

type RemoveAccountFromTenantResult struct {
	IsOwningTenantAssignment bool
	AccountSoftDeleted       bool
}

func RemoveAccountFromTenant(
	ctx ctxx.Context,
	tenantID int64,
	accountID int64,
) (*RemoveAccountFromTenantResult, error) {
	assignment, err := ctx.MainCtx().MainTx.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.TenantID(tenantID),
			tenantaccountassignment.AccountID(accountID),
		).
		Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return &RemoveAccountFromTenantResult{}, nil
		}

		return nil, err
	}
	membership := NewTenantMembership(assignment)

	if membership.IsOwningTenant() {
		err = softDeleteOwningAccount(ctx, accountID)
		if err != nil {
			return nil, err
		}

		return &RemoveAccountFromTenantResult{
			IsOwningTenantAssignment: true,
			AccountSoftDeleted:       true,
		}, nil
	}

	err = membership.Remove(ctx)
	if err != nil {
		return nil, err
	}

	err = invalidateSessionsIfNoActiveTenantAssignment(
		ctx,
		accountID,
	)
	if err != nil {
		return nil, err
	}

	return &RemoveAccountFromTenantResult{
		IsOwningTenantAssignment: false,
		AccountSoftDeleted:       false,
	}, nil
}

func softDeleteOwningAccount(ctx ctxx.Context, accountID int64) error {
	accountx, err := ctx.MainCtx().MainTx.Account.Query().
		Where(
			account.ID(accountID),
			account.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return nil
		}

		return err
	}

	accountm := account2.NewAccount(accountx)
	accountm.UnsafeDelete(ctx)

	err = accountm.RemoveAllTenantAssignments(ctx)
	if err != nil {
		return err
	}

	return nil
}

func invalidateSessionsIfNoActiveTenantAssignment(
	ctx ctxx.Context,
	accountID int64,
) error {
	accountx, err := ctx.MainCtx().MainTx.Account.Query().
		Where(
			account.ID(accountID),
			account.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return nil
		}

		return err
	}
	if accountx.Role != mainrole.User {
		return nil
	}

	hasActiveTenantAssignment, err := tenantaccess.NewTenantAccessService().HasActiveTenantAssignment(
		ctx,
		ctx.MainCtx().MainTx,
		accountID,
	)
	if err != nil {
		return err
	}
	if hasActiveTenantAssignment {
		return nil
	}

	accountm := account2.NewAccount(accountx)
	err = accountm.InvalidateSessions(ctx)

	return err
}
