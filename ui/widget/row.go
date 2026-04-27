package widget

type Row struct {
	Widget[Row]

	OverflowScroll bool
	Wrap           bool

	// TopAlign   bool // TODO wrap in Config or via Prefix?
	// FullHeight bool
	JustifyEnd bool
	Children   IWidget
}
