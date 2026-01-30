package spaces

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type EditSpaceCmdData struct {
	SpaceID     string `validate:"required" form_attr_type:"hidden"`
	Name        string `validate:"required"`
	Description string
}

type EditSpaceCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditSpaceCmdData]
}

func NewRenameSpace(infra *common.Infra, actions *Actions) *EditSpaceCmd {
	config := actionx.NewConfig(actions.Route("edit-space-cmd"), false)
	return &EditSpaceCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[EditSpaceCmdData](infra, config, wx.T("Edit space")),
	}
}

func (qq *EditSpaceCmd) Data(spaceID string, name, description string) *EditSpaceCmdData {
	return &EditSpaceCmdData{
		SpaceID:     spaceID,
		Name:        name,
		Description: description,
	}
}

func (qq *EditSpaceCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditSpaceCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	// assumes it is on spaces screen, not dashboard
	ctx.TenantCtx().TTx.Space.Update().
		SetName(data.Name).
		SetDescription(data.Description).
		Where(space.PublicID(entx.NewCIText(data.SpaceID))).
		ExecX(ctx)

	spacex := ctx.TenantCtx().TTx.Space.Query().OnlyX(ctx)
	spaceCtx := ctxx.NewSpaceContext(ctx.TenantCtx(), spacex)

	ctx.TenantCtx().TTx.File.Update().
		SetName(data.Name).
		Where(
			file.SpaceID(spacex.ID),
			file.IsDirectory(true),
			file.IsRootDir(true),
		).
		ExecX(spaceCtx)

	rw.Header().Set("HX-Trigger", event.SpaceUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("Changes saved."))

	return nil
}
