package admin

import (
	"fmt"
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type SetTenantUploadLimitOverrideForm struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSetTenantUploadLimitOverrideForm(infra *common.Infra, actions *Actions) *SetTenantUploadLimitOverrideForm {
	config := actionx.NewConfig(actions.Route("set-tenant-upload-limit-override"), true).SetUsesSeparatedCmd(true)
	return &SetTenantUploadLimitOverrideForm{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SetTenantUploadLimitOverrideForm) Data(
	tenantID string,
	nilableMaxUploadSizeMibOverride *int64,
) *SetTenantUploadLimitOverrideCmdData {
	data := &SetTenantUploadLimitOverrideCmdData{
		TenantID:         tenantID,
		UseGlobalDefault: nilableMaxUploadSizeMibOverride == nil,
	}

	if nilableMaxUploadSizeMibOverride == nil {
		uploadLimit, err := modelmain.NewUploadLimitFromBytes(qq.infra.SystemConfig().MaxUploadSizeBytes())
		if err != nil {
			log.Println(err)
			data.IsUnlimited = true
			data.MaxUploadSizeMib = 0
			return data
		}

		data.IsUnlimited = uploadLimit.IsUnlimited()
		data.MaxUploadSizeMib = uploadLimit.MiB()
		return data
	}

	uploadLimit, err := modelmain.NewUploadLimitFromMiB(*nilableMaxUploadSizeMibOverride)
	if err != nil {
		log.Println(err)
		data.IsUnlimited = true
		data.MaxUploadSizeMib = 0
		return data
	}

	data.IsUnlimited = uploadLimit.IsUnlimited()
	data.MaxUploadSizeMib = uploadLimit.MiB()
	return data
}

func (qq *SetTenantUploadLimitOverrideForm) ModalLinkAttrs(data *SetTenantUploadLimitOverrideCmdData, hxTargetForm string) wx.HTMXAttrs {
	return wx.HTMXAttrs{
		HxPost:        qq.FormEndpointWithParams(actionx.ResponseWrapperDialog, hxTargetForm),
		HxVals:        util.JSON(data),
		LoadInPopover: true,
	}
}

func (qq *SetTenantUploadLimitOverrideForm) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	return qq.FormHandler(rw, req, ctx)
}

func (qq *SetTenantUploadLimitOverrideForm) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormDataX[SetTenantUploadLimitOverrideCmdData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	wrapper := actionx.ResponseWrapper(req.URL.Query().Get("wrapper"))
	hxTarget := req.URL.Query().Get("hx-target")
	hxSwap := "outerHTML"
	if hxTarget == "" {
		hxSwap = "none"
	}

	var nilableFormSubmitLabel *wx.Text
	if wrapper == actionx.ResponseWrapperNone {
		nilableFormSubmitLabel = wx.T("Save")
	}

	refreshTarget := "closest form"
	if wrapper == actionx.ResponseWrapperDialog {
		refreshTarget = "closest dialog"
	}

	refreshFormAttrs := wx.HTMXAttrs{
		HxPost:    qq.FormEndpointWithParams(wrapper, hxTarget),
		HxTrigger: "change",
		HxInclude: "closest form",
		HxTarget:  refreshTarget,
		HxSwap:    "outerHTML",
	}

	form := &wx.Form{
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.actions.SetTenantUploadLimitOverrideCmd.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   hxSwap,
		},
		SubmitLabel: nilableFormSubmitLabel,
		Children: []wx.IWidget{
			&wx.Container{
				GapY: true,
				Child: []wx.IWidget{
					&wx.TextField{
						Name:         "TenantID",
						Type:         "hidden",
						DefaultValue: data.TenantID,
					},
					&wx.Checkbox{
						HTMXAttrs: refreshFormAttrs,
						Label:     wx.T("Use global default"),
						Name:      "UseGlobalDefault",
						Value:     "true",
						IsChecked: data.UseGlobalDefault,
					},
					&wx.Checkbox{
						HTMXAttrs: refreshFormAttrs,
						Label:     wx.T("Unlimited"),
						Name:      "IsUnlimited",
						Value:     "true",
						IsChecked: data.IsUnlimited,
					},
					&wx.TextField{
						Widget: wx.Widget[wx.TextField]{
							ID: "tenantUploadLimitMaxUploadSizeMib",
						},
						Label:        wx.T("Max upload size (MiB)"),
						Name:         "MaxUploadSizeMib",
						Type:         "number",
						Step:         "1",
						IsDisabled:   data.UseGlobalDefault || data.IsUnlimited,
						DefaultValue: fmt.Sprintf("%d", data.MaxUploadSizeMib),
					},
				},
			},
		},
	}

	qq.infra.Renderer().RenderX(rw, ctx,
		autil.WrapWidget(
			wx.T("Set tenant upload limit"),
			wx.T("Save"),
			form,
			wrapper,
			wx.DialogLayoutDefault,
		),
	)

	return nil
}
