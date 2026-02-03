package spaces

import (
	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
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
	fabs := []*wx.FloatingActionButton{}

	if ctx.TenantCtx().User.Role == tenantrole.Owner {
		fabs = append(fabs,
			&wx.FloatingActionButton{
				Icon: "add",
				Child: []wx.IWidget{
					wx.NewIcon("add"),
					wx.T("Create space"),
				},
				HTMXAttrs: qq.actions.CreateSpaceDialog.ModalLinkAttrs(
					qq.actions.CreateSpaceDialog.Data("", ""),
					"",
				),
			},
		)
	}

	return &wx.MainLayout{
		Navigation: partial2.NewNavigationRail(ctx, "spaces", fabs),
		Content: &wx.DefaultLayout{
			AppBar: qq.appBar(ctx),
			Content: qq.actions.SpaceCardsPartial.Widget(
				ctx,
			),
		},
	}
}

func (qq *SpacesPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "hub",
		},
		LeadingAltMobile: partial2.NewMainMenu(ctx),
		Title: &wx.AppBarTitle{
			Text: wx.Tuf("%s «%s»", wx.T("Spaces").String(ctx), ctx.TenantCtx().Tenant.Name),
		},
		Actions: []wx.IWidget{
			/*&wx.IconButton{
				Icon: "more_vert",
				Children: &wx.Menu{
					Items: []*wx.MenuItem{}, // TODO
				},
			},*/
		},
	}
}
