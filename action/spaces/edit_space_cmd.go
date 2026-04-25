package spaces

import (
	"github.com/simpledms/simpledms/core/db/entx"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/space"
	spacemodel "github.com/simpledms/simpledms/model/tenant/space"
	"github.com/simpledms/simpledms/ui/uix/event"
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
		FormHelper: autil.NewFormHelper[EditSpaceCmdData](infra, config, widget.T("Edit space")),
	}
}

func (qq *EditSpaceCmd) Data(spaceID string, name, description string) *EditSpaceCmdData {
	return &EditSpaceCmdData{
		SpaceID:     spaceID,
		Name:        name,
		Description: description,
	}
}

func (qq *EditSpaceCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditSpaceCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	spacex, err := ctx.TenantCtx().TTx.Space.Query().
		Where(space.PublicID(entx.NewCIText(data.SpaceID))).
		Only(ctx)
	if err != nil {
		return err
	}

	err = spacemodel.NewSpace(spacex).Edit(ctx, data.Name, data.Description)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.SpaceUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("Changes saved."))

	return nil
}
