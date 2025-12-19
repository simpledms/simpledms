package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// TODO impl a wrapper around message.Printer to record to database
type Printer struct {
	*message.Printer
}

func newPrinter(languagex language.Tag) *Printer {
	qq := message.NewPrinter(languagex)
	return &Printer{qq}
}
