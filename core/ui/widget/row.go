package widget

type Row struct {
	Widget[Row]

	OverflowScroll bool
	Wrap           bool

	// TopAlign   bool // TODO wrap in Config or via Prefix?
	// FullHeight bool
	JustifyEnd bool
	// GapXSize overrides the default horizontal gap; GapNone keeps the default gap-x-2 because
	// it is the zero value used by all existing rows.
	GapXSize Gap
	Children IWidget
}

func (qq *Row) GetGapXClass() string {
	switch qq.GapXSize {
	case Gap1:
		return "gap-x-1"
	case Gap3:
		return "gap-x-3"
	case Gap4:
		return "gap-x-4"
	case GapNone, Gap2:
		return "gap-x-2"
	default:
		return "gap-x-2"
	}
}
