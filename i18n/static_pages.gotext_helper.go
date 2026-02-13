package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// gotext helper for static page labels;
// necessary because they are not auto detected by `gotext update`
func staticPagesGotextHelper() {
	pp := message.NewPrinter(language.English)
	pp.Sprintf("Imprint")
	pp.Sprintf("Privacy policy")
	pp.Sprintf("Terms of service")
}
