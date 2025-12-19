package widget

import (
	"fmt"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/simpledms/simpledms/app/simpledms/ctxx"
)

// TODO get rid of global var...
// var printer = i18n.NewPrinter()

// TODO necessary or just use string? may allow for easier control of global styling?
// type Text struct {
// Data string
// }
type Text struct {
	Widget[Text]

	format string // TODO rename to Format?
	args   []any

	Wrap          bool
	NoTranslation bool // used later
	IsParagraph   bool
	IsBold        bool
}

func T(str string) *Text {
	// only for gotext extract; Sprint seems not to work, just Sprintf;
	// Printer.Sprintf in String() is not recognized because a struct variable is used as string,
	// works only if changed to a hard-coded string, not sure why...
	//
	// performance impact should probably be negligible
	_ = message.NewPrinter(language.English).Sprintf(str)
	return &Text{
		format: str,
	}
}

func Tf(format string, a ...any) *Text {
	// see comment above in T()
	_ = message.NewPrinter(language.English).Sprintf(format, a...)
	return &Text{
		format: format,
		args:   a,
	}
}

// the u means user-provided, thus don't translate
func Tu(str string) *Text {
	return &Text{
		format:        str,
		NoTranslation: true,
	}
}

func Tuf(format string, a ...any) *Text {
	return &Text{
		format:        format,
		args:          a,
		NoTranslation: true,
	}
}

func P(str string) *Text {
	_ = message.NewPrinter(language.English).Sprintf(str)
	return &Text{
		format:      str,
		IsParagraph: true,
	}
}

func (qq *Text) SetWrap() *Text {
	qq.Wrap = true
	return qq
}

/*
func (qq *Text) SetNoTranslation() *Text {
	qq.NoTranslation = true
	return qq
}
*/

// used in HTTPErr for error creation
func (qq *Text) StringUntranslated() string {
	return qq.format
}

func (qq *Text) String(ctx ctxx.Context) string {
	if qq.NoTranslation {
		// never has arguments, Sprintf just for robustness against future changes
		// TODO does it have performance impact?
		return fmt.Sprintf(qq.format, qq.args...)
	}

	// TODO does this make sense?
	for qi, arg := range qq.args {
		argy, isText := arg.(*Text)
		if isText {
			qq.args[qi] = argy.String(ctx)
		}
	}

	return ctx.VisitorCtx().Printer.Sprintf(qq.format, qq.args...)
}

func (qq *Text) GetString() string {
	return qq.String(qq.GetContext())
}

// only works for paragraphs or if wrapped
func (qq *Text) SetBold() *Text {
	qq.IsBold = true
	return qq
}
