package widget

type FilterChipType int

const (
	FilterChipTypeCheckbox FilterChipType = iota
	FilterChipTypeRadio
)

type FilterChip struct {
	Widget[FilterChip]
	HTMXAttrs

	Type FilterChipType

	Label        *Text
	LeadingIcon  string
	TrailingIcon string

	Name  string
	Value string

	IsChecked    bool
	IsSuggestion bool
}

func (qq *FilterChip) GetInputType() string {
	if qq.Type == FilterChipTypeRadio {
		return "radio"
	}
	return "checkbox"
}

func (qq *FilterChip) GetTooltip() *Text {
	if qq.IsSuggestion && !qq.IsChecked {
		return T("Suggestion")
	}
	return T("")
}
