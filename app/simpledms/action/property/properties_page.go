package property

import (
	acommon "github.com/simpledms/simpledms/app/simpledms/action/common"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/renderable"
	"github.com/simpledms/simpledms/app/simpledms/ui/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/httpx"
)

type PropertiesPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewPropertiesPage(infra *common.Infra, actions *Actions) *PropertiesPage {
	return &PropertiesPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *PropertiesPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	// state := autil.StateX[PropertiesPageState](rw, req)
	return qq.Render(rw, req, ctx, qq.infra, "Fields", qq.Widget(ctx))
}

func (qq *PropertiesPage) Widget(
	ctx ctxx.Context,
) renderable.Renderable {
	fabs := []*wx.FloatingActionButton{
		{
			Icon: "add",
			Child: []wx.IWidget{
				wx.NewIcon("add"),
				wx.T("Add field"),
			},
			HTMXAttrs: qq.actions.CreateProperty.ModalLinkAttrs(
				qq.actions.CreateProperty.Data(""),
				"",
			),
		},
	}

	var children []wx.IWidget

	children = append(children,
		qq.actions.PropertyList.Widget(
			ctx,
			qq.actions.PropertyList.Data(),
		),
	)

	mainLayout := &wx.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, "properties", fabs),
		Content: &wx.ListDetailLayout{
			AppBar: qq.appBar(ctx),
			List:   children,
		},
	}
	return mainLayout
}

func (qq *PropertiesPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "tune",
		},
		LeadingAltMobile: partial.NewMainMenu(ctx),
		Title: &wx.AppBarTitle{
			Text: wx.T("Fields"),
		},
		Actions: []wx.IWidget{
			/*&wx.IconButton{
				Icon: "more_vert",
				Children: &wx.Menu{
					Items: []*wx.MenuItem{}, // TODO
				},
			},
			*/
		},
	}
}
