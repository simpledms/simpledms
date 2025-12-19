package util

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/renderable"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FormHelper[T any] struct {
	infra *common.Infra
	*actionx.Config
	formTitle         *wx.Text
	submitButtonLabel *wx.Text
	// hxTarget            string
	isMultipartFormData bool
}

func NewFormHelper[T any](
	infra *common.Infra,
	config *actionx.Config,
	formTitle *wx.Text,
	// hxTarget string,
) *FormHelper[T] {
	return &FormHelper[T]{
		infra:             infra,
		Config:            config,
		formTitle:         formTitle,
		submitButtonLabel: wx.T("Save"),
		// hxTarget:  hxTarget,
	}
}

func NewFormHelperX[T any](
	infra *common.Infra,
	config *actionx.Config,
	formTitle *wx.Text,
	submitButtonLabel *wx.Text,
	// hxTarget string,
) *FormHelper[T] {
	if submitButtonLabel == nil {
		submitButtonLabel = wx.T("Save")
	}
	return &FormHelper[T]{
		infra:             infra,
		Config:            config,
		formTitle:         formTitle,
		submitButtonLabel: submitButtonLabel,
		// hxTarget:  hxTarget,
	}
}

func (qq *FormHelper[T]) SetIsMultipartFormData(val bool) {
	qq.isMultipartFormData = val
}

func (qq *FormHelper[T]) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	// TODO do prep if necessary... (filter selects, etc.)
	// TODO set default value, for example for current

	data, err := FormDataX[T](rw, req, ctx, true)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")
	hxTarget := req.URL.Query().Get("hx-target")

	qq.infra.Renderer().RenderX(rw, ctx,
		qq.Form(
			ctx,
			data,
			actionx.ResponseWrapper(wrapper),
			qq.submitButtonLabel,
			hxTarget,
		),
	)
	return nil
}

// TODO is this the correct location? it's app specific?
// TODO rename to PopoverLink or FullscreenLink?
// TODO probably deprecated? ModalLinkAttrs would be better choice
func (qq *FormHelper[T]) ModalLink(data *T, child wx.IWidget, hxTargetForm string) *wx.Link {
	return &wx.Link{
		HTMXAttrs: qq.ModalLinkAttrs(data, hxTargetForm),
		Child:     child,
	}
}

// hxTargetForm is deprecated and X-Query should be used instead to load a response/view to render
// TODO not sure about comment above, X-Query target must be set and it maybe via hxTargetForm?
func (qq *FormHelper[T]) ModalLinkAttrs(data *T, hxTargetForm string) wx.HTMXAttrs {
	return wx.HTMXAttrs{
		HxPost: qq.FormEndpointWithParams(actionx.ResponseWrapperDialog, hxTargetForm),
		HxVals: util.JSON(data),
		// LoadInDialog: true,
		LoadInPopover: true,
	}
}

// TODO make formTitle customizable, as param?
func (qq *FormHelper[T]) Form(
	ctx ctxx.Context,
	formData *T,
	wrapper actionx.ResponseWrapper,
	submitButtonLabel *wx.Text,
	hxTarget string,
) renderable.Renderable {
	hxSwap := "outerHTML"
	if hxTarget == "" {
		hxSwap = "none"
	}

	var formSubmitBtn *wx.Text
	if wrapper == actionx.ResponseWrapperNone {
		formSubmitBtn = submitButtonLabel
	}

	form := &wx.Form{
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   hxSwap,
		},
		SubmitLabel:         formSubmitBtn,
		IsMultipartFormData: qq.isMultipartFormData,
		Children: []wx.IWidget{
			wx.NewFormFields(ctx, formData),
		},
	}

	return WrapWidget(qq.formTitle, submitButtonLabel, form, wrapper, wx.DialogLayoutDefault)
}

// MapFormData and not just FormData or Data to prevent naming conflicts and mix-ups with default
// Data method (simulated data constructor)
// TODO why is this necessary? seems like just an alias
func (qq *FormHelper[T]) MapFormData(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) (*T, error) {
	return FormData[T](rw, req, ctx)
}
