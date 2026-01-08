package widget

// TODO rename to content?
type ScrollableContent struct {
	Widget[ScrollableContent]
	HTMXAttrs

	Children     IWidget
	BottomAppBar *BottomAppBar

	// padding, not margin, so that scrollbar is outside
	PaddingX bool // not used as of 13 April 2025
	GapY     bool
	MarginY  bool
	FlexCol  bool
}

func (qq *ScrollableContent) SetMarginY(marginY bool) *ScrollableContent {
	qq.MarginY = marginY
	return qq
}
