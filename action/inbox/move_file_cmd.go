package inbox

import (
	"log"
	"net/http"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
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

	destDir := qq.infra.FileRepo.GetX(ctx, data.CurrentDirID)
	filex := qq.infra.FileRepo.GetWithParentX(ctx, data.FileID)

	if !filex.Data.IsInInbox {
		log.Println("file not in inbox")
		return e.NewHTTPErrorf(http.StatusBadRequest, "File must be in inbox.")
	}

	// TODO is this okay? probably not, but works as long as nilablePArent on File is used in FileWithParent // FIXME
	filex.File, err = qq.infra.FileSystem().Move(ctx, destDir, filex.File, data.Filename, data.NewDirName)
	if err != nil {
		log.Println(err)
		return err
	}

	filex.Data.Update().SetIsInInbox(false).SaveX(ctx)

	action := &wx.Link{
		Href:  route.BrowseFile(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Parent(ctx).Data.PublicID.String(), filex.Data.PublicID.String()),
		Child: wx.T("Open file"),
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		wx.NewSnackbarf("Moved to «%s».", destDir.Data.Name).WithAction(action),
	)
}
