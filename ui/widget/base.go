package widget

// TODO name? HTML or Base or Main something else?
type Base struct {
	Widget[Base]
	Title    *Text   // TODO string or Text
	Content  IWidget // TODO naming?
	Children IWidget // TODO naming?

	ShouldRefreshEvery60Seconds bool
	// Scaffold *Scaffold

	ProgressIndicator *ProgressIndicator
}

func (qq *Base) GetProgressIndicator() *ProgressIndicator {
	if qq.ProgressIndicator == nil {
		qq.ProgressIndicator = &ProgressIndicator{
			Type:          ProgressIndicatorTypeLinear,
			Size:          ProgressIndicatorSizeSmall,
			IsFixedLayout: true,
		}
	}
	return qq.ProgressIndicator
}
