package browse

import (
	"log"
	"net/http"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
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

	// destDir := ctx.TenantCtx().TTx.File.GetX(ctx, data.CurrentDirID)
	destDir := qq.infra.FileRepo.GetX(ctx, data.CurrentDirID)
	fileWithParentx := ctx.TenantCtx().TTx.File.Query().WithParent().Where(file.PublicID(entx.NewCIText(data.FileID))).OnlyX(ctx)
	fileWithParent := filemodel.NewFile(fileWithParentx)

	fileWithParent, err = qq.infra.FileSystem().Move(ctx, destDir, fileWithParent, data.Filename, data.NewDirName)
	if err != nil {
		log.Println(err)
		return err
	}

	// show the appropriate link if either file or directory was moved
	var action *widget.Link
	if fileWithParent.Data.IsDirectory {
		action = &widget.Link{
			Href:  route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, fileWithParent.Data.PublicID.String()),
			Child: widget.T("Open directory"), // TODO Go to, or Open?
		}
	} else {
		parent, err := fileWithParent.Parent(ctx)
		if err != nil {
			log.Println(err)
			return err
		}

		action = &widget.Link{
			Href:  route.BrowseFile(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, parent.Data.PublicID.String(), fileWithParent.Data.PublicID.String()),
			Child: widget.T("Open file"), // TODO Go to, or Open?
		}
	}

	rw.Header().Set("HX-Trigger", event.FileMoved.String())
	rw.AddRenderables(widget.NewSnackbarf("Moved to «%s».", destDir.Data.Name).WithAction(action))
	return nil
}
