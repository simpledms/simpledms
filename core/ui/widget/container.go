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
	if qq.MaxWidth {
		// classes = append(classes, "max")
	}
	if qq.Scroll {
		// classes = append(classes, "scroll")
	}
	return strings.Join(classes, " ")
}
