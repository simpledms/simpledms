package spaces

import (
	autil "github.com/marcobeierer/go-core/action/util"

	acommon "github.com/marcobeierer/go-core/action/common"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/model/common/tenantrole"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/widget"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
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

func (qq *SpacesPage) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "spaces", fabs),
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
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
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
