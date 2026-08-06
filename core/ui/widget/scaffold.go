package widget

type Scaffold struct {
	AppBar               *AppBar
	Body                 IWidget
	Drawer               *Drawer
	FloatingActionButton *FloatingActionButton
	NavigationBar        *NavigationBar
}
