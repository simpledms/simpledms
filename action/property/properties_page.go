package property

import (
	acommon "github.com/marcobeierer/go-core/action/common"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/widget"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/partial"
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

func (qq *PropertiesPage) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	// state := autil.StateX[PropertiesPageState](rw, req)
	return qq.Render(rw, req, ctx, qq.infra, "Fields", qq.Widget(ctx))
}

func (qq *PropertiesPage) Widget(
	ctx ctxx.Context,
) renderable.Renderable {
	fabs := []*widget.FloatingActionButton{
		{
			Icon: "add",
			Child: []widget.IWidget{
				widget.NewIcon("add"),
				widget.T("Add field"),
			},
			HTMXAttrs: qq.actions.CreatePropertyCmd.ModalLinkAttrs(
				qq.actions.CreatePropertyCmd.Data(""),
				"",
			),
		},
	}

	var children []widget.IWidget

	children = append(children,
		qq.actions.PropertyListPartial.Widget(
			ctx,
			qq.actions.PropertyListPartial.Data(),
		),
	)

	mainLayout := &widget.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, qq.infra, "fields", fabs),
		Content: &widget.ListDetailLayout{
			AppBar: qq.appBar(ctx),
			List:   children,
		},
	}
	return mainLayout
}

func (qq *PropertiesPage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "tune",
		},
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title: &widget.AppBarTitle{
			Text: widget.T("Fields"),
		},
		Actions: []widget.IWidget{
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
