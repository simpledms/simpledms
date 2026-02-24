package modelmain

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entmain/session"
	"github.com/simpledms/simpledms/db/entmain/tenantaccountassignment"
	account2 "github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/model/common/mainrole"
)

type RemoveAccountFromTenantResult struct {
	IsOwningTenantAssignment bool
	AccountSoftDeleted       bool
}

type TenantAccountLifecycleService struct {
	tenantAccessService *TenantAccessService
}

func NewTenantAccountLifecycleService() *TenantAccountLifecycleService {
	return &TenantAccountLifecycleService{
		tenantAccessService: NewTenantAccessService(),
	}
}

func (qq *TenantAccountLifecycleService) RemoveAccountsForDeletedTenant(ctx ctxx.Context, tenantID int64) error {
	assignments, err := ctx.MainCtx().MainTx.TenantAccountAssignment.Query().
		Where(tenantaccountassignment.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return err
	}

	for _, assignment := range assignments {
		_, err = qq.RemoveAccountFromTenant(ctx, tenantID, assignment.AccountID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (qq *TenantAccountLifecycleService) RemoveAccountFromTenant(
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

	if assignment.IsOwningTenant {
		err = qq.softDeleteOwningAccount(ctx, accountID)
		if err != nil {
			return nil, err
		}

		return &RemoveAccountFromTenantResult{
			IsOwningTenantAssignment: true,
			AccountSoftDeleted:       true,
		}, nil
	}

	err = ctx.MainCtx().MainTx.TenantAccountAssignment.DeleteOneID(assignment.ID).Exec(ctx)
	if err != nil {
		return nil, err
	}

	err = qq.invalidateSessionsIfNoActiveTenantAssignment(
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

func (qq *TenantAccountLifecycleService) softDeleteOwningAccount(ctx ctxx.Context, accountID int64) error {
	_, err := ctx.MainCtx().MainTx.TenantAccountAssignment.Delete().
		Where(tenantaccountassignment.AccountID(accountID)).
		Exec(ctx)
	if err != nil {
		return err
	}

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

	return nil
}

func (qq *TenantAccountLifecycleService) invalidateSessionsIfNoActiveTenantAssignment(
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

	hasActiveTenantAssignment, err := qq.tenantAccessService.HasActiveTenantAssignment(
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

	_, err = ctx.MainCtx().MainTx.Session.Delete().
		Where(session.AccountID(accountID)).
		Exec(ctx)

	return err
}
