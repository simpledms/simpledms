package widget

type TextAreaStyleType int

const (
	TextAreaStyleTypeDefault TextAreaStyleType = iota
	TextAreaStyleTypeCompact
	TextAreaStyleTypeFullHeight
)

type TextArea struct {
	Widget[TextArea]

	Name         string
	Value        string
	Rows         int
	StyleType    TextAreaStyleType
	IsReadonly   bool
	HasAutofocus bool
}

func (qq *TextArea) GetRows() int {
	if qq.Rows > 0 {
		return qq.Rows
	}

	return 6
}

func (qq *TextArea) GetClass() string {
	class := "w-full rounded-md border border-outline-variant " +
		"bg-surface-container-low px-4 py-3 title-small font-mono"

	switch qq.StyleType {
	case TextAreaStyleTypeCompact:
		return class
	case TextAreaStyleTypeFullHeight:
		return "w-full min-h-full [field-sizing:content] resize-none rounded-md border " +
			"border-outline-variant bg-surface-container-low px-4 py-3 title-small font-mono"
	default:
		return "w-full min-h-[12rem] rounded-md border border-outline-variant " +
			"bg-surface-container-low px-4 py-3 title-small font-mono"
	}
}
