package tenantmembership

import (
	account2 "github.com/simpledms/simpledms/core/model/account"
	"github.com/simpledms/simpledms/core/model/common/mainrole"
	"github.com/simpledms/simpledms/core/model/tenantaccess"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
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
	repository := NewEntAccountLifecycleRepository()

	assignment, err := repository.TenantAccountAssignmentByTenantAndAccount(ctx, tenantID, accountID)
	if err != nil {
		if entmain.IsNotFound(err) {
			return &RemoveAccountFromTenantResult{}, nil
		}

		return nil, err
	}
	membership := NewTenantMembership(assignment)

	if membership.IsOwningTenant() {
		err = softDeleteOwningAccount(ctx, repository, accountID)
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
		repository,
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

func softDeleteOwningAccount(
	ctx ctxx.Context,
	repository AccountLifecycleRepository,
	accountID int64,
) error {
	accountx, err := repository.ActiveAccountByID(ctx, accountID)
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
	repository AccountLifecycleRepository,
	accountID int64,
) error {
	accountx, err := repository.ActiveAccountByID(ctx, accountID)
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
