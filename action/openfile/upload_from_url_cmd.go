package openfile

import (
	"context"
	"io"
	"net/http"
	"strings"

	autil "github.com/simpledms/simpledms/core/action/util"
	temporaryfilemodel "github.com/simpledms/simpledms/core/model/temporaryfile"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type UploadFromURLCmdData struct {
	URL string `form:"url" validate:"required"`
}

type UploadFromURLCmd struct {
	uploadFromURLService *temporaryfilemodel.UploadFromURLService
	*actionx.Config
}

func NewUploadFromURLCmd(actions *Actions, uploadFromURLService *temporaryfilemodel.UploadFromURLService) *UploadFromURLCmd {
	config := actionx.NewConfig(
		actions.Route("upload-from-url-cmd"),
		false,
	).EnableManualTxManagement()

	return &UploadFromURLCmd{
		uploadFromURLService: uploadFromURLService,
		Config:               config,
	}
}

func (qq *UploadFromURLCmd) Data(urlx string) *UploadFromURLCmdData {
	return &UploadFromURLCmdData{
		URL: urlx,
	}
}

func (qq *UploadFromURLCmd) SetDownloadFileForTesting(
	downloadFile func(context.Context, string) (string, io.ReadCloser, error),
) {
	qq.uploadFromURLService.SetDownloadFileForTesting(downloadFile)
}

func (qq *UploadFromURLCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UploadFromURLCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	uploadToken, err := qq.uploadFromURLService.UploadFromURL(ctx, strings.TrimSpace(data.URL))
	if err != nil {
		return err
	}

	rw.AddRenderables(wx.NewSnackbarf("File uploaded, please select a space."))
	if req.Header.Get("HX-Request") != "" {
		rw.Header().Set("HX-Redirect", route.SelectSpace(uploadToken))
		return nil
	}

	http.Redirect(rw, req.Request, route.SelectSpace(uploadToken), http.StatusFound)
	return nil
}
