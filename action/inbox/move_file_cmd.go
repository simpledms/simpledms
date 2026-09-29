package inbox

import (
	"log"
	"net/http"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	filingmodel "github.com/simpledms/simpledms/model/tenant/filing"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
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

	fileData, err := filingmodel.NewFilingService(qq.infra.FileSystem()).FileInboxDocument(
		ctx, data.FileID, data.CurrentDirID, data.Filename, data.NewDirName,
	)
	if err != nil {
		log.Println(err)
		return err
	}
	filex := qq.infra.FileRepo.GetWithParentX(ctx, fileData.PublicID.String())
	destDir := qq.infra.FileRepo.GetX(ctx, data.CurrentDirID)

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
