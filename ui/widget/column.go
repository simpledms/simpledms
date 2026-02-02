package widget

import (
	"strings"
)

type Gap int

const (
	GapNone Gap = iota
	Gap1    Gap = iota
	Gap2    Gap = iota
	Gap3    Gap = iota
	Gap4    Gap = iota
)

// TODO merge with Gap?
type Margin int

const (
	MarginNone Margin = iota
	Margin3
	Margin4
)

type Column struct {
	Widget[Column]
	HTMXAttrs

	MaxWidth bool
	GapYSize Gap
	MarginY  Margin
	Height   string
	// IsNarrowLayout bool

	// necessary for sign in page, alternative would be to also add
	// overflow-y-hidden, but not sure about side effects, so this is simpler
	// TODO make NoOverflowHidden the default?
	NoOverflowHidden bool

	// TODO necessary or should that be the default?
	AutoHeight bool

	Children IWidget
}

func (qq *Column) GetClass() string {
	var classes []string

	switch qq.GapYSize {
	case Gap1:
		classes = append(classes, "gap-y-1")
	case Gap2:
		classes = append(classes, "gap-y-2")
	case Gap3:
		classes = append(classes, "gap-y-3")
	case Gap4:
		classes = append(classes, "gap-y-4")
	case GapNone:
	default:
		// nothing
	}

	switch qq.MarginY {
	case Margin3:
		classes = append(classes, "my-3")
	case Margin4:
		classes = append(classes, "my-4")
	case MarginNone:
	default:
		// nothing
	}

	return strings.Join(classes, " ")
}
