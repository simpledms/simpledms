package admin

import (
	"fmt"
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	uploadlimitmodel "github.com/simpledms/simpledms/model/main/uploadlimit"
	"github.com/simpledms/simpledms/ui/util"
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
		uploadLimit, err := uploadlimitmodel.NewUploadLimitFromBytes(qq.infra.SystemConfig().MaxUploadSizeBytes())
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

	uploadLimit, err := uploadlimitmodel.NewUploadLimitFromMiB(*nilableMaxUploadSizeMibOverride)
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

func (qq *SetTenantUploadLimitOverrideForm) ModalLinkAttrs(data *SetTenantUploadLimitOverrideCmdData, hxTargetForm string) widget.HTMXAttrs {
	return widget.HTMXAttrs{
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

	var nilableFormSubmitLabel *widget.Text
	if wrapper == actionx.ResponseWrapperNone {
		nilableFormSubmitLabel = widget.T("Save")
	}

	refreshTarget := "closest form"
	if wrapper == actionx.ResponseWrapperDialog {
		refreshTarget = "closest dialog"
	}

	refreshFormAttrs := widget.HTMXAttrs{
		HxPost:    qq.FormEndpointWithParams(wrapper, hxTarget),
		HxTrigger: "change",
		HxInclude: "closest form",
		HxTarget:  refreshTarget,
		HxSwap:    "outerHTML",
	}

	form := &widget.Form{
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.actions.SetTenantUploadLimitOverrideCmd.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   hxSwap,
		},
		SubmitLabel: nilableFormSubmitLabel,
		Children: []widget.IWidget{
			&widget.Container{
				GapY: true,
				Child: []widget.IWidget{
					&widget.TextField{
						Name:         "TenantID",
						Type:         "hidden",
						DefaultValue: data.TenantID,
					},
					&widget.Checkbox{
						HTMXAttrs: refreshFormAttrs,
						Label:     widget.T("Use global default"),
						Name:      "UseGlobalDefault",
						Value:     "true",
						IsChecked: data.UseGlobalDefault,
					},
					&widget.Checkbox{
						HTMXAttrs: refreshFormAttrs,
						Label:     widget.T("Unlimited"),
						Name:      "IsUnlimited",
						Value:     "true",
						IsChecked: data.IsUnlimited,
					},
					&widget.TextField{
						Widget: widget.Widget[widget.TextField]{
							ID: "tenantUploadLimitMaxUploadSizeMib",
						},
						Label:        widget.T("Max upload size (MiB)"),
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
			widget.T("Set tenant upload limit"),
			widget.T("Save"),
			form,
			wrapper,
			widget.DialogLayoutDefault,
		),
	)

	return nil
}
