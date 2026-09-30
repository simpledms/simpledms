package widget

import (
	"strings"
)

type Container struct {
	Widget[Container]
	HTMXAttrs

	MaxWidth bool
	// Height   string
	Scroll    bool
	MaxHeight bool
	GapY      bool
	Gap       bool
	// FlexGrow bool
	// HideOnMobile bool
	// MobileOnly   bool

	Child IWidget
}

func (qq *Container) GetClass() string {
	classes := []string{}
	// classes = append(classes, "max")
	// classes = append(classes, "scroll")
	return strings.Join(classes, " ")
}
