package documenttype

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/library"
	"github.com/simpledms/simpledms/ui/renderable"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ImportFromLibraryDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[ImportFromLibraryCmdData]
}

func NewImportFromLibraryDialog(infra *common.Infra, actions *Actions) *ImportFromLibraryDialog {
	config := actionx.NewConfig(
		actions.Route("import-document-types-from-library-dialog"),
		false,
	)
	return &ImportFromLibraryDialog{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelperX[ImportFromLibraryCmdData](infra, config, wx.T("Import from library"), wx.T("Import")),
	}
}

func (qq *ImportFromLibraryDialog) Data() *ImportFromLibraryCmdData {
	return &ImportFromLibraryCmdData{}
}

func (qq *ImportFromLibraryDialog) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	return qq.FormHandler(rw, req, ctx)
}

func (qq *ImportFromLibraryDialog) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	wrapper := req.URL.Query().Get("wrapper")
	hxTarget := req.URL.Query().Get("hx-target")

	return qq.infra.Renderer().Render(rw, ctx,
		qq.Form(ctx, actionx.ResponseWrapper(wrapper), hxTarget),
	)
}

func (qq *ImportFromLibraryDialog) Form(
	ctx ctxx.Context,
	wrapper actionx.ResponseWrapper,
	hxTarget string,
) renderable.Renderable {
	service := library.NewService()
	if service.SpaceHasMetadata(ctx) {
		return autil.WrapWidget(
			wx.T("Import from library"),
			wx.T("Import"),
			wx.T("Import is only available for empty spaces."),
			wrapper,
			wx.DialogLayoutDefault,
		)
	}

	templates := library.BuiltinTemplates()

	form := &wx.Form{
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.actions.ImportFromLibraryCmd.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   "outerHTML",
		},
		Children: []wx.IWidget{
			&wx.Container{
				GapY: true,
				Child: []wx.IWidget{
					qq.libraryTemplateSelection(ctx, templates),
				},
			},
		},
	}

	return autil.WrapWidget(wx.T("Import from library"), wx.T("Import"), form, wrapper, wx.DialogLayoutDefault)
}

func (qq *ImportFromLibraryDialog) libraryTemplateSelection(ctx ctxx.Context, templates []library.BuiltinTemplate) wx.IWidget {
	if len(templates) == 0 {
		return &wx.EmptyState{
			Headline: wx.T("No library document types available yet."),
		}
	}

	var items []wx.IWidget
	items = append(items, wx.P("Select document types to import:"))

	for _, template := range templates {
		label := wx.T(template.Name).String(ctx)
		items = append(items, &wx.Checkbox{
			Label: wx.Tu(label),
			Name:  "library_template_keys",
			Value: template.Key,
		})
	}

	return &wx.Container{
		GapY:  true,
		Child: items,
	}
}
