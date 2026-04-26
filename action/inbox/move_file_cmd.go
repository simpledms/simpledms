package inbox

import (
	"log"
	"net/http"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
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

func (qq *MoveFileCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	action := &widget.Link{
		Href: route.BrowseFile(
			ctx.TenantCtx().TenantID,
			ctx.SpaceCtx().SpaceID,
			filex.Parent(ctx).Data.PublicID.String(),
			filex.Data.PublicID.String(),
		),
		Child: widget.T("Open file"),
	}

	rw.AddRenderables(
		widget.NewSnackbarf("Moved to «%s».", destDir.Data.Name).WithAction(action),
	)

	rw.Header().Set("HX-Trigger", event.FileMoved.String())
	// TODO not nice because logic to reload list and close details is implemented by handling FileMoved event
	// TODO select next file to process instead
	rw.Header().Set("HX-Replace-Url", route.InboxRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID))

	return nil
}
