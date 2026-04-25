package auth

import (
	"html/template"
	"net"

	acommon "github.com/simpledms/simpledms/core/action/common"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
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

func (qq *SignInPage) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	// duplicate in simpledms-saas.SignInPage
	hostname := req.Host
	if host, _, err := net.SplitHostPort(req.Host); err == nil {
		// err check is necessary because net.SplitHostPort returns an error
		// if req.Host does not contain a port
		hostname = host
	}
	isSafeRequest := hostname == "localhost" || req.TLS != nil
	if !isSafeRequest && !qq.infra.SystemConfig().AllowInsecureCookies() {
		rw.AddRenderables(widget.NewSnackbarf("Sign in only works over HTTPS or on localhost.").
			SetIsError(true).
			SetCustomAutoDismissTimeoutInMs(100000),
		)
	}

	return qq.Render(rw, req, ctx, qq.infra, "Sign in", qq.Widget(ctx))
}

func (qq *SignInPage) Widget(ctx ctxx.Context) *widget.NarrowLayout {
	// TODO link impressum

	var children []widget.IWidget

	children = append(children,
		widget.H(widget.HeadingTypeHeadlineMd, widget.Tuf("%s", widget.T("Sign in [subject]").String(ctx))),
		qq.actions.SignInCmd.Form(
			ctx,
			qq.actions.SignInCmd.Data("", ""),
			actionx.ResponseWrapperNone,
			widget.T("Sign in"),
			"",
		),
		&widget.Button{
			Label:     widget.T("Sign in with passkey"),
			StyleType: widget.ButtonStyleTypeElevated,
			HTMXAttrs: widget.HTMXAttrs{
				HxOn: &widget.HxOn{
					Event:   "click",
					Handler: template.JS("window.simpledmsPasskeySignIn(event)"),
				},
			},
		},
		qq.actions.ResetPasswordCmd.ModalLink(
			qq.actions.ResetPasswordCmd.Data(""),
			widget.T("Forgot password?"),
			"",
		),
		qq.actions.PasskeyRecoverySignInCmd.ModalLink(
			qq.actions.PasskeyRecoverySignInCmd.Data("", ""),
			widget.T("Use backup code"),
			"",
		),
	)

	column := &widget.Column{
		GapYSize:         widget.Gap4,
		NoOverflowHidden: true,
		Children:         children,
	}

	return &widget.NarrowLayout{
		Content: column,
		AppBar:  qq.appBar(ctx),
		Navigation: partial2.NewNavigationRail(
			ctx,
			qq.infra,
			"sign-in",
			nil,
		),
		WithPoweredBy: !qq.infra.SystemConfig().CommercialLicenseEnabled(),
	}
}

func (qq *SignInPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading:          &widget.Icon{Name: "folder_open"},
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
		Title:            &widget.AppBarTitle{Text: widget.Tu("SimpleDMS")},
		Actions:          []widget.IWidget{},
	}
}
