package tagging

import (
	autil "github.com/marcobeierer/go-core/action/util"
	wx "github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/ui/uix/event"
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

func (qq *DeleteTagCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeleteTagCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	// TODO use soft delete instead? must solve unique index then, maybe add a uuid?
	//		would make sense to keep tagAssignments in this case

	// TODO cleanup from time to time all tags that get deleted and files no longer exist
	//		or just after a long time

	tagName, err := taggingmodel.NewTagService().Delete(ctx, data.TagID)
	if err != nil {
		return err
	}

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
