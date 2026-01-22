package tagging

// package action

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ToggleFileTagCmdData struct {
	FileID int64
	TagID  int64
}

// this is just a command, not a component
type ToggleFileTagCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewToggleFileTagCmd(infra *common.Infra, actions *Actions) *ToggleFileTagCmd {
	config := actionx.NewConfig(
		actions.Route("toggle-file-tag-cmd"),
		false,
	)
	return &ToggleFileTagCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ToggleFileTagCmd) Data(fileID int64, tagID int64) *ToggleFileTagCmdData {
	return &ToggleFileTagCmdData{
		FileID: fileID,
		TagID:  tagID,
	}
}

func (qq *ToggleFileTagCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ToggleFileTagCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := ctx.TenantCtx().TTx.File.GetX(ctx, data.FileID)
	tagx := ctx.TenantCtx().TTx.Tag.GetX(ctx, data.TagID)
	isSelected := filex.QueryTagAssignment().Where(tagassignment.TagID(data.TagID)).ExistX(ctx)

	var snackbar *wx.Snackbar

	// TODO move logic to model
	if isSelected {
		ctx.TenantCtx().TTx.TagAssignment.
			Delete().
			Where(
				tagassignment.FileID(data.FileID),
				tagassignment.TagID(data.TagID),
				tagassignment.SpaceID(ctx.SpaceCtx().Space.ID),
			).
			ExecX(ctx)
		snackbar = wx.NewSnackbarf("«%s» unassigned.", tagx.Name)
	} else {
		ctx.TenantCtx().TTx.TagAssignment.Create().
			SetFileID(data.FileID).
			SetTagID(data.TagID).
			SetSpaceID(ctx.SpaceCtx().Space.ID).
			// SetIsInherited(false).
			SaveX(ctx)
		snackbar = wx.NewSnackbarf("«%s» assigned.", tagx.Name)
	}

	// must be set before writing to rw
	rw.Header().Set("HX-Trigger", event.TagUpdated.String())

	// TODO is this necessary or should caller decide?
	// req.Header.Set("HX-Reswap", "none")

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		snackbar,
	)
}
