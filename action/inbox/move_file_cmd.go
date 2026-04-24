package inbox

import (
	"log"
	"net/http"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type MoveFileCmd struct {
	*acommon.MoveFile
	infra   *common.Infra
	actions *Actions
}

func NewMoveFileCmd(infra *common.Infra, actions *Actions) *MoveFileCmd {
	config := actionx.NewConfig(
		actions.Route("move-file-cmd"),
		false,
	)
	return &MoveFileCmd{
		MoveFile: acommon.NewMoveFile(infra, actions.Common, config),
		infra:    infra,
		actions:  actions,
	}
}

func (qq *MoveFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return e.NewHTTPErrorf(http.StatusMethodNotAllowed, "Only allowed in folder mode.")
	}

	data, err := autil.FormData[acommon.MoveFileFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	fileDTO := repos.Read.FileByPublicIDWithParentX(ctx, data.FileID)

	if !fileDTO.IsInInbox {
		log.Println("file not in inbox")
		return e.NewHTTPErrorf(http.StatusBadRequest, "File must be in inbox.")
	}

	movedFileWithParent, err := qq.infra.FileSystem().MoveByPublicIDs(
		ctx,
		data.CurrentDirID,
		data.FileID,
		data.Filename,
		data.NewDirName,
	)
	if err != nil {
		log.Println(err)
		return err
	}
	if movedFileWithParent.Parent == nil {
		return e.NewHTTPErrorf(http.StatusInternalServerError, "Moved file parent is missing.")
	}

	err = repos.Write.SetFileInInboxByIDX(ctx, movedFileWithParent.ID, false)
	if err != nil {
		log.Println(err)
		return err
	}

	action := &wx.Link{
		Href: route.BrowseFile(
			ctx.TenantCtx().TenantID,
			ctx.SpaceCtx().SpaceID,
			movedFileWithParent.Parent.PublicID,
			movedFileWithParent.PublicID,
		),
		Child: wx.T("Open file"),
	}

	rw.AddRenderables(
		wx.NewSnackbarf("Moved to «%s».", movedFileWithParent.Parent.Name).WithAction(action),
	)

	rw.Header().Set("HX-Trigger", event.FileMoved.String())
	// TODO not nice because logic to reload list and close details is implemented by handling FileMoved event
	// TODO select next file to process instead
	rw.Header().Set("HX-Replace-Url", route.InboxRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID))

	return nil
}
