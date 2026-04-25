package widget

import (
	"github.com/simpledms/simpledms/ctxx"
)

type FormFields struct {
	Widget[FormFields]
	Elements formElements
}

func NewFormFields(ctx ctxx.Context, data any) *FormFields {
	// TODO should be refactored, current implementation was taken over from go-formfields
	//		and is way to complicated for current use case where we can construct wx.TextFields directly.
	return &FormFields{
		Elements: newFormElements(ctx, data),
	}
}
