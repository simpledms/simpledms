package tagging

// package action

import (
	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/enttenant/tagassignment"
	"github.com/simpledms/simpledms/app/simpledms/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ToggleFileTagData struct {
	FileID int64
	TagID  int64
}

// this is just a command, not a component
type ToggleFileTag struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewToggleFileTag(infra *common.Infra, actions *Actions) *ToggleFileTag {
	config := actionx.NewConfig(
		actions.Route("toggle-file-tag"),
		false,
	)
	return &ToggleFileTag{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ToggleFileTag) Data(fileID int64, tagID int64) *ToggleFileTagData {
	return &ToggleFileTagData{
		FileID: fileID,
		TagID:  tagID,
	}
}

func (qq *ToggleFileTag) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ToggleFileTagData](rw, req, ctx)
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
