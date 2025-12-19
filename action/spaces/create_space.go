package spaces

import (
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/event"
	"github.com/simpledms/simpledms/model/common/spacerole"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateSpaceData struct {
	Name              string `validate:"required"`
	Description       string
	AddMeAsSpaceOwner bool
	// FolderMode  bool
	// TODO icon
}

type CreateSpace struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateSpaceData]
}

func NewCreateSpace(infra *common.Infra, actions *Actions) *CreateSpace {
	config := actionx.NewConfig(
		actions.Route("create-space"),
		false,
	)
	return &CreateSpace{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[CreateSpaceData](infra, config, wx.T("Create space")),
	}
}

func (qq *CreateSpace) Data(name, description string) *CreateSpaceData {
	return &CreateSpaceData{
		Name:        name,
		Description: description,
	}
}

func (qq *CreateSpace) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreateSpaceData](rw, req, ctx)
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
