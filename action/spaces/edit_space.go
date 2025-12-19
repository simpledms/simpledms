package spaces

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type EditSpaceData struct {
	SpaceID     string `validate:"required" form_attr_type:"hidden"`
	Name        string `validate:"required"`
	Description string
}

type EditSpace struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditSpaceData]
}

func NewRenameSpace(infra *common.Infra, actions *Actions) *EditSpace {
	config := actionx.NewConfig(actions.Route("edit-space"), false)
	return &EditSpace{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[EditSpaceData](infra, config, wx.T("Edit space")),
	}
}

func (qq *EditSpace) Data(spaceID string, name, description string) *EditSpaceData {
	return &EditSpaceData{
		SpaceID:     spaceID,
		Name:        name,
		Description: description,
	}
}

func (qq *EditSpace) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[EditSpaceData](rw, req, ctx)
	if err != nil {
		return err
	}

	// assumes it is on spaces screen, not dashboard
	ctx.TenantCtx().TTx.Space.Update().
		SetName(data.Name).
		SetDescription(data.Description).
		Where(space.PublicID(entx.NewCIText(data.SpaceID))).
		ExecX(ctx)

	rw.Header().Set("HX-Trigger", event.SpaceUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("Changes saved."))

	return nil
}
