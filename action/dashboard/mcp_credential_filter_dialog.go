package dashboard

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type MCPCredentialFilterDialog struct {
	infra *common.Infra
	*actionx.Config
}

func NewMCPCredentialFilterDialog(
	infra *common.Infra,
	actions *Actions,
) *MCPCredentialFilterDialog {
	return &MCPCredentialFilterDialog{
		infra: infra,
		Config: actionx.NewConfig(
			actions.Route("mcp-credential-filter-dialog"),
			true,
		),
	}
}

func (qq *MCPCredentialFilterDialog) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	state := autil.StateX[MCPCredentialListPartialData](rw, req)
	if _, _, err := state.statusFilter(); err != nil {
		return err
	}
	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(state))
}

func (qq *MCPCredentialFilterDialog) Widget(
	state *MCPCredentialListPartialData,
) *widget.Dialog {
	showActive, showRevoked, _ := state.statusFilter()
	return &widget.Dialog{
		Widget: widget.Widget[widget.Dialog]{
			ID: qq.ID(),
		},
		Headline:     widget.T("Filter MCP credentials"),
		IsOpenOnLoad: true,
		Layout:       widget.DialogLayoutSideSheet,
		Child: &widget.Container{
			Widget: widget.Widget[widget.Container]{
				ID: "mcpCredentialStatusFilter",
			},
			Child: []*widget.FilterChip{
				newCredentialStatusFilterChip(
					widget.T("Active"),
					credentialStatusActive,
					showActive,
					event.MCPCredentialFilterChanged,
				),
				newCredentialStatusFilterChip(
					widget.T("Revoked"),
					credentialStatusRevoked,
					showRevoked,
					event.MCPCredentialFilterChanged,
				),
			},
		},
	}
}

func (qq *MCPCredentialFilterDialog) ID() string {
	return "mcpCredentialFilterDialog"
}
