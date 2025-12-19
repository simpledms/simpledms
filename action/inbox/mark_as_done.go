package inbox

// package action

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type MarkAsDoneData struct {
	FileID string
}

type MarkAsDone struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	// inboxDir   *ent.File
	// storageDir *ent.File
}

func NewMarkAsDone(infra *common.Infra, actions *Actions) *MarkAsDone {
	config := actionx.NewConfig(
		actions.Route("mark-as-done"),
		false,
	)
	return &MarkAsDone{
		infra:   infra,
		actions: actions,
		Config:  config,
		// inboxDir:   inboxDir,
		// storageDir: storageDir,
	}
}

func (qq *MarkAsDone) Data(fileID string) *MarkAsDoneData {
	return &MarkAsDoneData{
		FileID: fileID,
	}
}

func (qq *MarkAsDone) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[MarkAsDoneData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetWithParentX(ctx, data.FileID)

	// assignment := filex.Data.QuerySpaceAssignment().Where(spacefileassignment.SpaceID(ctx.SpaceCtx().Space.ID)).OnlyX(ctx)

	if !filex.Data.IsInInbox {
		log.Println("file not in inbox")
		return e.NewHTTPErrorf(http.StatusBadRequest, "File must be in inbox.")
	}

	filex.Data.Update().SetIsInInbox(false).SaveX(ctx)
	// assignment = assignment.Update().SetIsInInbox(false).SaveX(ctx)

	// FIXME not correct, should not be opened in parent dir, but flat
	action := &wx.Link{
		Href:  route.BrowseFile(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Parent(ctx).Data.PublicID.String(), filex.Data.PublicID.String()),
		Child: wx.T("Open file"),
	}

	rw.AddRenderables(
		wx.NewSnackbarf("Marked file «%s» as done.", filex.Data.Name).WithAction(action),
	)

	return nil

	/*
		state := autil.StateX[PageState](rw, req)
		selectedFileID := int64(0)

		// duplicate in AssignFile and MoveFile
		// select next file in queue
		//
		// OnlyX or OnlyID doesn't work with Limit, returns error if multiple before Limit is applied
		files := qq.actions.ListFiles.filesQuery(tx, state).Limit(1).AllX(ctx)
		if len(files) == 0 {
			selectedFileID = 0
		} else {
			selectedFileID = files[0].ID
		}

		rw.Header().Set("HX-Retarget", "#innerContent")
		rw.Header().Set("HX-Reswap", "innerHTML")


		return qq.infra.Renderer().Render(
			rw,
			qq.actions.Page.WidgetHandler(rw, req, tx, selectedFileID),
			wx.NewSnackbarf("Marked file «%s» as done.", oldFilename).WithAction(action),
		)

	*/
}
