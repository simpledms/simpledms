package widget

// TODO add link capabilities on button? otherwise it's always used with Link or Form
// TODO merge with Button?
type IconButton struct {
	Widget[IconButton]
	HTMXAttrs

	Icon                string
	PopoverTarget       string
	PopoverTargetAction string
	ReplaceURL          string

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
