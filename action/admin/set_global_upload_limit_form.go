package admin

import (
	"fmt"
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type SetGlobalUploadLimitForm struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSetGlobalUploadLimitForm(infra *common.Infra, actions *Actions) *SetGlobalUploadLimitForm {
	config := actionx.NewConfig(actions.Route("set-global-upload-limit"), true).SetUsesSeparatedCmd(true)
	return &SetGlobalUploadLimitForm{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SetGlobalUploadLimitForm) Data() *SetGlobalUploadLimitCmdData {
	uploadLimit, err := modelmain.NewUploadLimitFromBytes(qq.infra.SystemConfig().MaxUploadSizeBytes())
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

func (qq *SetGlobalUploadLimitForm) ModalLinkAttrs(data *SetGlobalUploadLimitCmdData, hxTargetForm string) wx.HTMXAttrs {
	return wx.HTMXAttrs{
		HxPost:        qq.FormEndpointWithParams(actionx.ResponseWrapperDialog, hxTargetForm),
		HxVals:        util.JSON(data),
		LoadInPopover: true,
	}
}

func (qq *SetGlobalUploadLimitForm) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	return qq.FormHandler(rw, req, ctx)
}

func (qq *SetGlobalUploadLimitForm) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormDataX[SetGlobalUploadLimitCmdData](rw, req, ctx, true)
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
			HxPost:   qq.actions.SetGlobalUploadLimitCmd.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   hxSwap,
		},
		SubmitLabel: nilableFormSubmitLabel,
		Children: []wx.IWidget{
			&wx.Container{
				GapY: true,
				Child: []wx.IWidget{
					&wx.Checkbox{
						HTMXAttrs: refreshFormAttrs,
						Label:     wx.T("Unlimited"),
						Name:      "IsUnlimited",
						Value:     "true",
						IsChecked: data.IsUnlimited,
					},
					&wx.TextField{
						Widget: wx.Widget[wx.TextField]{
							ID: "globalUploadLimitMaxUploadSizeMib",
						},
						Label:        wx.T("Max upload size (MiB)"),
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
			wx.T("Set global upload limit"),
			wx.T("Save"),
			form,
			wrapper,
			wx.DialogLayoutDefault,
		),
	)

	return nil
}
