package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Embedded service config fields are not discovered by the action-field extractor.
//
//nolint:unused // gotext discovers these translation labels without a runtime call.
func serviceConfigFormFieldsGotextHelper() {
	pp := message.NewPrinter(language.English)
	pp.Sprintf("Turnstile site key")
	pp.Sprintf("Turnstile secret")
}
