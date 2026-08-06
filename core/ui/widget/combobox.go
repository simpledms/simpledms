package widget

// combination of Input and Menu
type Combobox struct {
	Widget[Combobox]
	Input *Input // TODO embed?
	Menu  *Menu
}

func (qq *Combobox) GetPopoverTarget() string {
	return qq.GetMenu().GetID()
}

func (qq *Combobox) GetMenu() *Menu {
	if qq.Menu == nil {
		qq.Menu = &Menu{}
	}
	qq.Menu.Position = PositionBottom
	return qq.Menu
}
