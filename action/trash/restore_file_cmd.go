package trash

import (
	"net/http"

	"github.com/marcobeierer/go-core/db/entx"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type RestoreFileCmdData struct {
	FileID string
}

type RestoreFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewRestoreFileCmd(infra *common.Infra, actions *Actions) *RestoreFileCmd {
	return &RestoreFileCmd{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("restore-file-cmd"),
			false,
		),
	}
}

func (qq *RestoreFileCmd) Data(fileID string) *RestoreFileCmdData {
	return &RestoreFileCmdData{
		FileID: fileID,
	}
}

func (qq *RestoreFileCmd) DataWithOptions(fileID string) *RestoreFileCmdData {
	return &RestoreFileCmdData{
		FileID: fileID,
	}
}

func (qq *RestoreFileCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[RestoreFileCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	filex := ctx.SpaceCtx().TTx.File.Query().
		Where(
			file.PublicID(entx.NewCIText(data.FileID)),
			file.SpaceID(ctx.SpaceCtx().Space.ID), // not necessary because implicit, just for safety
		).
		OnlyX(ctxWithDeleted)

	if filex.IsDirectory {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Folders cannot be restored.")
	}
	if filex.DeletedAt.IsZero() {
		return e.NewHTTPErrorf(http.StatusBadRequest, "File is not deleted.")
	}

	parentExists := false
	if filex.ParentID != 0 {
		parentExists = ctx.AppCtx().TTx.File.Query().
			Where(
				file.ID(filex.ParentID),
				file.SpaceID(ctx.SpaceCtx().Space.ID),
			).
			ExistX(ctx)
	}

	update := filex.Update().
		ClearDeletedAt().
		ClearDeletedBy()

	if !parentExists {
		update = update.
			SetIsInInbox(true).
			SetParentID(ctx.SpaceCtx().SpaceRootDir().ID)
	}

	filex = update.SaveX(ctx)

	if !parentExists {
		rw.AddRenderables(widget.NewSnackbarf("The original parent folder is missing. Restored to Inbox."))
	} else {
		rw.AddRenderables(widget.NewSnackbarf("File restored."))
	}

	rw.Header().Set("HX-Retarget", "#details")
	rw.Header().Set("HX-Reswap", "innerHTML")
	// TODO not nice because logic to reload list and close details is implemented by handling FileRestored event
	rw.Header().Set("HX-Replace-Url", route.TrashRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID))
	rw.Header().Set("HX-Trigger", event.FileRestored.String())

	return qq.infra.Renderer().Render(rw, ctx, &widget.View{})
}
