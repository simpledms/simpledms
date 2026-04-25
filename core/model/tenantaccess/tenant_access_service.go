package tenantaccess

import (
	"context"
	"time"

	"github.com/simpledms/simpledms/core/db/entmain"
	"github.com/simpledms/simpledms/core/db/entmain/account"
	"github.com/simpledms/simpledms/core/db/entmain/predicate"
	"github.com/simpledms/simpledms/core/db/entmain/session"
	"github.com/simpledms/simpledms/core/db/entmain/tenant"
	"github.com/simpledms/simpledms/core/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/core/model/common/mainrole"
	"github.com/simpledms/simpledms/core/model/common/tenantrole"
	"github.com/simpledms/simpledms/ctxx"
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

func (qq *TenantAccessService) IsOwningTenantAdminForAccount(
	ctx *ctxx.MainContext,
	actingAccountID int64,
	targetAccountID int64,
) (bool, error) {
	owningTenantID, hasOwningTenantID, err := qq.OwningTenantIDForAccount(
		ctx,
		targetAccountID,
	)
	if err != nil {
		return false, err
	}
	if !hasOwningTenantID {
		return false, nil
	}

	return qq.IsActiveTenantOwner(
		ctx,
		actingAccountID,
		owningTenantID,
	)
}

func (qq *TenantAccessService) OwningTenantIDForAccount(
	ctx *ctxx.MainContext,
	accountID int64,
) (int64, bool, error) {
	now := qq.now()

	owningAssignments, err := ctx.MainTx.TenantAccountAssignment.Query().
		Where(qq.activeTenantAssignmentPredicates(
			now,
			tenantaccountassignment.AccountID(accountID),
			tenantaccountassignment.IsOwningTenant(true),
		)...).
		All(ctx)
	if err != nil {
		return 0, false, err
	}

	if len(owningAssignments) > 0 {
		return owningAssignments[0].TenantID, true, nil
	}

	activeAssignments, err := ctx.MainTx.TenantAccountAssignment.Query().
		Where(qq.activeTenantAssignmentPredicates(
			now,
			tenantaccountassignment.AccountID(accountID),
		)...).
		All(ctx)
	if err != nil {
		return 0, false, err
	}

	if len(activeAssignments) != 1 {
		return 0, false, nil
	}

	legacyOwningAssignment := activeAssignments[0]
	err = ctx.MainTx.TenantAccountAssignment.UpdateOneID(legacyOwningAssignment.ID).
		SetIsOwningTenant(true).
		Exec(ctx)
	if err != nil {
		return 0, false, err
	}

	return legacyOwningAssignment.TenantID, true, nil
}

func (qq *TenantAccessService) TenantOwnsActiveAccounts(
	ctx *ctxx.MainContext,
	tenantID int64,
) (bool, error) {
	now := qq.now()

	hasOwnsActiveAccounts, err := ctx.MainTx.TenantAccountAssignment.Query().
		Where(qq.activeTenantAssignmentPredicates(
			now,
			tenantaccountassignment.TenantID(tenantID),
			tenantaccountassignment.IsOwningTenant(true),
		)...).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	if hasOwnsActiveAccounts {
		return true, nil
	}

	legacyAssignments, err := ctx.MainTx.TenantAccountAssignment.Query().
		Where(qq.activeTenantAssignmentPredicates(
			now,
			tenantaccountassignment.TenantID(tenantID),
			tenantaccountassignment.IsOwningTenant(false),
		)...).
		All(ctx)
	if err != nil {
		return false, err
	}

	for _, legacyAssignment := range legacyAssignments {
		owningTenantID, hasOwningTenantID, err := qq.OwningTenantIDForAccount(
			ctx,
			legacyAssignment.AccountID,
		)
		if err != nil {
			return false, err
		}
		if !hasOwningTenantID {
			continue
		}
		if owningTenantID == tenantID {
			return true, nil
		}
	}

	return false, nil
}

func (qq *TenantAccessService) IsSoleActiveOwnerOfAnyActiveTenant(
	ctx context.Context,
	mainTx *entmain.Tx,
	accountID int64,
) (bool, error) {
	now := qq.now()

	ownedAssignments, err := mainTx.TenantAccountAssignment.Query().
		Where(qq.activeTenantAssignmentPredicates(
			now,
			tenantaccountassignment.AccountID(accountID),
			tenantaccountassignment.RoleEQ(tenantrole.Owner),
		)...).
		All(ctx)
	if err != nil {
		return false, err
	}

	for _, assignment := range ownedAssignments {
		ownersCount, err := mainTx.TenantAccountAssignment.Query().
			Where(qq.activeTenantAssignmentPredicates(
				now,
				tenantaccountassignment.TenantID(assignment.TenantID),
				tenantaccountassignment.RoleEQ(tenantrole.Owner),
			)...).
			Count(ctx)
		if err != nil {
			return false, err
		}

		if ownersCount <= 1 {
			return true, nil
		}
	}

	return false, nil
}

func (qq *TenantAccessService) InvalidateSessionsForTenantUsersWithoutActiveAssignments(
	ctx context.Context,
	mainTx *entmain.Tx,
	tenantID int64,
) error {
	assignments, err := mainTx.TenantAccountAssignment.Query().
		Where(
			tenantaccountassignment.TenantID(tenantID),
			tenantaccountassignment.HasAccountWith(
				account.DeletedAtIsNil(),
				account.RoleEQ(mainrole.User),
			),
		).
		All(ctx)
	if err != nil {
		return err
	}

	accountIDsByID := make(map[int64]struct{}, len(assignments))
	for _, assignment := range assignments {
		accountIDsByID[assignment.AccountID] = struct{}{}
	}

	var accountIDsWithoutActiveTenant []int64
	for accountID := range accountIDsByID {
		hasActiveAssignment, err := qq.HasActiveTenantAssignment(ctx, mainTx, accountID)
		if err != nil {
			return err
		}

		if hasActiveAssignment {
			continue
		}

		accountIDsWithoutActiveTenant = append(accountIDsWithoutActiveTenant, accountID)
	}

	if len(accountIDsWithoutActiveTenant) == 0 {
		return nil
	}

	_, err = mainTx.Session.Delete().
		Where(session.AccountIDIn(accountIDsWithoutActiveTenant...)).
		Exec(ctx)

	return err
}
