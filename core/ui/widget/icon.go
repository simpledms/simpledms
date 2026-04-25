package widget

import "strings"

type Shape int

const (
	ShapeNone Shape = iota
	ShapeCircle
)

type Color int

const (
	ColorNone Color = iota
	ColorPrimary
	ColorSecondary
)

type IconSize int

const (
	IconSizeDefault IconSize = iota
	IconSizeSmall
	// IconSizeMedium
	// IconSizeExtraSmall
	IconSizeLarge
)

type Icon struct {
	Widget[Icon]

	Name  string // TODO or data or icon?
	color Color
	Size  IconSize

	// TODO refactor to enum
	hasSmallPadding      bool
	hasPadding           bool
	hasVerticalPadding   bool
	hasHorizontalPadding bool

	hasBorder bool
	shape     Shape
}

func NewIcon(name string) *Icon {
	return &Icon{Name: name}
}

func (qq *Icon) Color(color Color) *Icon {
	qq.color = color
	return qq
}

func (qq *Icon) SmallPadding() *Icon {
	qq.hasSmallPadding = true
	return qq
}

func (qq *Icon) VerticalPadding() *Icon {
	qq.hasVerticalPadding = true
	return qq
}

func (qq *Icon) HorizontalPadding() *Icon {
	qq.hasHorizontalPadding = true
	return qq
}

func (qq *Icon) Padding() *Icon {
	qq.hasPadding = true
	return qq
}

func (qq *Icon) Border() *Icon {
	qq.hasBorder = true
	return qq
}

func (qq *Icon) Shape(shape Shape) *Icon {
	qq.shape = shape
	return qq
}

func (qq *Icon) GetClass() string {
	var classes []string
	switch qq.Size {
	// case IconSizeExtraSmall:
	// classes = append(classes, "icon-xs")
	case IconSizeSmall:
		classes = append(classes, "icon-sm")
	// case IconSizeMedium:
	// classes = append(classes, "icon-md")
	case IconSizeLarge:
		classes = append(classes, "icon-lg")
	default:
		classes = append(classes, "icon-base")
	}

	if qq.hasPadding {
		classes = append(classes, "p-2")
	}
	if qq.hasSmallPadding {
		// classes = append(classes, "small-padding")
	}
	if qq.hasVerticalPadding {
		// classes = append(classes, "vertical-padding")
	}
	if qq.hasHorizontalPadding {
		// classes = append(classes, "horizontal-padding")
	}
	if qq.hasBorder {
		classes = append(classes, "border")
	}
	switch qq.shape {
	case ShapeCircle:
		classes = append(classes, "rounded-full")
	}
	switch qq.color {
	case ColorPrimary:
		classes = append(classes, "text-primary")
	case ColorSecondary:
		classes = append(classes, "text-secondary")
	}
	return strings.Join(classes, " ")
}

/*
type IconOpts struct {
	Color string
	Size  int
}

type icon struct {
	icon string // TODO or data??
	opts *IconOpts
}

func Icon(iconx string) *icon {
	return &icon{
		icon: iconx,
		opts: &IconOpts{},
	}
}

func IconWithOpts(iconx string, opts *IconOpts) *icon {
	return &icon{
		icon: iconx,
		opts: opts,
	}
}

func (qq *icon) Render() {

}
*/
