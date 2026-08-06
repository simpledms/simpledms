package widget

type formElement struct {
	Name               string
	Type               string // string, int, slice ...
	RawType            string // for example genderpb.Gender for enums
	Element            string // input, select ...
	Attributes         formAttributes
	DefaultValue       string
	LeadingIcon        string
	TrailingIcon       string
	Validation         formValidationRules
	Children           formElements
	Resource           string
	ResourceLabelField string
	Widget             IWidget
}
