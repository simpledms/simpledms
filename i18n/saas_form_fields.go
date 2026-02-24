package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// gotext helper for SaaS-specific form labels.
func saasFormFieldsGotextHelper() {
	pp := message.NewPrinter(language.English)
	pp.Sprintf("Sign up")
	pp.Sprintf("Sign up [subject]")
	pp.Sprintf("Organization name")
	pp.Sprintf("Country")
	pp.Sprintf("Accept terms of service")
	pp.Sprintf("Accept privacy policy")
	pp.Sprintf("Registration successful, please check your emails for your password.")
}
