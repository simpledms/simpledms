package widget

import (
	"strings"
)

type Search struct {
	Widget[Search]
	HTMXAttrs

	LeadingIconButton          IWidget // IconButton or Icon
	LeadingIconButtonAltMobile IWidget
	SupportingText             *Text
	SupportingTextAltMobile    *Text
	TrailingIconButton         *IconButton

	Autofocus bool

	Name  string
	Value string
}

func (qq *Search) GetClass() string {
	classes := []string{
		// "field", "round", "border", "fill", "max",
	}
	if qq.LeadingIconButton != nil {
		// check only works if interface has no type
		// var backButton IWidget = works
		// var backButton *Link = doesn't work (interface has type set)
		// classes = append(classes, "prefix")
	}
	if qq.TrailingIconButton != nil {
		// classes = append(classes, "suffix")
	}
	return strings.Join(classes, " ")
}
