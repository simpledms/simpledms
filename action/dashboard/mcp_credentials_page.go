package dashboard

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type MCPCredentialsPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewMCPCredentialsPage(infra *common.Infra, actions *Actions) *MCPCredentialsPage {
	return &MCPCredentialsPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *MCPCredentialsPage) Handler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	list, err := qq.actions.MCPCredentialListPartial.Widget(ctx, 0)
	if err != nil {
		return err
	}
	rw.Header().Set("Cache-Control", "no-store")
	return qq.Render(rw, req, ctx, qq.infra, "MCP credentials", &widget.MainLayout{
		Navigation: partial.NewNavigationRail(ctx.MainCtx(), qq.infra, "account", nil),
		Content: &widget.DefaultLayout{
			AppBar: &widget.AppBar{
				Title:            &widget.AppBarTitle{Text: widget.T("MCP credentials")},
				Leading:          widget.NewIcon("vpn_key"),
				LeadingAltMobile: partial.NewNavigationRailToggle(),
			},
			Content: &widget.Column{
				GapYSize: widget.Gap3,
				Children: []widget.IWidget{
					&widget.Button{
						Label:     widget.T("Create MCP credential"),
						Icon:      widget.NewIcon("add"),
						StyleType: widget.ButtonStyleTypeFilled,
						HTMXAttrs: widget.HTMXAttrs{
							Role: "button",
							HxPost: qq.actions.CreateMCPCredentialCmd.FormEndpointWithParams(
								actionx.ResponseWrapperDialog, "closest dialog",
							),
							LoadInPopover: true,
						},
					},
					list,
				},
			},
		},
	})
}
