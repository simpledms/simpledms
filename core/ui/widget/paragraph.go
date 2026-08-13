package widget

type Paragraph struct {
	Widget[Paragraph]

	Text  *Text
	Class string
}

func NewParagraph(text *Text) *Paragraph {
	return &Paragraph{
		Text: text,
	}
}
