package widget

type DefaultLayout struct {
	Widget[DefaultLayout]

	AppBar  *AppBar
	Content IWidget

	WithPoweredBy bool
	// no pointer to auto initialize
	// ProgressIndicator *ProgressIndicator
}

/*
func (qq *DefaultLayout) GetProgressIndicator() *ProgressIndicator {
	qq.ProgressIndicator = &ProgressIndicator{
		Type:  ProgressIndicatorTypeLinear,
		Size:  ProgressIndicatorSizeSmall,
		Value: nil,
		Color: "",
	}

	return qq.ProgressIndicator
}
*/
