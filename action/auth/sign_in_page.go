package auth

import (
	"net"

	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/country"
	"github.com/simpledms/simpledms/model/common/language"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type SignInPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewSignInPage(infra *common.Infra, actions *Actions) *SignInPage {
	return &SignInPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *SignInPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	// duplicate in simpledms-saas.SignInPage
	hostname := req.Host
	if host, _, err := net.SplitHostPort(req.Host); err == nil {
		// err check is necessary because net.SplitHostPort returns an error
		// if req.Host does not contain a port
		hostname = host
	}
	isSafeRequest := hostname == "localhost" || req.TLS != nil
	if !isSafeRequest && !qq.infra.SystemConfig().AllowInsecureCookies() {
		rw.AddRenderables(wx.NewSnackbarf("Sign in only works over HTTPS or on localhost.").
			SetIsError(true).
			SetCustomAutoDismissTimeoutInMs(100000),
		)
	}

	return qq.Render(rw, req, ctx, qq.infra, "Sign in", qq.Widget(ctx))
}

func (qq *SignInPage) Widget(ctx ctxx.Context) *wx.NarrowLayout {
	// TODO link impressum

	var children []wx.IWidget

	// TODO 2fa
	children = append(children,
		wx.H(wx.HeadingTypeHeadlineMd, wx.Tuf("%s", wx.T("Sign in [subject]").String(ctx))),
		qq.actions.SignInCmd.Form(
			ctx,
			qq.actions.SignInCmd.Data("", "", ""),
			actionx.ResponseWrapperNone,
			wx.T("Sign in"),
			"",
		),
		qq.actions.ResetPasswordCmd.ModalLink(
			qq.actions.ResetPasswordCmd.Data(""),
			wx.T("Forgot password?"),
			"",
		),
	)

	if qq.infra.SystemConfig().IsSaaSModeEnabled() {
		children = append(
			children,
			qq.actions.SignUpCmd.ModalLink(
				// TODO set country and language based on browser settings
				qq.actions.SignUpCmd.Data("", "", "", country.Unknown, language.Unknown, false),
				wx.T("Don't have an account? Sign up."),
				"",
			))
	}

	column := &wx.Column{
		GapYSize:         wx.Gap4,
		NoOverflowHidden: true,
		Children:         children,
	}

	return &wx.NarrowLayout{
		Content: column,
		/*Navigation: partial2.NewNavigationRail(
			ctx,
			qq.infra,
			"sign-in",
			nil,
		),*/
		WithPoweredBy: !qq.infra.SystemConfig().CommercialLicenseEnabled(),
	}
}
