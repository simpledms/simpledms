package widget

type DataAttributes struct {
	Widget[DataAttributes]

	Hidden bool
	Values map[string]*Text
}
