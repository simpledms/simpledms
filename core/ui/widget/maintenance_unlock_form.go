package widget

type MaintenanceUnlockForm struct {
	Widget[MaintenanceUnlockForm]

	PassphraseLabel *Text
	SubmitLabel     *Text
	RequiredMessage *Text
	InvalidMessage  *Text
	FailureMessage  *Text
	StartupMessage  *Text
}

func NewMaintenanceUnlockForm() *MaintenanceUnlockForm {
	return &MaintenanceUnlockForm{
		PassphraseLabel: T("Application passphrase"),
		SubmitLabel:     T("Unlock application"),
		RequiredMessage: T("Passphrase is required."),
		InvalidMessage:  T("Invalid passphrase."),
		FailureMessage:  T("Something went wrong. Please try again."),
		StartupMessage:  T("Application unlocked. Starting up."),
	}
}
