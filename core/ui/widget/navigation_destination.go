package widget

type NavigationDestination struct {
	Widget[NavigationDestination]
	HTMXAttrs

	Href string // TODO good idea?

	Label string
	Icon  string
	// IconSelected *Icon
	Value    string
	IsActive bool
}
