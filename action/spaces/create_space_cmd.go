package spaces

import (
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/spacerole"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateSpaceCmdData struct {
	Name              string `validate:"required"`
	Description       string
	AddMeAsSpaceOwner bool
	// FolderMode  bool
	// TODO icon
}

type CreateSpaceCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateSpaceCmdData]
}

func NewCreateSpaceCmd(infra *common.Infra, actions *Actions) *CreateSpaceCmd {
	config := actionx.NewConfig(
		actions.Route("create-space"),
		false,
	)
	return &CreateSpaceCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[CreateSpaceCmdData](infra, config, wx.T("Create space")),
	}
}

func (qq *CreateSpaceCmd) Data(name, description string) *CreateSpaceCmdData {
	return &CreateSpaceCmdData{
		Name:        name,
		Description: description,
	}
}

func (qq *CreateSpaceCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreateSpaceCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	// set first space the user creates as default
	// isDefault := ctx.TenantCtx().TTx.Space.Query().CountX(ctx) == 0
	isDefault := false

	spacex := ctx.TenantCtx().TTx.Space.Create().
		SetName(data.Name).
		SetDescription(data.Description).
		SetIsFolderMode(true). // data.FolderMode).
		SaveX(ctx)

	// TODO could also use privacy.DecisionContext(ctx, privacy.Allow) instead
	spaceCtx := ctxx.NewSpaceContext(ctx.TenantCtx(), spacex)

	if data.AddMeAsSpaceOwner {
		_ = ctx.TenantCtx().TTx.SpaceUserAssignment.
			Create().
			SetSpaceID(spacex.ID).
			SetUserID(ctx.TenantCtx().User.ID).
			SetRole(spacerole.Owner).
			SetIsDefault(isDefault).
			SaveX(spaceCtx)

	}

	_ = ctx.TenantCtx().TTx.File.Create().
		SetName(data.Name).
		SetIsDirectory(true).
		SetIndexedAt(time.Now()).
		SetModifiedAt(time.Now()).
		SetSpaceID(spacex.ID).
		SetIsRootDir(true).
		SaveX(spaceCtx)

	// TODO is there a better way to do this? in combination with AddSpaceIDs
	/*ctx.TenantCtx().TTx.SpaceFileAssignment.Update().
	SetIsRootDir(true).
	Where(
		spacefileassignment.SpaceID(spacex.ID),
		spacefileassignment.FileID(rootDir.ID),
	).ExecX(ctx)
	*/

	rw.Header().Set("HX-Trigger", event.SpaceCreated.String())
	rw.AddRenderables(wx.NewSnackbarf("Space «%s» created.", data.Name))

	return nil
}
