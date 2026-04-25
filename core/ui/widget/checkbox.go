package widget

type Checkbox struct {
	Widget[Checkbox]
	HTMXAttrs

	Label      *Text
	Name       string
	Value      string
	IsChecked  bool
	IsRequired bool // for forms

	// not exported to not confuse user
	isStateInherited bool

	// HxPost   string
	// HxVals   template.JS
	// HxTarget string
	// HxSwap   string
	// HxSelect  string
	// HxConfirm string

	label *Label
}

func (qq *Checkbox) GetLabel() *Label {
	if qq.label == nil {
		qq.label = NewLabelFor(LabelTypeLg, qq.GetID()+"-checkbox", qq.Label)
	}
	return qq.label
}

func (qq *Checkbox) GetValue() string {
	if qq.Value == "" {
		return "1" // TODO is this okay as a default?
	}
	return qq.Value
}

func (qq *Checkbox) IsStateInherited() bool {
	return qq.isStateInherited
}

func (qq *Checkbox) SetIsStateInherited(isStateInherited bool) {
	qq.isStateInherited = isStateInherited
}
