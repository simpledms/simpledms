package i18n

import (
	"golang.org/x/text/language"

	corei18n "github.com/marcobeierer/go-core/i18n"
)

//go:generate go run ./cmd/extract_form_fields.go
//go:generate go tool gotext update -out catalog.gen.go github.com/simpledms/simpledms

// I18n is the shared core i18n runtime configured for SimpleDMS languages.
type I18n = corei18n.I18n

// NewI18n creates the SimpleDMS i18n runtime.
func NewI18n() *I18n {
	return corei18n.NewI18n(
		language.English,
		language.German,
		language.French,
		language.Italian,
	)
}
