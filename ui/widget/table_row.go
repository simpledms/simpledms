package widget

type TableRow struct {
	Widget[TableRow]
	HTMXAttrs

	Cells       []*TableCell
	ContextMenu *Menu
	IsSelected  bool
}
