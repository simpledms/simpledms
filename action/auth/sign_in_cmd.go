package auth

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/entmain/account"
	"github.com/simpledms/simpledms/db/entx"
	account2 "github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/model/common/mainrole"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type SignInCmdData struct {
	Email                       string `validate:"required,email" form_attrs:"autofocus"`
	Password                    string `validate:"required" form_attr_type:"password"`
	TwoFactorAuthenticationCode string `form_attr_type:"hidden"`
	// TODO show to user how long it would be valid (x hours or if session ends, whatever comes first) or sign out
	TemporarySession bool
}

type SignInCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[SignInCmdData]
}

func NewSignInCmd(infra *common.Infra, actions *Actions) *SignInCmd {
	config := actionx.NewConfig(
		actions.Route("sign-in-cmd"),
		false,
	)
	return &SignInCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[SignInCmdData](infra, config, wx.T("Sign in")),
	}
}

func (qq *SignInCmd) Data(email, password, twoFactorAuthenticationCode string) *SignInCmdData {
	return &SignInCmdData{
		Email:                       email,
		Password:                    password,
		TwoFactorAuthenticationCode: twoFactorAuthenticationCode,
	}
}

func (qq *SignInCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SignInCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	// not checked if tenant is already initialized because tenant is independent of user;
	// user can belong to multiple tenants

	accountx, err := ctx.VisitorCtx().MainTx.Account.Query().Where(account.Email(entx.NewCIText(data.Email))).Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return e.NewHTTPErrorWithSnackbar(
				http.StatusBadRequest,
				wx.NewSnackbarf("Found no account for this email address."),
			)
		}
		log.Println(err)
		return err
	}
	accountm := account2.NewAccount(accountx)

	isValid, err := accountm.Auth(ctx, data.Password, data.TwoFactorAuthenticationCode)
	if !isValid {
		rw.AddRenderables(wx.NewSnackbarf("Invalid credentials. Please try again."))
		return err
	}

	if qq.infra.SystemConfig().IsSaaSModeEnabled() && accountx.Role == mainrole.User {
		hasActiveTenantAssignment, err := modelmain.NewTenantAccessService().HasActiveTenantAssignment(
			ctx,
			ctx.VisitorCtx().MainTx,
			accountx.ID,
		)
		if err != nil {
			log.Println(err)
			return e.NewHTTPErrorf(http.StatusInternalServerError, "Could not verify organization access.")
		}

		if !hasActiveTenantAssignment {
			return e.NewHTTPErrorWithSnackbar(
				http.StatusForbidden,
				wx.NewSnackbarf("Your organization is no longer active. Please contact support."),
			)
		}
	}

	cookie, err := cookiex.SetSessionCookie(rw, req, data.TemporarySession, qq.infra.SystemConfig().AllowInsecureCookies())
	if err != nil {
		log.Println(err)
		return err
	}

	deletableAt := cookiex.DeletableAt(cookie)

	ctx.VisitorCtx().MainTx.Session.
		Create().
		SetAccountID(accountx.ID).
		SetValue(cookie.Value).
		SetIsTemporarySession(data.TemporarySession).
		SetDeletableAt(deletableAt).
		SetExpiresAt(cookie.Expires).
		SaveX(ctx)

	// TODO not shown
	rw.AddRenderables(wx.NewSnackbarf("Logged in successfully."))

	rw.Header().Set("HX-Redirect", route.Dashboard())

	// TODO redirect or just render content
	// TODO redirect seems not to work
	//
	// duplicate code in Router
	/*
		tenantx, err := accountx.
			QueryTenantAssignment().
			Where(tenantaccountassignment.IsDefault(true)).
			QueryTenant().
			Where(tenant.InitializedAtNotNil()).
			Only(ctx)
		if err != nil {
			rw.Header().Set("HX-Redirect", route.Dashboard())
		} else {
			rw.Header().Set("HX-Redirect", route.SpacesRoot(tenantx.PublicID.String()))
		}
	*/

	return nil
}
