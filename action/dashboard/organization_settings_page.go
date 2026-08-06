package dashboard

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	accountmodel "github.com/simpledms/simpledms/model/main/account"
	tenantmodel "github.com/simpledms/simpledms/model/main/tenant"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
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

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(
			ctx,
			qq.infra,
			partial2.TenantSettingsNavigationRailValue(tenantID),
			nil,
		),
		Content: &widget.DefaultLayout{
			AppBar: qq.appBar(ctx),
			Content: &widget.Container{
				Widget: widget.Widget[widget.Container]{
					ID: "organizationSettings",
				},
				GapY: true,
				HTMXAttrs: widget.HTMXAttrs{
					HxGet:     route2.OrganizationSettings(tenantID),
					HxTrigger: event.HxTrigger(event.AccountUpdated),
					HxTarget:  "#content",
				},
				Child: qq.content(ctx),
			},
		},
	}
}

func (qq *OrganizationSettingsPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "settings",
		},
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.Tuf("%s «%s»", widget.T("Settings").String(ctx), ctx.TenantCtx().Tenant.Name),
		},
	}
}

func (qq *OrganizationSettingsPage) content(ctx ctxx.Context) widget.IWidget {
	button, ok := qq.passkeyEnforcementBtn(ctx, ctx.TenantCtx().Tenant)
	if !ok {
		return []*widget.Grid{}
	}

	return []*widget.Grid{{
		Heading: widget.H(widget.HeadingTypeTitleMd, widget.T("Passkeys")),
		Children: []*widget.Card{{
			Style:    widget.CardStyleFilled,
			Headline: widget.H(widget.HeadingTypeTitleLg, widget.T("Passkeys")),
			Actions:  []*widget.Button{button},
		}},
	}}
}

func (qq *OrganizationSettingsPage) passkeyEnforcementBtn(
	ctx ctxx.Context,
	tenantx *entmain.Tenant,
) (*widget.Button, bool) {
	tenantm := tenantmodel.NewTenant(tenantx)
	accountm := accountmodel.NewAccount(ctx.MainCtx().Account)
	if !tenantm.IsOwner(accountm) {
		return nil, false
	}
	if !tenantm.IsInitialized() {
		return nil, false
	}

	buttonLabel := widget.T("Enable passkey enforcement")
	confirmText := widget.T("Enable passkey enforcement for this organization? Members will need passkeys to sign in.")
	if tenantx.PasskeyAuthEnforced {
		buttonLabel = widget.T("Disable passkey enforcement")
		confirmText = widget.T("Disable passkey enforcement for this organization? Members can use passwords again if allowed.")
	}

	return &widget.Button{
		Label:     buttonLabel,
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: widget.HTMXAttrs{
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
