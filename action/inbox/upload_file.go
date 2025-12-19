package inbox

// package action

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

type UploadFileData struct {
	File []byte `schema:"-"`
}

type UploadFile struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[UploadFileData]
	// inboxDirInfo *ent.FileInfo
}

func NewUploadFile(infra *common.Infra, actions *Actions) *UploadFile {
	config := actionx.NewConfig(
		actions.Route("upload-file"),
		false,
	)
	formHelper := autil.NewFormHelper[UploadFileData](
		infra,
		config,
		wx.T("Upload file"),
	)
	formHelper.SetIsMultipartFormData(true)
	return &UploadFile{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
		// inboxDirInfo: infra.Factory().InboxDirInfo(),
	}
}

func (qq *UploadFile) Data() *UploadFileData {
	return &UploadFileData{}
}

func (qq *UploadFile) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	_, err := autil.FormData[UploadFileData](rw, req, ctx)
	if err != nil {
		return err
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
	filename = filepath.Clean(filename)

	filex, err := qq.infra.FileSystem().SaveFile(
		ctx,
		uploadedFile,
		filename,
		true,
		ctx.SpaceCtx().SpaceRootDir().ID,
	)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Retarget", "#innerContent")
	rw.Header().Set("HX-Reswap", "innerHTML")

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.actions.Page.WidgetHandler(rw, req, ctx, filex.PublicID.String()),
		wx.NewSnackbarf("«%s» uploaded.", filex.Name),
	)
}
