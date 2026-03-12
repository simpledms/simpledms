package admin

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type SetGlobalUploadLimitCmdData struct {
	IsUnlimited      bool
	MaxUploadSizeMib int64 `validate:"min=0"`
}

type SetGlobalUploadLimitCmd struct {
	infra              *common.Infra
	actions            *Actions
	uploadLimitService *modelmain.UploadLimitService
	*actionx.Config
}

func NewSetGlobalUploadLimitCmd(infra *common.Infra, actions *Actions) *SetGlobalUploadLimitCmd {
	config := actionx.NewConfig(actions.Route("set-global-upload-limit-cmd"), false)
	return &SetGlobalUploadLimitCmd{
		infra:              infra,
		actions:            actions,
		uploadLimitService: modelmain.NewUploadLimitService(),
		Config:             config,
	}
}

func (qq *SetGlobalUploadLimitCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SetGlobalUploadLimitCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	uploadLimit, err := qq.uploadLimitService.SetGlobalUploadLimit(
		ctx,
		qq.infra.SystemConfig(),
		data.IsUnlimited,
		data.MaxUploadSizeMib,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	uploadLimitLabel := uploadLimit.LabelWithUnlimited(wx.T("unlimited").String(ctx))

	rw.Header().Set("HX-Trigger", event.UploadLimitUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("Global upload limit updated to %s.", uploadLimitLabel))

	return nil
}
