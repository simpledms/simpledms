package tenantuser

import (
	"log"
	"net/http"

	accountmodel "github.com/simpledms/simpledms/core/model/account"
	"github.com/simpledms/simpledms/core/model/common/language"
	"github.com/simpledms/simpledms/core/model/common/tenantrole"
	"github.com/simpledms/simpledms/core/model/mailer"
	tenantmodel "github.com/simpledms/simpledms/core/model/tenant"
	"github.com/simpledms/simpledms/core/model/tenantmembership"
	"github.com/simpledms/simpledms/core/util/e"
	"github.com/simpledms/simpledms/ctxx"
	usermodel "github.com/simpledms/simpledms/model/tenant/user"
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
	repository := NewEntTenantUserRepository()

	exists, err := repository.AccountExistsByEmail(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		return e.NewHTTPErrorf(
			http.StatusConflict,
			"A user with this email address already exists, please contact support if you want to add this user anyway.",
		)
	}

	accountx, err := repository.CreateAccount(ctx, firstName, lastName, language, email)
	if err != nil {
		return err
	}
	accountm := accountmodel.NewAccount(accountx)

	tenantm := tenantmodel.NewTenant(ctx.TenantCtx().Tenant)
	err = tenantm.AddAccountAssignment(ctx, accountm, role, true, true)
	if err != nil {
		return err
	}

	_, err = repository.CreateTenantUser(ctx, accountx, role)
	if err != nil {
		return err
	}

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
	repository := NewEntTenantUserRepository()

	userx, err := repository.UserByPublicID(ctx, userPublicID)
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
