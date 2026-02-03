package spaces

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
		FormHelper: autil.NewFormHelper[CreateSpaceCmdData](infra, config, wx.T("Create space")),
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

	form := &wx.Form{
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.actions.CreateSpaceCmd.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   "outerHTML",
		},
		Children: []wx.IWidget{
			&wx.Container{
				GapY: true,
				Child: []wx.IWidget{
					wx.NewFormFields(ctx, data),
					libraryTemplateSection(ctx, templates),
				},
			},
		},
	}

	return autil.WrapWidget(wx.T("Create space"), wx.T("Save"), form, wrapper, wx.DialogLayoutDefault)
}

func libraryTemplateSection(ctx ctxx.Context, templates []library.BuiltinTemplate) wx.IWidget {
	if len(templates) == 0 {
		return &wx.EmptyState{
			Headline: wx.T("No library document types available yet."),
		}
	}

	var items []wx.IWidget
	items = append(items, wx.P("Select document types to add to this space:"))

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
