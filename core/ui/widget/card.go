package widget

type CardStyle int

const (
	CardStyleElevated CardStyle = iota
	CardStyleFilled
	CardStyleOutlined
)

type Card struct {
	Widget[Card]
	HTMXAttrs

	Style CardStyle

	// Image *Image
	Headline       *Heading // TODO Title or Headline? Text or Heading?
	Subhead        *Text
	SupportingText *Text

	Content     IWidget
	ContextMenu *Menu

	Actions []*Button
}

func (qq *Card) IsStyleFilled() bool {
	return qq.Style == CardStyleFilled
}

func (qq *Card) IsStyleOutlined() bool {
	return qq.Style == CardStyleOutlined
}

func (qq *Card) IsStyleElevated() bool {
	return qq.Style == CardStyleElevated
}
