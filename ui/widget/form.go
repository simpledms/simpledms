package widget

type Form struct {
	Widget[Form]
	HTMXAttrs

	SubmitLabel         *Text
	IsMultipartFormData bool
	HiddenSubmit        bool

	Children []IWidget

	submitButton *Button
}

func (qq *Form) GetSubmitButton() *Button {
	if qq.SubmitLabel == nil {
		return nil
	}

	if qq.submitButton == nil {
		qq.submitButton = &Button{
			Type:      "submit",
			Label:     qq.SubmitLabel,
			FormID:    qq.GetID(),
			StyleType: ButtonStyleTypeTonal,
		}
	}

	return qq.submitButton
}
