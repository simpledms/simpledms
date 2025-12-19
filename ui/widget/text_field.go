package widget

// TODO extend with menu as replacement for select elements, see:
//		https://m3.material.io/components/menus/guidelines

type TextField struct {
	Widget[TextField]
	HTMXAttrs // TODO not sure if rendered at correct location

	Label        *Text
	Name         string
	Type         string
	Step         string
	IsRequired   bool
	HasAutofocus bool

	LeadingIcon  *Icon
	DefaultValue string

	// legacy stuff
	// necessary in formAttributesWithoutType tmpl
	// Attributes formAttributes
	// Validation formValidationRules
}
