package widget

// TODO rename to page or Portial?
type View struct {
	Widget[View]
	// TODO add Title?
	Children IWidget // more flexible than []IWidget
}

// disabled id support in Widget and make clear to prevent accidential misuse...
func (qq *View) GetID() string {
	return ""
}
