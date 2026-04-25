package admin

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	uploadlimitmodel "github.com/simpledms/simpledms/core/model/uploadlimit"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type SetTenantUploadLimitOverrideCmdData struct {
	TenantID         string `form_attr_type:"hidden"`
	UseGlobalDefault bool
	IsUnlimited      bool
	MaxUploadSizeMib int64 `validate:"min=0"`
}

type SetTenantUploadLimitOverrideCmd struct {
	infra              *common.Infra
	actions            *Actions
	uploadLimitService *uploadlimitmodel.UploadLimitService
	*actionx.Config
}

func NewSetTenantUploadLimitOverrideCmd(infra *common.Infra, actions *Actions) *SetTenantUploadLimitOverrideCmd {
	config := actionx.NewConfig(actions.Route("set-tenant-upload-limit-override-cmd"), false)
	return &SetTenantUploadLimitOverrideCmd{
		infra:              infra,
		actions:            actions,
		uploadLimitService: uploadlimitmodel.NewUploadLimitService(),
		Config:             config,
	}
}

func (qq *SetTenantUploadLimitOverrideCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SetTenantUploadLimitOverrideCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	nilableUploadLimitOverride, err := qq.uploadLimitService.SetTenantUploadLimitOverride(
		ctx,
		data.TenantID,
		data.UseGlobalDefault,
		data.IsUnlimited,
		data.MaxUploadSizeMib,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	uploadLimitLabel := widget.T("global default").String(ctx)
	if nilableUploadLimitOverride != nil {
		uploadLimitLabel = nilableUploadLimitOverride.LabelWithUnlimited(widget.T("unlimited").String(ctx))
	}

	rw.Header().Set("HX-Trigger", events.UploadLimitUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("Tenant upload limit updated to %s.", uploadLimitLabel))

	return nil
}
