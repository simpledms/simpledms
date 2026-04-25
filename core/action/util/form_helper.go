package util

import (
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/renderable"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	actionx2 "github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type FormHelper[T any] struct {
	infra *common.Infra
	*actionx2.Config
	formTitle         *widget.Text
	submitButtonLabel *widget.Text
	// hxTarget            string
	isMultipartFormData bool
}

func NewFormHelper[T any](
	infra *common.Infra,
	config *actionx2.Config,
	formTitle *widget.Text,
	// hxTarget string,
) *FormHelper[T] {
	return &FormHelper[T]{
		infra:             infra,
		Config:            config,
		formTitle:         formTitle,
		submitButtonLabel: widget.T("Save"),
		// hxTarget:  hxTarget,
	}
}

func NewFormHelperX[T any](
	infra *common.Infra,
	config *actionx2.Config,
	formTitle *widget.Text,
	submitButtonLabel *widget.Text,
	// hxTarget string,
) *FormHelper[T] {
	if submitButtonLabel == nil {
		submitButtonLabel = widget.T("Save")
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

// usually you don't want to overwrite this and create a separate form action instead;
// this is mainly a helper for forms that can be rendered from the XData or XFormData struct
func (qq *FormHelper[T]) FormHandler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
			actionx2.ResponseWrapper(wrapper),
			qq.submitButtonLabel,
			hxTarget,
		),
	)
	return nil
}

// TODO is this the correct location? it's app specific?
// TODO rename to PopoverLink or FullscreenLink?
// TODO probably deprecated? ModalLinkAttrs would be better choice
func (qq *FormHelper[T]) ModalLink(data *T, child widget.IWidget, hxTargetForm string) *widget.Link {
	return &widget.Link{
		HTMXAttrs: qq.ModalLinkAttrs(data, hxTargetForm),
		Child:     child,
	}
}

// hxTargetForm is deprecated and X-Query should be used instead to load a response/view to render
// TODO not sure about comment above, X-Query target must be set and it maybe via hxTargetForm?
func (qq *FormHelper[T]) ModalLinkAttrs(data *T, hxTargetForm string) widget.HTMXAttrs {
	return widget.HTMXAttrs{
		HxPost: qq.FormEndpointWithParams(actionx2.ResponseWrapperDialog, hxTargetForm),
		HxVals: util.JSON(data),
		// LoadInDialog: true,
		LoadInPopover: true,
	}
}

// TODO make formTitle customizable, as param?
func (qq *FormHelper[T]) Form(
	ctx ctxx.Context,
	formData *T,
	wrapper actionx2.ResponseWrapper,
	submitButtonLabel *widget.Text,
	hxTarget string,
) renderable.Renderable {
	hxSwap := "outerHTML"
	if hxTarget == "" {
		hxSwap = "none"
	}

	var formSubmitBtn *widget.Text
	if wrapper == actionx2.ResponseWrapperNone {
		formSubmitBtn = submitButtonLabel
	}

	form := &widget.Form{
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   hxSwap,
		},
		SubmitLabel:         formSubmitBtn,
		IsMultipartFormData: qq.isMultipartFormData,
		Children: []widget.IWidget{
			widget.NewFormFields(ctx, formData),
		},
	}

	return WrapWidget(qq.formTitle, submitButtonLabel, form, wrapper, widget.DialogLayoutDefault)
}

// MapFormData and not just FormData or Data to prevent naming conflicts and mix-ups with default
// Data method (simulated data constructor)
// TODO why is this necessary? seems like just an alias
func (qq *FormHelper[T]) MapFormData(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) (*T, error) {
	return FormData[T](rw, req, ctx)
}
