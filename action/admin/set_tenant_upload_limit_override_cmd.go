package admin

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	uploadlimitmodel "github.com/simpledms/simpledms/model/main/uploadlimit"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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

func (qq *SetTenantUploadLimitOverrideCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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

	uploadLimitLabel := wx.T("global default").String(ctx)
	if nilableUploadLimitOverride != nil {
		uploadLimitLabel = nilableUploadLimitOverride.LabelWithUnlimited(wx.T("unlimited").String(ctx))
	}

	rw.Header().Set("HX-Trigger", event.UploadLimitUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("Tenant upload limit updated to %s.", uploadLimitLabel))

	return nil
}
