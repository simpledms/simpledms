package widget

type DetailsWithSheet struct {
	Widget[DetailsWithSheet]
	AppBar    *AppBar
	Child     IWidget // TODO rename to content?
	Sheet     IWidget
	SideSheet *Dialog
}
