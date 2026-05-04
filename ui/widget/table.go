package widget

type Table struct {
	Widget[Table]
	HTMXAttrs

	Columns      []*TableColumn
	Rows         []*TableRow
	HideOnMobile bool
}
