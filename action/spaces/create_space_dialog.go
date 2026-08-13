package spaces

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/tenant/library"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateSpaceDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateSpaceCmdData]
}

func NewCreateSpaceDialog(infra *common.Infra, actions *Actions) *CreateSpaceDialog {
	config := actionx.NewConfig(
		actions.Route("create-space-dialog"),
		true,
	).SetUsesSeparatedCmd(true)
	return &CreateSpaceDialog{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[CreateSpaceCmdData](infra, config, widget.T("Create space")),
	}
}

func (qq *CreateSpaceDialog) Data(name, description string) *CreateSpaceCmdData {
	return &CreateSpaceCmdData{
		Name:        name,
		Description: description,
	}
}

func (qq *CreateSpaceDialog) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	return qq.FormHandler(rw, req, ctx)
}

func (qq *CreateSpaceDialog) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormDataX[CreateSpaceCmdData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")
	hxTarget := req.URL.Query().Get("hx-target")

	return qq.infra.Renderer().Render(rw, ctx,
		qq.Form(
			ctx,
			data,
			actionx.ResponseWrapper(wrapper),
			hxTarget,
		),
	)
}

func (qq *CreateSpaceDialog) Form(
	ctx ctxx.Context,
	data *CreateSpaceCmdData,
	wrapper actionx.ResponseWrapper,
	hxTarget string,
) renderable.Renderable {
	templates := library.BuiltinTemplates()

	form := &widget.Form{
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.actions.CreateSpaceCmd.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   "outerHTML",
		},
		Children: []widget.IWidget{
			&widget.Container{
				GapY: true,
				Child: []widget.IWidget{
					widget.NewFormFields(ctx, data),
					libraryTemplateSection(ctx, templates),
				},
			},
		},
	}

	return autil.WrapWidget(widget.T("Create space"), widget.T("Save"), form, wrapper, widget.DialogLayoutDefault)
}

func libraryTemplateSection(ctx ctxx.Context, templates []library.BuiltinTemplate) widget.IWidget {
	if len(templates) == 0 {
		return &widget.EmptyState{
			Headline: widget.T("No library document types available yet."),
		}
	}

	var items []widget.IWidget
	items = append(items, widget.P("Select document types to add to this space:"))

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
