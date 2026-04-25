package widget

// TODO not used as intended as of 17 June 24...
type NavigationRail struct {
	Widget[NavigationRail]
	// PrimaryFAB    *FloatingActionButton
	MenuBtn      *IconButton
	FABs         []*FloatingActionButton
	Destinations []*NavigationDestination
}
