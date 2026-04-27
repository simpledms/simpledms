package widget

// TODO refactor to use standard widgets
type PasskeyRegisterDialogContent struct {
	Widget[PasskeyRegisterDialogContent]

	DialogID string
}

func (qq *PasskeyRegisterDialogContent) NameFieldName() string {
	return "passkey_name"
}

func (qq *PasskeyRegisterDialogContent) FormID() string {
	return qq.GetID() + "-form"
}

func (qq *PasskeyRegisterDialogContent) NameFieldID() string {
	return qq.GetID() + "-name"
}

func (qq *PasskeyRegisterDialogContent) ErrorID() string {
	return qq.GetID() + "-error"
}

func (qq *PasskeyRegisterDialogContent) Form() *Form {
	return &Form{
		Widget: Widget[Form]{
			ID: qq.FormID(),
		},
		Children: []IWidget{
			qq.DescriptionParagraph(),
			qq.NameField(),
			qq.PasswordDisabledNoticeParagraph(),
			qq.RecoveryCodesNoticeParagraph(),
			qq.ErrorParagraph(),
		},
	}
}

func (qq *PasskeyRegisterDialogContent) Description() *Text {
	return T("Give this passkey an optional name so you can recognize it later.")
}

func (qq *PasskeyRegisterDialogContent) RecoveryCodesNotice() *Text {
	return T("After registration, printable backup codes will be shown once. Save them before closing.")
}

func (qq *PasskeyRegisterDialogContent) PasswordDisabledNotice() *Text {
	return T("After setup, password sign-in is disabled for this account. Use passkeys and backup codes instead.")
}

func (qq *PasskeyRegisterDialogContent) NameLabel() *Text {
	return T("Passkey name (optional)")
}

func (qq *PasskeyRegisterDialogContent) DescriptionParagraph() *Paragraph {
	return &Paragraph{
		Text:  qq.Description(),
		Class: "body-medium text-on-surface-variant",
	}
}

func (qq *PasskeyRegisterDialogContent) PasswordDisabledNoticeParagraph() *Paragraph {
	return &Paragraph{
		Text:  qq.PasswordDisabledNotice(),
		Class: "body-small text-error",
	}
}

func (qq *PasskeyRegisterDialogContent) RecoveryCodesNoticeParagraph() *Paragraph {
	return &Paragraph{
		Text:  qq.RecoveryCodesNotice(),
		Class: "body-small text-on-surface-variant",
	}
}

func (qq *PasskeyRegisterDialogContent) ErrorParagraph() *Paragraph {
	return &Paragraph{
		Widget: Widget[Paragraph]{
			ID: qq.ErrorID(),
		},
		Text:  Tu(""),
		Class: "body-small text-error min-h-5",
	}
}

func (qq *PasskeyRegisterDialogContent) SubmitLabel() *Text {
	return T("Register passkey")
}

func (qq *PasskeyRegisterDialogContent) NameField() *TextField {
	return &TextField{
		Widget: Widget[TextField]{
			ID: qq.NameFieldID(),
		},
		Label:        qq.NameLabel(),
		Name:         qq.NameFieldName(),
		Type:         "text",
		HasAutofocus: true,
	}
}

func (qq *PasskeyRegisterDialogContent) SubmitButton() *Button {
	return &Button{
		Type:      "submit",
		Label:     qq.SubmitLabel(),
		StyleType: ButtonStyleTypeTonal,
	}
}
