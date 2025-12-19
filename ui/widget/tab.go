package widget

import (
	"strings"
)

type Tab struct {
	Widget[Tab]
	HTMXAttrs

	Icon     string
	Label    *Text
	Badge    *Badge
	IsActive bool

	IsFlowing bool // TODO find a better name; IsFloating?

	// for use in side sheets
	// TODO should be on TabBar
	IncreasedHeight bool
}

func (qq *Tab) GetClass() string {
	var classes []string
	if qq.Icon != "" {
		classes = append(classes, "h-16")
	} else if qq.IncreasedHeight {
		// TODO as default if always used in sheet?
		classes = append(classes, "h-14")
	} else {
		classes = append(classes, "h-12")
	}
	return strings.Join(classes, " ")
}
