package widget

// Toolbar contains actions related to the current page.
type Toolbar struct {
	Widget[Toolbar]

	label      string
	children   []IWidget // TODO or Content?
	isFloating bool
	isVertical bool
	isVibrant  bool
}

// NewToolbar creates a standard horizontal docked toolbar.
func NewToolbar(label string, children ...IWidget) *Toolbar {
	return &Toolbar{
		label:    label,
		children: children,
	}
}

// Label returns the toolbar's accessible label.
func (qq *Toolbar) Label() string {
	return qq.label
}

// Children returns the toolbar actions.
func (qq *Toolbar) Children() []IWidget {
	return qq.children
}

// IsFloating reports whether the toolbar floats above the page content.
func (qq *Toolbar) IsFloating() bool {
	return qq.isFloating
}

// IsVertical reports whether the floating toolbar is vertical.
func (qq *Toolbar) IsVertical() bool {
	return qq.isVertical
}

// IsVibrant reports whether the toolbar uses the vibrant color scheme.
func (qq *Toolbar) IsVibrant() bool {
	return qq.isVibrant
}

// SetFloating changes the toolbar from docked to floating.
func (qq *Toolbar) SetFloating() *Toolbar {
	qq.isFloating = true
	return qq
}

// SetVertical changes the toolbar to a vertical floating toolbar.
func (qq *Toolbar) SetVertical() *Toolbar {
	qq.isFloating = true
	qq.isVertical = true
	return qq
}

// SetVibrant applies the vibrant color scheme.
func (qq *Toolbar) SetVibrant() *Toolbar {
	qq.isVibrant = true
	return qq
}
