package widget

type NarrowLayout struct {
	Widget[NarrowLayout]

	// TODO just a workaround, layout should not know about AppBar
	AppBar *AppBar

	Navigation *NavigationRail
	Content    IWidget

	WithPoweredBy bool
}
