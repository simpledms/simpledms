package widget

type MainLayout struct {
	Widget[MainLayout]

	Navigation *NavigationRail
	Content    IWidget

	SideSheet *Dialog
	// not in use as of 13 April 2025
	Sheet IWidget
}
