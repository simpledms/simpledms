package auth

import (
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
	return qq.Render(rw, req, ctx, qq.infra, "Sign in", qq.Widget(ctx))
}

func (qq *SignInPage) Widget(ctx ctxx.Context) *wx.NarrowLayout {
	// impressum
	// forget password
	// sign in button
	// sign up

	// TODO 2fa
	children := []wx.IWidget{
		wx.H(wx.HeadingTypeHeadlineMd, wx.Tuf("%s | SimpleDMS", wx.T("Sign in [subject]").String(ctx))),
		qq.actions.SignIn.Form(
			ctx,
			qq.actions.SignIn.Data("", "", ""),
			actionx.ResponseWrapperNone,
			wx.T("Sign in"),
			"",
		),
		qq.actions.ResetPassword.ModalLink(
			qq.actions.ResetPassword.Data(""),
			wx.T("Forget password?"),
			"",
		),
	}

	if qq.infra.SystemConfig().IsSaaSModeEnabled() {
		children = append(
			children,
			qq.actions.SignUp.ModalLink(
				// TODO set country and language based on browser settings
				qq.actions.SignUp.Data("", "", "", country.Unknown, language.Unknown, false),
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
	}
}
