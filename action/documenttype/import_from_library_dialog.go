package documenttype

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/widget"
	actionx2 "github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/tenant/library"
)

type ImportFromLibraryDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
	*autil.FormHelper[ImportFromLibraryCmdData]
}

func NewImportFromLibraryDialog(infra *common.Infra, actions *Actions) *ImportFromLibraryDialog {
	config := actionx2.NewConfig(
		actions.Route("import-document-types-from-library-dialog"),
		false,
	)
	return &ImportFromLibraryDialog{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelperX[ImportFromLibraryCmdData](infra, config, widget.T("Import from library"), widget.T("Import")),
	}
}

func (qq *ImportFromLibraryDialog) Data() *ImportFromLibraryCmdData {
	return &ImportFromLibraryCmdData{}
}

func (qq *ImportFromLibraryDialog) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	return qq.FormHandler(rw, req, ctx)
}

func (qq *ImportFromLibraryDialog) FormHandler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	wrapper := req.URL.Query().Get("wrapper")
	hxTarget := req.URL.Query().Get("hx-target")

	return qq.infra.Renderer().Render(rw, ctx,
		qq.Form(ctx, actionx2.ResponseWrapper(wrapper), hxTarget),
	)
}

func (qq *ImportFromLibraryDialog) Form(
	ctx ctxx.Context,
	wrapper actionx2.ResponseWrapper,
	hxTarget string,
) renderable.Renderable {
	service := library.NewService()
	if service.SpaceHasMetadata(ctx) {
		return autil.WrapWidget(
			widget.T("Import from library"),
			widget.T("Import"),
			widget.T("Import is only available for empty spaces."),
			wrapper,
			widget.DialogLayoutDefault,
		)
	}

	templates := library.BuiltinTemplates()

	form := &widget.Form{
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.actions.ImportFromLibraryCmd.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   "outerHTML",
		},
		Children: []widget.IWidget{
			&widget.Container{
				GapY: true,
				Child: []widget.IWidget{
					qq.libraryTemplateSelection(ctx, templates),
				},
			},
		},
	}

	return autil.WrapWidget(widget.T("Import from library"), widget.T("Import"), form, wrapper, widget.DialogLayoutDefault)
}

func (qq *ImportFromLibraryDialog) libraryTemplateSelection(ctx ctxx.Context, templates []library.BuiltinTemplate) widget.IWidget {
	if len(templates) == 0 {
		return &widget.EmptyState{
			Headline: widget.T("No library document types available yet."),
		}
	}

	var items []widget.IWidget
	items = append(items, widget.P("Select document types to import:"))

	for _, template := range templates {
		label := widget.T(template.Name).String(ctx)
		items = append(items, &widget.Checkbox{
			Label: widget.Tu(label),
			Name:  "library_template_keys",
			Value: template.Key,
		})
	}

	return &widget.Container{
		GapY:  true,
		Child: items,
	}
}
