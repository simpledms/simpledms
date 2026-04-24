package browse

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
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

func (qq *DeleteFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeleteFileData](rw, req, ctx)
	if err != nil {
		return err
	}

	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	fileDTO := repos.Read.FileByPublicIDX(ctx, data.FileID)
	fileWithChildren := repos.Read.FileByPublicIDWithChildrenX(ctx, data.FileID)
	// fileInfo := ctx.TenantCtx().TTx.FileInfoPartial.Query().Where(fileinfo.PublicFileID(data.FileID)).OnlyX(ctx)

	// only delete empry dirs, otherwise we have to iterate recursively over all files... also risky for user
	// and harder to undo
	if fileWithChildren.IsDirectory && fileWithChildren.ChildDirectoryCount+fileWithChildren.ChildFileCount > 0 {
		log.Println("Folder not empty")
		return e.NewHTTPErrorf(http.StatusBadRequest, "Folder isn't empty.")
	}

	err = repos.Write.SoftDeleteFileByIDX(ctx, fileDTO.ID, ctx.SpaceCtx().User.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Trigger", event.FileDeleted.String())
	if fileDTO.IsDirectory {
		rw.AddRenderables(wx.NewSnackbarf("Folder deleted."))
	} else {
		rw.AddRenderables(wx.NewSnackbarf("File deleted."))
	}

	return nil
}
