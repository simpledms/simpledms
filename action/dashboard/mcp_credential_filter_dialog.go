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
	return newCredentialStatusFilterDialog(
		qq.ID(),
		widget.T("Filter MCP credentials"),
		"mcpCredentialStatusFilter",
		state.CredentialStatusValues,
		event.MCPCredentialFilterChanged,
	)
}

func (qq *MCPCredentialFilterDialog) ID() string {
	return "mcpCredentialFilterDialog"
}
