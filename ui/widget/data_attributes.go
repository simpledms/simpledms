package widget

type DataAttributes struct {
	Widget[DataAttributes]

	Class  string
	Hidden bool
	Values map[string]*Text
}
