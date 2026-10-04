package dashboard

import (
	"log"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
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
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	state := autil.StateX[MCPCredentialListPartialData](rw, req)
	page, err := qq.Widget(ctx, req, state)
	if err != nil {
		log.Println(err)
		return err
	}
	rw.Header().Set("Cache-Control", "no-store")
	return qq.Render(rw, req, ctx, qq.infra, "MCP credentials", page)
}

func (qq *MCPCredentialsPage) Widget(
	ctx ctxx.Context,
	req *httpx.Request,
	data *MCPCredentialListPartialData,
) (renderable.Renderable, error) {
	overview, err := qq.actions.MCPCredentialListPartial.Widget(
		ctx,
		req,
		qq.actions.MCPCredentialListPartial.Data("", data.CredentialStatusValues...),
	)
	if err != nil {
		return nil, err
	}
	fabs := []*widget.FloatingActionButton{{
		Icon: "add",
		Child: []widget.IWidget{
			widget.NewIcon("add"),
			widget.T("Create MCP credential"),
		},
		HTMXAttrs: qq.actions.CreateMCPCredentialCmd.ModalLinkAttrs(
			qq.actions.CreateMCPCredentialCmd.Data(""),
		),
	}}

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(
			ctx.MainCtx(),
			qq.infra,
			"mcp-credentials",
			fabs,
		),
		Content: &widget.View{
			Children: []widget.IWidget{
				&widget.ListDetailLayout{
					AppBar: qq.appBar(data),
					List:   overview,
				},
				autil.DefaultSideSheetTrigger("mcpCredentialsDefaultSideSheetTrigger", widget.HTMXAttrs{
					HxPost: qq.actions.MCPCredentialFilterDialog.Endpoint(),
				}),
			},
		},
	}, nil
}

func (qq *MCPCredentialsPage) appBar(data *MCPCredentialListPartialData) *widget.AppBar {
	return &widget.AppBar{
		Leading:          widget.NewIcon("smart_toy"),
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.T("MCP credentials"),
		},
		Actions: []widget.IWidget{qq.filterButton(data, false)},
	}
}

func (qq *MCPCredentialsPage) filterButton(
	data *MCPCredentialListPartialData,
	isOOB bool,
) *widget.Container {
	return newCredentialFilterButton(
		"mcpCredentialFilterButton",
		widget.T("Filter MCP credentials"),
		qq.actions.MCPCredentialFilterDialog.Endpoint(),
		data.CredentialStatusValues,
		isOOB,
	)
}
