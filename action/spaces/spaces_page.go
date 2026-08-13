package spaces

import (
	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	"github.com/simpledms/simpledms/util/httpx"
)

type SpacesPageData struct {
}

type SpacesPageState struct {
}

// Spaces prefix makes it easier to search for file...
type SpacesPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewSpacesPage(infra *common.Infra, actions *Actions) *SpacesPage {
	return &SpacesPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *SpacesPage) Data() *SpacesPageData {
	return &SpacesPageData{}
}

func (qq *SpacesPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[SpacesPageState](rw, req)
	return qq.Render(rw, req, ctx, qq.infra, "Spaces", qq.Widget(ctx, state))
}

func (qq *SpacesPage) Widget(ctx ctxx.Context, state *SpacesPageState) renderable.Renderable {
	fabs := []*widget.FloatingActionButton{}

	if ctx.TenantCtx().User.Role == tenantrole.Owner {
		fabs = append(fabs,
			&widget.FloatingActionButton{
				Icon: "add",
				Child: []widget.IWidget{
					widget.NewIcon("add"),
					widget.T("Create space"),
				},
				HTMXAttrs: qq.actions.CreateSpaceDialog.ModalLinkAttrs(
					qq.actions.CreateSpaceDialog.Data("", ""),
					"",
				),
			},
		)
	}

	return &widget.MainLayout{
		Navigation: partial2.NewNavigationRail(
			ctx,
			qq.infra,
			partial2.SpacesNavigationRailValue(ctx.TenantCtx().TenantID),
			fabs,
		),
		Content: &widget.DefaultLayout{
			AppBar: qq.appBar(ctx),
			Content: qq.actions.SpaceCardsPartial.Widget(
				ctx,
			),
		},
	}
}

func (qq *SpacesPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "hub",
		},
		LeadingAltMobile: partial2.NewNavigationRailToggle(),
		Title: &widget.AppBarTitle{
			Text: widget.Tuf("%s «%s»", widget.T("Spaces").String(ctx), ctx.TenantCtx().Tenant.Name),
		},
		Actions: []widget.IWidget{
			/*&wx.IconButton{
				Icon: "more_vert",
				Children: &wx.Menu{
					Items: []*wx.MenuItem{}, // TODO
				},
			},*/
		},
	}
}
