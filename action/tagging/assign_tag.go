package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AssignTagData struct {
	FileID string
	TagID  int64
}

type AssignTag struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAssignTag(
	infra *common.Infra,
	actions *Actions,
) *AssignTag {
	config := actionx.NewConfig(
		actions.Route("assign-tag"),
		false,
	)
	return &AssignTag{
		infra,
		actions,
		config,
	}
}

func (qq *AssignTag) Data(fileID string, tagID int64) *AssignTagData {
	return &AssignTagData{
		FileID: fileID,
		TagID:  tagID,
	}
}

func (qq *AssignTag) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignTagData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	assignment := ctx.TenantCtx().TTx.TagAssignment.Create().
		SetFileID(filex.Data.ID).
		SetTagID(data.TagID).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		// SetIsInherited(false).
		SaveX(ctx)

	tag := assignment.QueryTag().OnlyX(ctx)

	// must be set before writing to rw
	rw.Header().Set("HX-Trigger", event.TagUpdated.String())

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.actions.AssignedTags.EditListItem.ListItem(ctx, data.FileID, tag),
		wx.NewSnackbarf("«%s» assigned.", tag.Name),
	)
}
