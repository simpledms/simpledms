package spaces

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	spacemodel "github.com/simpledms/simpledms/model/tenant/space"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type CreateSpaceCmdData struct {
	Name              string `validate:"required"`
	Description       string
	AddMeAsSpaceOwner bool
	// FolderMode  bool
	// TODO icon
}

type CreateSpaceCmdFormData struct {
	CreateSpaceCmdData  `structs:",flatten"`
	LibraryTemplateKeys []string `form:"library_template_keys"`
}

type CreateSpaceCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewCreateSpaceCmd(infra *common.Infra, actions *Actions) *CreateSpaceCmd {
	config := actionx.NewConfig(
		actions.Route("create-space-cmd"),
		false,
	)
	return &CreateSpaceCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *CreateSpaceCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreateSpaceCmdFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	_, err = spacemodel.Create(
		ctx,
		data.Name,
		data.Description,
		data.AddMeAsSpaceOwner,
		data.LibraryTemplateKeys,
	)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.SpaceCreated.String())
	rw.AddRenderables(wx.NewSnackbarf("Space «%s» created.", data.Name))

	return nil
}
