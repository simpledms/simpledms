package widget

// ListStyle controls the visual treatment of a List.
type ListStyle int

const (
	// ListStyleStandard renders transparent expressive list items and is the default.
	ListStyleStandard ListStyle = iota
	// ListStyleSegmented renders filled expressive list items.
	// It currently does not produce visible segmentation in light mode because
	// surface and surfaceBright are equal in the current theme.
	ListStyleSegmented
)

type List struct {
	Widget[List]
	HTMXAttrs
	Children      IWidget
	HideOnDesktop bool
	Style         ListStyle
}

// IsStyleStandard reports whether the list uses the transparent expressive style.
func (qq *List) IsStyleStandard() bool {
	return qq.Style == ListStyleStandard
}
