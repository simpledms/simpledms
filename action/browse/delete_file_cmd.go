package browse

import (
	"log"
	"net/http"
	"time"

	"github.com/marcobeierer/go-core/db/entx"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	wx "github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type DeleteFileData struct {
	FileID string `form_attr_type:"hidden"`
	// CurrentPath string `form_attr_type:"hidden"`
}

type DeleteFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteFileCmd(
	infra *common.Infra,
	actions *Actions,
) *DeleteFileCmd {
	return &DeleteFileCmd{
		infra,
		actions,
		actionx.NewConfig(
			actions.Route("delete-file-cmd"),
			false,
		),
	}
}

func (qq *DeleteFileCmd) Data(fileID string) *DeleteFileData {
	return &DeleteFileData{
		FileID: fileID,
	}
}

func (qq *DeleteFileCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeleteFileData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := ctx.SpaceCtx().Space.QueryFiles().WithChildren().Where(file.PublicID(entx.NewCIText(data.FileID))).OnlyX(ctx)
	// fileInfo := ctx.TenantCtx().TTx.FileInfoPartial.Query().Where(fileinfo.PublicFileID(data.FileID)).OnlyX(ctx)

	// only delete empry dirs, otherwise we have to iterate recursively over all files... also risky for user
	// and harder to undo
	if filex.IsDirectory && len(filex.Edges.Children) > 0 {
		log.Println("Folder not empty")
		return e.NewHTTPErrorf(http.StatusBadRequest, "Folder isn't empty.")
	}

	filex = filex.Update().
		SetDeletedAt(time.Now()).
		SetDeleter(ctx.SpaceCtx().User).
		SaveX(ctx)

	rw.Header().Set("HX-Trigger", event.FileDeleted.String())
	if filex.IsDirectory {
		rw.AddRenderables(wx.NewSnackbarf("Folder deleted."))
	} else {
		rw.AddRenderables(wx.NewSnackbarf("File deleted."))
	}

	return nil
}
