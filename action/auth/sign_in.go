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
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/language"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/cookiex"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type SignInData struct {
	Email                       string `validate:"required,email" form_attrs:"autofocus"`
	Password                    string `validate:"required" form_attr_type:"password"`
	TwoFactorAuthenticationCode string `form_attr_type:"hidden"`
	// TODO show to user how long it would be valid (x hours or if session ends, whatever comes first) or sign out
	TemporarySession bool
}

type SignIn struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[SignInData]
}

func NewSignIn(infra *common.Infra, actions *Actions) *SignIn {
	config := actionx.NewConfig(
		actions.Route("sign-in"),
		false,
	)
	return &SignIn{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[SignInData](infra, config, wx.T("Sign in")),
	}
}

func (qq *SignIn) Data(email, password, twoFactorAuthenticationCode string) *SignInData {
	return &SignInData{
		Email:                       email,
		Password:                    password,
		TwoFactorAuthenticationCode: twoFactorAuthenticationCode,
	}
}

func (qq *SignIn) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SignInData](rw, req, ctx)
	if err != nil {
		return err
	}

	// not checked if tenant is already initialized because tenant is independent of user;
	// user can belong to multiple tenants

	accountx, err := ctx.VisitorCtx().MainTx.Account.Query().Where(account.Email(entx.NewCIText(data.Email))).Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			snackbar := wx.NewSnackbarf("Found no account for this email address.").
				WithAction(
					qq.actions.SignUp.ModalLink(
						qq.actions.SignUp.Data("", "", "", country.Unknown, language.Unknown, false),
						wx.T("Sign up now."),
						""),
				)
			return e.NewHTTPErrorWithSnackbar(http.StatusBadRequest, snackbar)
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

	cookie, err := cookiex.SetSessionCookie(rw, req, data.TemporarySession)
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
