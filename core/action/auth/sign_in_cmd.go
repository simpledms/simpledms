package auth

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/core/db/entmain"
	"github.com/simpledms/simpledms/core/db/entmain/account"
	"github.com/simpledms/simpledms/core/db/entx"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	account3 "github.com/simpledms/simpledms/core/model/account"
	"github.com/simpledms/simpledms/core/model/common/mainrole"
	tenantaccessmodel "github.com/simpledms/simpledms/core/model/tenantaccess"
	"github.com/simpledms/simpledms/core/ui/uix/route"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type SignInCmdData struct {
	Email    string `validate:"required,email" form_attrs:"autofocus"`
	Password string `validate:"required" form_attr_type:"password"`
	// TODO show to user how long it would be valid (x hours or if session ends, whatever comes first) or sign out
	TemporarySession bool
}

type SignInCmd struct {
	infra              *common.Infra
	actions            *Actions
	requestRateLimiter *account3.RequestRateLimiter
	*actionx.Config
	*autil.FormHelper[SignInCmdData]
}

func NewSignInCmd(
	infra *common.Infra,
	actions *Actions,
	requestRateLimiter *account3.RequestRateLimiter,
) *SignInCmd {
	config := actionx.NewConfig(
		actions.Route("sign-in-cmd"),
		false,
	)
	return &SignInCmd{
		infra:              infra,
		actions:            actions,
		requestRateLimiter: requestRateLimiter,
		Config:             config,
		FormHelper:         autil.NewFormHelper[SignInCmdData](infra, config, widget.T("Sign in")),
	}
}

func (qq *SignInCmd) Data(email, password string) *SignInCmdData {
	return &SignInCmdData{
		Email:    email,
		Password: password,
	}
}

func (qq *SignInCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SignInCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if !qq.requestRateLimiter.Allow(
		rateLimitKey("sign-in-ip", clientIPFromRequest(req)),
		signInRateLimitWindow,
		signInRateLimitPerIP,
	) {
		return e.NewHTTPErrorf(http.StatusTooManyRequests, "Too many sign-in attempts. Please try again shortly.")
	}

	// not perfect, a real user can be locked out...
	if !qq.requestRateLimiter.Allow(
		rateLimitKey("sign-in-email", normalizeRateLimitedEmail(data.Email)),
		signInRateLimitWindow,
		signInRateLimitPerEmail,
	) {
		return e.NewHTTPErrorf(http.StatusTooManyRequests, "Too many sign-in attempts. Please try again shortly.")
	}

	// not checked if tenant is already initialized because tenant is independent of user;
	// user can belong to multiple tenants

	accountx, err := ctx.VisitorCtx().MainTx.Account.Query().Where(account.Email(entx.NewCIText(data.Email))).Only(ctx)
	if err != nil {
		if entmain.IsNotFound(err) {
			rw.AddRenderables(widget.NewSnackbarf("Invalid credentials. Please try again."))
			return nil
		}
		log.Println(err)
		return err
	}
	accountm := account3.NewAccount(accountx)
	passkeyPolicy, err := accountm.PasskeyPolicy(ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	isTenantPasskeyEnrollmentRequired := passkeyPolicy.IsTenantPasskeyEnrollmentRequired()

	isValid, err := accountm.AuthWithPasskeyPolicy(ctx, data.Password, passkeyPolicy)
	if !isValid {
		rw.AddRenderables(widget.NewSnackbarf("Invalid credentials. Please try again."))
		return err
	}

	if qq.infra.SystemConfig().IsSaaSModeEnabled() && accountx.Role == mainrole.User {
		hasActiveTenantAssignment, err := tenantaccessmodel.NewTenantAccessService().HasActiveTenantAssignment(
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
				widget.NewSnackbarf("Your organization is no longer active. Please contact support."),
			)
		}
	}

	err = createAccountSession(
		rw,
		req,
		ctx,
		accountx,
		data.TemporarySession || isTenantPasskeyEnrollmentRequired,
		qq.infra.SystemConfig().AllowInsecureCookies(),
	)
	if err != nil {
		log.Println(err)
		return err
	}

	if isTenantPasskeyEnrollmentRequired {
		rw.AddRenderables(widget.NewSnackbarf("Passkey setup is required by your organization. Register a passkey now."))
	} else {
		rw.AddRenderables(widget.NewSnackbarf("Logged in successfully."))
	}

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
