package widget

type Switch struct {
	Widget[Switch]
	HTMXAttrs

	Label          *Text
	SupportingText *Text
	Name           string
	Value          string
	IsChecked      bool
	IsDisabled     bool
	IsRequired     bool
	UncheckedIcon  *Icon
	CheckedIcon    *Icon
}

func (qq *Switch) GetValue() string {
	if qq.Value == "" {
		return "1"
	}
	return qq.Value
}
