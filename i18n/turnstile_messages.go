package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Turnstile's template messages need explicit extraction for the translation catalog.
//
//nolint:unused // gotext discovers these labels without a runtime call.
func turnstileMessagesGotextHelper() {
	pp := message.NewPrinter(language.English)
	pp.Sprintf("Verification failed. Please try again.")
	pp.Sprintf("Please wait for verification to complete.")
	pp.Sprintf("Verification expired. Please try again.")
	pp.Sprintf("Verification is temporarily unavailable. Please try again.")
	pp.Sprintf("Registration is being submitted. Please wait.")
	pp.Sprintf("Verification timed out. Please try again.")
	pp.Sprintf("JavaScript is required for verification.")
}
