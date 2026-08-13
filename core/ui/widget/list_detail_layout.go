package widget

// https://m3.material.io/foundations/layout/canonical-layouts/list-detail
type ListDetailLayout struct {
	Widget[ListDetailLayout]

	AppBar *AppBar // or PrimaryAppBar and SecondaryAppBar
	List   IWidget

	// DetailAppBar *AppBar
	Detail *DetailsWithSheet

	SelectedItemID string
}

func (qq *ListDetailLayout) GetDetail() *DetailsWithSheet {
	// necessary to render #details block as target if not defined
	if qq.Detail == nil {
		qq.Detail = &DetailsWithSheet{}
	}
	return qq.Detail
}

// TODO use
func (qq *ListDetailLayout) HasSelection() bool {
	return false
}
