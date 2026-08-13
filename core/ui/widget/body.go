package widget

type BodyType string

const (
	BodyTypeSm = "body-sm"
	BodyTypeMd = "body-md"
	BodyTypeLg = "body-lg"
)

type Body struct {
	Widget[Body]
	Type BodyType
	Text *Text
}

func NewBody(typex BodyType, text *Text) *Body {
	return &Body{
		Text: text,
		Type: typex,
	}
}

func NewBodyf(typex BodyType, format string, a ...any) *Body {
	return &Body{
		Text: Tf(format, a...),
		Type: typex,
	}
}
