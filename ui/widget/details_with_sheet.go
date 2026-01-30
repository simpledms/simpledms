package widget

type DetailsWithSheet struct {
	Widget[DetailsWithSheet]
	HTMXAttrs
	AppBar    *AppBar
	Child     IWidget // TODO rename to content?
	Sheet     IWidget
	SideSheet *Dialog
}
