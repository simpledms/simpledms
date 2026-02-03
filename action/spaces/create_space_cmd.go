package spaces

import (
	"log"
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/spacerole"
	"github.com/simpledms/simpledms/model/library"
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

func (qq *CreateSpaceCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreateSpaceCmdFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	// set first space the user creates as default
	// isDefault := ctx.TenantCtx().TTx.Space.Query().CountX(ctx) == 0
	isDefault := false

	spacex, err := ctx.TenantCtx().TTx.Space.Create().
		SetName(data.Name).
		SetDescription(data.Description).
		SetIsFolderMode(true). // data.FolderMode).
		Save(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	// TODO could also use privacy.DecisionContext(ctx, privacy.Allow) instead
	spaceCtx := ctxx.NewSpaceContext(ctx.TenantCtx(), spacex)

	if data.AddMeAsSpaceOwner {
		_, err := ctx.TenantCtx().TTx.SpaceUserAssignment.
			Create().
			SetSpaceID(spacex.ID).
			SetUserID(ctx.TenantCtx().User.ID).
			SetRole(spacerole.Owner).
			SetIsDefault(isDefault).
			Save(spaceCtx)
		if err != nil {
			return err
		}

	}

	_, err = ctx.TenantCtx().TTx.File.Create().
		SetName(data.Name).
		SetIsDirectory(true).
		SetIndexedAt(time.Now()).
		SetModifiedAt(time.Now()).
		SetSpaceID(spacex.ID).
		SetIsRootDir(true).
		Save(spaceCtx)
	if err != nil {
		return err
	}

	if len(data.LibraryTemplateKeys) > 0 {
		service := library.NewService()
		err = service.ImportBuiltinDocumentTypes(spaceCtx, data.LibraryTemplateKeys, false)
		if err != nil {
			return err
		}
	}

	rw.Header().Set("HX-Trigger", event.SpaceCreated.String())
	rw.AddRenderables(wx.NewSnackbarf("Space «%s» created.", data.Name))

	return nil
}
