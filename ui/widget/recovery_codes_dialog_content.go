package widget

// TODO refactor to use standard widgets
type RecoveryCodesDialogContent struct {
	Widget[RecoveryCodesDialogContent]

	CodesText   string
	AccountName string
}

func (qq *RecoveryCodesDialogContent) ImportantNotice() *Text {
	return T("Important: these backup codes are shown only once. Save, print, or download them now before closing this dialog.")
}

func (qq *RecoveryCodesDialogContent) RiskNotice() *Text {
	return T("If you lose these codes and your passkey, account recovery may no longer be possible.")
}

func (qq *RecoveryCodesDialogContent) BackupCodesTitle() *Text {
	return T("SimpleDMS backup codes")
}

func (qq *RecoveryCodesDialogContent) AccountLabel() *Text {
	return T("Account")
}

func (qq *RecoveryCodesDialogContent) GeneratedLabel() *Text {
	return T("Generated")
}

func (qq *RecoveryCodesDialogContent) KeepSecureNotice() *Text {
	return T("Keep these backup codes in a secure place.")
}

func (qq *RecoveryCodesDialogContent) ShownOnceNotice() *Text {
	return T("These codes are shown only once.")
}

func (qq *RecoveryCodesDialogContent) CodesHeading() *Text {
	return T("Codes")
}

func (qq *RecoveryCodesDialogContent) PrintOpenedMessage() *Text {
	return T("Print dialog opened.")
}

func (qq *RecoveryCodesDialogContent) CopiedMessage() *Text {
	return T("The backup codes were copied to clipboard.")
}

func (qq *RecoveryCodesDialogContent) CopyFailedMessage() *Text {
	return T("Could not copy backup codes automatically.")
}

func (qq *RecoveryCodesDialogContent) DownloadedMessage() *Text {
	return T("The backup codes were downloaded.")
}

func (qq *RecoveryCodesDialogContent) ImportantNoticeParagraph() *Paragraph {
	return &Paragraph{
		Text:  qq.ImportantNotice(),
		Class: "body-medium text-on-surface-variant",
	}
}

func (qq *RecoveryCodesDialogContent) RiskNoticeParagraph() *Paragraph {
	return &Paragraph{
		Text:  qq.RiskNotice(),
		Class: "body-small text-error",
	}
}

func (qq *RecoveryCodesDialogContent) TextareaID() string {
	return qq.GetID() + "-textarea"
}

func (qq *RecoveryCodesDialogContent) CodesTextArea() *TextArea {
	return &TextArea{
		Widget: Widget[TextArea]{
			ID: qq.TextareaID(),
		},
		Value:      qq.CodesText,
		Rows:       11,
		IsReadonly: true,
		Class:      "w-full rounded-md border border-outline-variant bg-surface-container-low px-4 py-3 title-small font-mono",
	}
}

func (qq *RecoveryCodesDialogContent) MessagesDataAttributes() *DataAttributes {
	return &DataAttributes{
		Widget: Widget[DataAttributes]{
			ID: qq.StatusID() + "-messages",
		},
		Hidden: true,
		Values: map[string]*Text{
			"backup-codes-title":   qq.BackupCodesTitle(),
			"account-label":        qq.AccountLabel(),
			"generated-label":      qq.GeneratedLabel(),
			"keep-secure-notice":   qq.KeepSecureNotice(),
			"shown-once-notice":    qq.ShownOnceNotice(),
			"codes-heading":        qq.CodesHeading(),
			"print-opened-message": qq.PrintOpenedMessage(),
			"copied-message":       qq.CopiedMessage(),
			"copy-failed-message":  qq.CopyFailedMessage(),
			"downloaded-message":   qq.DownloadedMessage(),
		},
	}
}

func (qq *RecoveryCodesDialogContent) CopyButtonID() string {
	return qq.GetID() + "-copy-btn"
}

func (qq *RecoveryCodesDialogContent) DownloadButtonID() string {
	return qq.GetID() + "-download-btn"
}

func (qq *RecoveryCodesDialogContent) PrintButtonID() string {
	return qq.GetID() + "-print-btn"
}

func (qq *RecoveryCodesDialogContent) PrintButton() *Button {
	return &Button{
		Widget: Widget[Button]{
			ID: qq.PrintButtonID(),
		},
		Label:     T("Print codes"),
		StyleType: ButtonStyleTypeElevated,
	}
}

func (qq *RecoveryCodesDialogContent) CopyButton() *Button {
	return &Button{
		Widget: Widget[Button]{
			ID: qq.CopyButtonID(),
		},
		Label:     T("Copy codes"),
		StyleType: ButtonStyleTypeElevated,
	}
}

func (qq *RecoveryCodesDialogContent) DownloadButton() *Button {
	return &Button{
		Widget: Widget[Button]{
			ID: qq.DownloadButtonID(),
		},
		Label:     T("Download"),
		StyleType: ButtonStyleTypeElevated,
	}
}

func (qq *RecoveryCodesDialogContent) StatusID() string {
	return qq.GetID() + "-status"
}
