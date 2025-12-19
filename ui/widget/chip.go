package widget

type Chip struct {
	Widget[Chip]
	HTMXAttrs

	Leading  IWidget
	Label    *Text
	Trailing IWidget

	Name string
}

// TODO
func (qq *Chip) GetRole() string {
	return "checkbox"
}

func (qq *Chip) GetName() string {
	return qq.Name
}
