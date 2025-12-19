package widget

type Row struct {
	Widget[Row]

	OverflowScroll bool

	// TopAlign   bool // TODO wrap in Config or via Prefix?
	// FullHeight bool
	Children IWidget
}
