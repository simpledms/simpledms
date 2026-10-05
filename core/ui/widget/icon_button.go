package widget

// TODO add link capabilities on button? otherwise it's always used with Link or Form
// TODO merge with Button?
type IconButton struct {
	Widget[IconButton]
	HTMXAttrs

	Icon                string
	PopoverTarget       string
	PopoverTargetAction string
	// HasPopupMenu marks PopoverTarget as a menu; the context menu runtime keeps aria-expanded
	// in sync.
	HasPopupMenu bool
	ReplaceURL   string

	Tooltip *Text
	// Label is an accessible name, not visible button text.
	Label      *Text
	IsToggle   bool
	IsSelected bool

	Children IWidget // used for menu // TODO get rid of label and icon?
}

// necessary to ship large script only if needed
func (qq *IconButton) HasMenu() bool {
	_, hasMenu := qq.Children.(*Menu)
	return hasMenu
}

func (qq *IconButton) GetPopoverTarget() string {
	if qq.PopoverTarget != "" {
		return qq.PopoverTarget
	}
	if menu, hasMenu := qq.Children.(*Menu); hasMenu {
		return menu.GetID()
	}
	return ""
}
