package widget

// SelectField is a native, keyboard-accessible choice among dynamic form options.
type SelectField struct {
	Widget[SelectField]
	Label        *Text
	Name         string
	DefaultValue string
	Options      []*SelectOption
	IsRequired   bool
}
