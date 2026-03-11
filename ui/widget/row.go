package widget

type Row struct {
	Widget[Row]

	OverflowScroll bool
	Class          string

	// TopAlign   bool // TODO wrap in Config or via Prefix?
	// FullHeight bool
	JustifyEnd bool
	Children   IWidget
}
