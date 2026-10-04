package widget

import (
	"fmt"
	"strings"
)

type Badge struct {
	Widget[Badge]
	HTMXAttrs
	Value     int
	IsInline  bool
	IsInverse bool
	// IsSmall renders a dot without value, for example to indicate a state instead of a count;
	// see https://m3.material.io/components/badges/specs
	IsSmall bool
	// keeps the element in the DOM, for example as HTMX target, but hides it visually
	IsHiddenIfZero bool
}

func (qq *Badge) GetValue() string {
	if qq.Value < 1000 {
		return fmt.Sprintf("%d", qq.Value)
	}
	return "999+"
}

func (qq *Badge) GetClass() string {
	classes := []string{}
	// classes := []string{"badge", "primary"}
	// classes = append(classes, "none")
	if qq.IsHiddenIfZero && qq.Value == 0 {
		classes = append(classes, "hidden")
	}
	return strings.Join(classes, " ")

}
