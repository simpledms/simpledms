package browse

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

	// show the appropriate link if either file or directory was moved
	var action *wx.Link
	if movedFileWithParent.IsDirectory {
		action = &wx.Link{
			Href:  route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, movedFileWithParent.PublicID),
			Child: wx.T("Open directory"), // TODO Go to, or Open?
		}
	} else {
		action = &wx.Link{
			Href: route.BrowseFile(
				ctx.TenantCtx().TenantID,
				ctx.SpaceCtx().SpaceID,
				movedFileWithParent.Parent.PublicID,
				movedFileWithParent.PublicID,
			),
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
		qq.actions.ListDirPartial.WidgetHandler(
			rw,
			req,
			ctx,
			dirIDStr,
			"",
		),
		wx.NewSnackbarf("Moved to «%s».", movedFileWithParent.Parent.Name).WithAction(action),
	)
}
