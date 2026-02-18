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
	pp.Sprintf("Accept terms of service")
	pp.Sprintf("Accept privacy policy")
}
