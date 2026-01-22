package browse

import (
	"log"
	"net/http"
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type DeleteFileData struct {
	FileID string `form_attr_type:"hidden"`
	// CurrentPath string `form_attr_type:"hidden"`
}

type DeleteFile struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteFile(
	infra *common.Infra,
	actions *Actions,
) *DeleteFile {
	return &DeleteFile{
		infra,
		actions,
		actionx.NewConfig(
			actions.Route("delete-file-cmd"),
			false,
		),
	}
}

func (qq *DeleteFile) Data(fileID string) *DeleteFileData {
	return &DeleteFileData{
		FileID: fileID,
	}
}

func (qq *DeleteFile) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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
