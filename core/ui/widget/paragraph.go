package widget

type ParagraphStyleType int

const (
	ParagraphStyleTypeDefault ParagraphStyleType = iota
	ParagraphStyleTypeSupporting
	ParagraphStyleTypeSupportingSmall
	ParagraphStyleTypeError
	ParagraphStyleTypeErrorReserved
	ParagraphStyleTypeStatus
)

type Paragraph struct {
	Widget[Paragraph]

	Text      *Text
	StyleType ParagraphStyleType
}

func NewParagraph(text *Text) *Paragraph {
	return &Paragraph{
		Text: text,
	}
}

func (qq *Paragraph) GetClass() string {
	switch qq.StyleType {
	case ParagraphStyleTypeSupporting:
		return "body-medium text-on-surface-variant"
	case ParagraphStyleTypeSupportingSmall:
		return "body-small text-on-surface-variant"
	case ParagraphStyleTypeError:
		return "body-small text-error"
	case ParagraphStyleTypeErrorReserved:
		return "body-small text-error min-h-5"
	case ParagraphStyleTypeStatus:
		return "body-lg text-on-surface p-4"
	default:
		return "body-md text-on-surface"
	}
}
