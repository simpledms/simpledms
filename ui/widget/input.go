package widget

// Deprecated: use TextField instead; only used for search at the moment
type Input struct {
	Widget[Input]
	HTMXAttrs

	LeadingIcon  *Icon
	Label        *Text
	TrailingIcon *Icon

	Name         string
	Type         string // TODO enum
	Placeholder  string
	HasAutofocus bool

	Step string // for numbers
}

func (qq *Input) GetType() string {
	if qq.Type != "" {
		return qq.Type
	}
	return "text"
}
