package widget

type TextArea struct {
	Widget[TextArea]

	Name         string
	Value        string
	Rows         int
	Class        string
	IsReadonly   bool
	HasAutofocus bool
}

func (qq *TextArea) GetRows() int {
	if qq.Rows > 0 {
		return qq.Rows
	}

	return 6
}
