package widget

type LabelType string

const (
	LabelTypeSm = "label-sm"
	LabelTypeMd = "label-md"
	LabelTypeLg = "label-lg"
)

type Label struct {
	Widget[Label]
	Type LabelType
	Text *Text
	For  string
}

func NewLabel(typex LabelType, text *Text) *Label {
	return &Label{
		Text: text,
		Type: typex,
	}
}

// forx as second param to be more consistent with NewLabelForf
func NewLabelFor(typex LabelType, forx string, text *Text) *Label {
	return &Label{
		Text: text,
		Type: typex,
		For:  forx,
	}
}

func NewLabelf(typex LabelType, format string, a ...any) *Label {
	return &Label{
		Text: Tf(format, a...),
		Type: typex,
	}
}

func NewLabelForf(typex LabelType, forx string, format string, a ...any) *Label {
	return &Label{
		Text: Tf(format, a...),
		Type: typex,
		For:  forx,
	}
}

func (qq *Label) SetFor(forx string) *Label {
	qq.For = forx
	return qq
}
