package admin

import (
	"fmt"
	"log"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	uploadlimitmodel "github.com/simpledms/simpledms/core/model/uploadlimit"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	actionx2 "github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type SetGlobalUploadLimitForm struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
}

func NewSetGlobalUploadLimitForm(infra *common.Infra, actions *Actions) *SetGlobalUploadLimitForm {
	config := actionx2.NewConfig(actions.Route("set-global-upload-limit"), true).SetUsesSeparatedCmd(true)
	return &SetGlobalUploadLimitForm{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SetGlobalUploadLimitForm) Data() *SetGlobalUploadLimitCmdData {
	uploadLimit, err := uploadlimitmodel.NewUploadLimitFromBytes(qq.infra.SystemConfig().MaxUploadSizeBytes())
	if err != nil {
		log.Println(err)
		return &SetGlobalUploadLimitCmdData{
			IsUnlimited:      true,
			MaxUploadSizeMib: 0,
		}
	}

	return &SetGlobalUploadLimitCmdData{
		IsUnlimited:      uploadLimit.IsUnlimited(),
		MaxUploadSizeMib: uploadLimit.MiB(),
	}
}

func (qq *SetGlobalUploadLimitForm) ModalLinkAttrs(data *SetGlobalUploadLimitCmdData, hxTargetForm string) widget.HTMXAttrs {
	return widget.HTMXAttrs{
		HxPost:        qq.FormEndpointWithParams(actionx2.ResponseWrapperDialog, hxTargetForm),
		HxVals:        util.JSON(data),
		LoadInPopover: true,
	}
}

func (qq *SetGlobalUploadLimitForm) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	return qq.FormHandler(rw, req, ctx)
}

func (qq *SetGlobalUploadLimitForm) FormHandler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormDataX[SetGlobalUploadLimitCmdData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	wrapper := actionx2.ResponseWrapper(req.URL.Query().Get("wrapper"))
	hxTarget := req.URL.Query().Get("hx-target")
	hxSwap := "outerHTML"
	if hxTarget == "" {
		hxSwap = "none"
	}

	var nilableFormSubmitLabel *widget.Text
	if wrapper == actionx2.ResponseWrapperNone {
		nilableFormSubmitLabel = widget.T("Save")
	}

	refreshTarget := "closest form"
	if wrapper == actionx2.ResponseWrapperDialog {
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
			HxPost:   qq.actions.SetGlobalUploadLimitCmd.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   hxSwap,
		},
		SubmitLabel: nilableFormSubmitLabel,
		Children: []widget.IWidget{
			&widget.Container{
				GapY: true,
				Child: []widget.IWidget{
					&widget.Checkbox{
						HTMXAttrs: refreshFormAttrs,
						Label:     widget.T("Unlimited"),
						Name:      "IsUnlimited",
						Value:     "true",
						IsChecked: data.IsUnlimited,
					},
					&widget.TextField{
						Widget: widget.Widget[widget.TextField]{
							ID: "globalUploadLimitMaxUploadSizeMib",
						},
						Label:        widget.T("Max upload size (MiB)"),
						Name:         "MaxUploadSizeMib",
						Type:         "number",
						Step:         "1",
						IsDisabled:   data.IsUnlimited,
						DefaultValue: fmt.Sprintf("%d", data.MaxUploadSizeMib),
					},
				},
			},
		},
	}

	qq.infra.Renderer().RenderX(rw, ctx,
		autil.WrapWidget(
			widget.T("Set global upload limit"),
			widget.T("Save"),
			form,
			wrapper,
			widget.DialogLayoutDefault,
		),
	)

	return nil
}
