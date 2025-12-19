package i18n

import (
	"golang.org/x/text/language"
)

//go:generate go run ./cmd/extract_form_fields.go
//go:generate go tool gotext update -out catalog.gen.go github.com/simpledms/simpledms

type I18n struct {
}

func NewI18n() *I18n {
	qq := &I18n{}
	qq.init()
	return qq
}

func (qq *I18n) Printer(languagex language.Tag) *Printer {
	return newPrinter(languagex)
}

func (qq *I18n) init() {}
