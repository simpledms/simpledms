package tenantuser

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	accountmodel "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/model/main/common/language"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/model/main/mailer"
	tenantmodel "github.com/simpledms/simpledms/model/main/tenant"
	"github.com/simpledms/simpledms/model/main/tenantmembership"
	usermodel "github.com/simpledms/simpledms/model/tenant/user"
	"github.com/simpledms/simpledms/util/e"
)

func Create(
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
	accountm := accountmodel.NewAccount(accountx)

	tenantm := tenantmodel.NewTenant(ctx.TenantCtx().Tenant)
	err = tenantm.AddAccountAssignment(ctx, accountm, role, true, true)
	if err != nil {
		return err
	}

	ctx.TenantCtx().TTx.User.Create().
		SetAccountID(accountx.ID).
		SetRole(role).
		SetEmail(accountx.Email).
		SetFirstName(accountx.FirstName).
		SetLastName(accountx.LastName).
		SaveX(ctx)

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

func Delete(
	ctx ctxx.Context,
	tenantID int64,
	userPublicID string,
	actingAccountID int64,
	actingUserID int64,
) (*tenantmembership.RemoveAccountFromTenantResult, error) {
	userx, err := ctx.TenantCtx().TTx.User.Query().
		Where(user.PublicID(entx.NewCIText(userPublicID))).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	if userx.AccountID == actingAccountID {
		return nil, e.NewHTTPErrorf(
			http.StatusConflict,
			"You cannot delete your own user in organization management.",
		)
	}

	userm := usermodel.NewUser(userx)
	err = userm.Delete(ctx, actingUserID)
	if err != nil {
		return nil, err
	}

	result, err := tenantmembership.RemoveAccountFromTenant(
		ctx,
		tenantID,
		userm.Data.AccountID,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return result, nil
}
