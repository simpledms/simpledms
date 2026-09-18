package openfile

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	temporaryfilemodel "github.com/simpledms/simpledms/model/main/temporaryfile"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type UploadFromURLCmdData struct {
	URL          string `form:"url" validate:"required"`
	Source       string `form:"source"`
	PermissionID string `form:"permission_id"`
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

func (qq *UploadFromURLCmd) SetDownloadFileForTesting(
	downloadFile func(context.Context, string) (string, io.ReadCloser, error),
) {
	qq.uploadFromURLService.SetDownloadFileForTesting(downloadFile)
}

func (qq *UploadFromURLCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UploadFromURLCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	uploadToken, err := qq.uploadFromURLService.UploadFromURL(
		ctx,
		strings.TrimSpace(data.URL),
		strings.TrimSpace(data.Source),
	)
	if err != nil {
		if strings.TrimSpace(data.Source) == temporaryfilemodel.OpenCloudURLSource {
			log.Printf(
				"[OpenCloud import] download failed permission_id=%q error_type=%T",
				strings.TrimSpace(data.PermissionID),
				err,
			)
		}
		return err
	}

	rw.AddRenderables(wx.NewSnackbarf("File uploaded, please select a space."))
	if req.Header.Get("HX-Request") != "" {
		location, err := json.Marshal(map[string]string{
			"path":   route.SelectSpace(uploadToken),
			"target": "#innerContent",
			"select": "#innerContent",
			"swap":   "outerHTML",
		})
		if err != nil {
			log.Println(err)
			return err
		}
		rw.Header().Set("HX-Location", string(location))
		return nil
	}

	http.Redirect(rw, req.Request, route.SelectSpace(uploadToken), http.StatusFound)
	return nil
}
