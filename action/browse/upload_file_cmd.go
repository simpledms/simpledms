package browse

import (
	"log"
	"net/http"
	"path/filepath"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type UploadFileCmdData struct {
	ParentDirID string `form_attr_type:"hidden"`
	File        []byte `schema:"-"`
	// for renaming
	// TODO preset to uploaded file name
	// TODO option to quickly rename according to pattern defined for folder
	Filename   string // TODO only if in FolderMode
	AddToInbox bool   // TODO only in non-folder mode
}

type UploadFileCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[UploadFileCmdData]
}

func NewUploadFileCmd(
	infra *common.Infra,
	actions *Actions,
) *UploadFileCmd {
	config := actionx.NewConfig(
		actions.Route("upload-file-cmd"),
		false,
	)

	formHelper := autil.NewFormHelper[UploadFileCmdData](
		infra,
		config,
		wx.T("Upload file"),
		// "#fileList",
	)
	formHelper.SetIsMultipartFormData(true)

	return &UploadFileCmd{
		infra,
		actions,
		config,
		formHelper,
	}
}

func (qq *UploadFileCmd) Data(parentDirID string, filename string, addToInbox bool) *UploadFileCmdData {
	return &UploadFileCmdData{
		ParentDirID: parentDirID,
		File:        []byte(""),
		Filename:    filename,
		AddToInbox:  addToInbox,
	}
}

func (qq *UploadFileCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UploadFileCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if data.ParentDirID == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "No parent dir provided.")
	}

	uploadedFile, uploadedFileHeader, err := req.FormFile("File")
	if err != nil {
		// TODO also triggers if no file provided
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusInternalServerError, "could not read file")
	}
	defer func() {
		if err := uploadedFile.Close(); err != nil {
			log.Println(err)
		}
	}()

	filename := uploadedFileHeader.Filename
	if data.Filename != "" {
		filename = data.Filename
	}
	filename = filepath.Clean(filename)

	// parentDirInfo := ctx.TenantCtx().TTx.FileInfoPartial.Query().Where(fileinfo.PublicFileID(data.ParentDirID)).OnlyX(ctx)
	parentDir := qq.infra.FileRepo.GetX(ctx, data.ParentDirID)

	filex, err := qq.infra.FileSystem().SaveFile(
		ctx,
		uploadedFile,
		filename,
		data.AddToInbox,
		parentDir.Data.ID,
	)
	if err != nil {
		return err
	}

	// TODO trigger event (wouldn't work with uppy because not an HTMX request...)

	rw.AddRenderables(wx.NewSnackbarf("«%s» uploaded.", filex.Name))

	// TODO render snackbar
	/*
		qq.infra.Renderer().RenderX(rw, ctx,
			// TODO get rid of this
			qq.actions.ListDirPartial.WidgetHandler(
				rw,
				req,
				ctx,
				parentDir.Data.PublicID.String(),
				"",
			),
			wx.NewSnackbarf("«%s» uploaded.", filex.Name),
		)

	*/
	return nil
}
