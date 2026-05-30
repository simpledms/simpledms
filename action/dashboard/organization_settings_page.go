package dashboard

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	accountmodel "github.com/simpledms/simpledms/model/main/account"
	tenantmodel "github.com/simpledms/simpledms/model/main/tenant"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/httpx"
)

type OrganizationSettingsPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewOrganizationSettingsPage(
	infra *common.Infra,
	actions *Actions,
) *OrganizationSettingsPage {
	return &OrganizationSettingsPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *OrganizationSettingsPage) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	return qq.Render(rw, req, ctx, qq.infra, "Settings", qq.Widget(ctx))
}

func (qq *OrganizationSettingsPage) Widget(ctx ctxx.Context) renderable.Renderable {
	tenantID := ctx.TenantCtx().TenantID

	return &wx.MainLayout{
		Navigation: partial2.NewNavigationRail(
			ctx,
			qq.infra,
			partial2.TenantSettingsNavigationRailValue(tenantID),
			nil,
		),
		Content: &wx.DefaultLayout{
			AppBar: qq.appBar(ctx),
			Content: &wx.Container{
				Widget: wx.Widget[wx.Container]{
					ID: "organizationSettings",
				},
				GapY: true,
				HTMXAttrs: wx.HTMXAttrs{
					HxGet:     route2.OrganizationSettings(tenantID),
					HxTrigger: event.HxTrigger(event.AccountUpdated),
					HxTarget:  "#content",
				},
				Child: qq.content(ctx),
			},
		},
	}
}

func (qq *OrganizationSettingsPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "settings",
		},
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &wx.AppBarTitle{
			Text: wx.Tuf("%s «%s»", wx.T("Settings").String(ctx), ctx.TenantCtx().Tenant.Name),
		},
	}
}

func (qq *OrganizationSettingsPage) content(ctx ctxx.Context) wx.IWidget {
	button, ok := qq.passkeyEnforcementBtn(ctx, ctx.TenantCtx().Tenant)
	if !ok {
		return []*wx.Grid{}
	}

	return []*wx.Grid{{
		Heading: wx.H(wx.HeadingTypeTitleMd, wx.T("Passkeys")),
		Children: []*wx.Card{{
			Style:    wx.CardStyleFilled,
			Headline: wx.H(wx.HeadingTypeTitleLg, wx.T("Passkeys")),
			Actions:  []*wx.Button{button},
		}},
	}}
}

func (qq *OrganizationSettingsPage) passkeyEnforcementBtn(
	ctx ctxx.Context,
	tenantx *entmain.Tenant,
) (*wx.Button, bool) {
	tenantm := tenantmodel.NewTenant(tenantx)
	accountm := accountmodel.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return nil, false
	}
	if !tenantm.IsInitialized() {
		return nil, false
	}

	buttonLabel := wx.T("Enable passkey enforcement")
	confirmText := wx.T("Enable passkey enforcement for this organization? Members will need passkeys to sign in.")
	if tenantx.PasskeyAuthEnforced {
		buttonLabel = wx.T("Disable passkey enforcement")
		confirmText = wx.T("Disable passkey enforcement for this organization? Members can use passwords again if allowed.")
	}

	return &wx.Button{
		Label:     buttonLabel,
		StyleType: wx.ButtonStyleTypeElevated,
		HTMXAttrs: wx.HTMXAttrs{
			HxPost: qq.actions.ToggleTenantPasskeyEnforcementCmd.Endpoint(),
			HxVals: util.JSON(
				qq.actions.ToggleTenantPasskeyEnforcementCmd.Data(
					tenantx.PublicID.String(),
					!tenantx.PasskeyAuthEnforced,
				),
			),
			HxConfirm: confirmText.String(ctx),
			HxSwap:    "none",
		},
	}, true
}
