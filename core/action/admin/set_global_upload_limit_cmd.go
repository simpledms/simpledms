package admin

import (
	"log"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	uploadlimitmodel "github.com/simpledms/simpledms/core/model/uploadlimit"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type SetGlobalUploadLimitCmdData struct {
	IsUnlimited      bool
	MaxUploadSizeMib int64 `validate:"min=0"`
}

type SetGlobalUploadLimitCmd struct {
	infra              *common.Infra
	actions            *Actions
	uploadLimitService *uploadlimitmodel.UploadLimitService
	*actionx.Config
}

func NewSetGlobalUploadLimitCmd(infra *common.Infra, actions *Actions) *SetGlobalUploadLimitCmd {
	config := actionx.NewConfig(actions.Route("set-global-upload-limit-cmd"), false)
	return &SetGlobalUploadLimitCmd{
		infra:              infra,
		actions:            actions,
		uploadLimitService: uploadlimitmodel.NewUploadLimitService(),
		Config:             config,
	}
}

func (qq *SetGlobalUploadLimitCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	uploadLimitLabel := uploadLimit.LabelWithUnlimited(widget.T("unlimited").String(ctx))

	rw.Header().Set("HX-Trigger", events.UploadLimitUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("Global upload limit updated to %s.", uploadLimitLabel))

	return nil
}
