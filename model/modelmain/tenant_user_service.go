package modelmain

import (
	"log"
	"net/http"
	"time"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/account"
	enttenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	accountmodel "github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/model/common/mainrole"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/model/mailer"
	"github.com/simpledms/simpledms/util/e"
)

type TenantUserService struct{}

func NewTenantUserService() *TenantUserService {
	return &TenantUserService{}
}

func (qq *TenantUserService) Create(
	ctx ctxx.Context,
	role tenantrole.TenantRole,
	email string,
	firstName string,
	lastName string,
	language language.Language,
	signInURL string,
) error {
	exists, err := ctx.TenantCtx().MainTx.Account.Query().
		Where(account.EmailEQ(entx.NewCIText(email))).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return e.NewHTTPErrorf(
			http.StatusConflict,
			"A user with this email address already exists, please contact support if you want to add this user anyway.",
		)
	}

	accountx := ctx.TenantCtx().MainTx.Account.Create().
		SetFirstName(firstName).
		SetLastName(lastName).
		SetLanguage(language).
		SetEmail(entx.NewCIText(email)).
		SetRole(mainrole.User).
		SaveX(ctx)

	_ = ctx.TenantCtx().MainTx.TenantAccountAssignment.Create().
		SetTenant(ctx.TenantCtx().Tenant).
		SetAccount(accountx).
		SetRole(role).
		SetIsOwningTenant(true).
		SetIsDefault(true).
		SaveX(ctx)

	ctx.TenantCtx().TTx.User.Create().
		SetAccountID(accountx.ID).
		SetRole(role).
		SetEmail(accountx.Email).
		SetFirstName(accountx.FirstName).
		SetLastName(accountx.LastName).
		SaveX(ctx)

	accountm := accountmodel.NewAccount(accountx)
	password, expiresAt, err := accountm.GenerateTemporaryPassword(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	mailer.NewMailer().CreateUser(
		ctx,
		accountx,
		password,
		expiresAt,
		signInURL,
	)

	return nil
}

func (qq *TenantUserService) Delete(
	ctx ctxx.Context,
	tenantID int64,
	userPublicID string,
	actingAccountID int64,
	actingUserID int64,
) (*RemoveAccountFromTenantResult, error) {
	userx := ctx.TenantCtx().TTx.User.Query().
		Where(user.PublicID(entx.NewCIText(userPublicID))).
		OnlyX(ctx)

	if userx.AccountID == actingAccountID {
		return nil, e.NewHTTPErrorf(
			http.StatusConflict,
			"You cannot delete your own user in organization management.",
		)
	}

	ctxWithPrivacyOverride := enttenantprivacy.DecisionContext(ctx, enttenantprivacy.Allow)

	ctx.TenantCtx().TTx.SpaceUserAssignment.Delete().
		Where(spaceuserassignment.UserID(userx.ID)).
		ExecX(ctxWithPrivacyOverride)

	ctx.TenantCtx().TTx.User.UpdateOneID(userx.ID).
		SetDeletedAt(time.Now()).
		SetDeletedBy(actingUserID).
		ExecX(ctx)

	result, err := NewTenantAccountLifecycleService().RemoveAccountFromTenant(
		ctx,
		tenantID,
		userx.AccountID,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return result, nil
}
