package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DeleteTagCmdData struct {
	TagID int64
	// FileID int64 // optional
}

type DeleteTagCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteTagCmd(infra *common.Infra, actions *Actions) *DeleteTagCmd {
	config := actionx.NewConfig(
		actions.Route("delete-tag-cmd"),
		false,
	)
	return &DeleteTagCmd{
		infra, actions, config,
	}
}

// TODO makes no sense in this order because fileID is optional, but more
//
//	consistent
func (qq *DeleteTagCmd) Data(tagID int64) *DeleteTagCmdData {
	return &DeleteTagCmdData{
		TagID: tagID,
		// FileID: fileID,
	}
}

func (qq *DeleteTagCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeleteTagCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	// TODO use soft delete instead? must solve unique index then, maybe add a uuid?
	//		would make sense to keep tagAssignments in this case

	// TODO cleanup from time to time all tags that get deleted and files no longer exist
	//		or just after a long time

	tagName := ctx.TenantCtx().TTx.Tag.GetX(ctx, data.TagID).Name

	// TODO unassign tag from all files

	// FIXME permissions; must belong to space
	ctx.TenantCtx().TTx.
		Tag.
		DeleteOneID(data.TagID).
		ExecX(ctx)

	/*
		tagx := qq.infra.Client().
			Tag.
			Query().
			Where(tag.ID(data.TagID)).
			WithFiles().
			OnlyX(ctx)


		hardDelete := false

		// hardDelete if not assigned or just assigned to file in which
		// context deletion was triggered
		assignmentCount := len(tagx.Edges.Files)
		if assignmentCount == 0 {
			hardDelete = true
		} else if assignmentCount == 1 && tagx.Edges.Files[0].ID == data.FileID {
			hardDelete = true
		}

		if hardDelete {
			qq.infra.Client().
				Tag.
				DeleteOneID(data.TagID).
				ExecX(ctx)
		} else {
			// TODO mark as deleted
			// TODO hide via hook?

			// add UUID or change Index
		}

		if len(tagx.Edges.TagAssignment)

		qq.infra.Client().
			TagAssignment.
			Delete().
			Where(tagassignment.TagID(data.TagID))

	*/

	// not necessarily, can be tested, but is not worth it
	rw.Header().Set("HX-Trigger", event.TagDeleted.String())
	rw.Header().Set("HX-Reswap", "none")
	rw.AddRenderables(wx.NewSnackbarf("«%s» deleted.", tagName))

	return nil
}
