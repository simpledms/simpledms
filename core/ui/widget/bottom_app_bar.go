package widget

type BottomAppBar struct {
	Widget[BottomAppBar]
	Actions  []IWidget
	Children IWidget // TODO or Content?
	FAB      *FloatingActionButton
}
