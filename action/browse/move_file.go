package browse

import (
	"log"
	"net/http"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type MoveFile struct {
	*acommon.MoveFile
	infra   *common.Infra
	actions *Actions
}

func NewMoveFile(infra *common.Infra, actions *Actions) *MoveFile {
	config := actionx.NewConfig(
		actions.Route("move-file"),
		false,
	)
	return &MoveFile{
		MoveFile: acommon.NewMoveFile(infra, actions.Common, config),
		infra:    infra,
		actions:  actions,
	}
}

func (qq *MoveFile) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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
	fileWithParent := qq.infra.FileRepo.GetXX(fileWithParentx)

	fileWithParent, err = qq.infra.FileSystem().Move(ctx, destDir, fileWithParent, data.Filename, data.NewDirName)
	if err != nil {
		log.Println(err)
		return err
	}

	// show the appropriate link if either file or directory was moved
	var action *wx.Link
	if fileWithParent.Data.IsDirectory {
		action = &wx.Link{
			Href:  route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, fileWithParent.Data.PublicID.String()),
			Child: wx.T("Open directory"), // TODO Go to, or Open?
		}
	} else {
		parent, err := fileWithParent.Parent(ctx)
		if err != nil {
			log.Println(err)
			return err
		}

		action = &wx.Link{
			Href:  route.BrowseFile(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, parent.Data.PublicID.String(), fileWithParent.Data.PublicID.String()),
			Child: wx.T("Open file"), // TODO Go to, or Open?
		}
	}

	// important that current dir from URL in case move was used from search results
	dirIDStr := req.PathValue("dir_id")
	// dirID64 := int64(0)
	if dirIDStr == "" {
		// load root
		dirIDStr = ctx.SpaceCtx().SpaceRootDir().PublicID.String()
	}

	// TODO update URL: only if file is moved from file context menu

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.actions.ListDir.WidgetHandler(
			rw,
			req,
			ctx,
			dirIDStr,
			"",
		),
		wx.NewSnackbarf("Moved to «%s».", destDir.Data.Name).WithAction(action),
	)
}
