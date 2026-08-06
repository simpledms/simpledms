package widget

type AppBar struct {
	Widget[AppBar]

	Leading          IWidget
	LeadingAltMobile IWidget // used for main menu on mobile
	Title            IWidget
	Actions          []IWidget
	Search           *Search
	// IsBottom bool // TODO not sure if good idea, maybe separate element, seems to be very different
	// Bottom  Widget

	ProgressIndicator *ProgressIndicator
}

func (qq *AppBar) GetProgressIndicator() *ProgressIndicator {
	// for the moment global indicator is enough, in app bar doesn't add much value
	// and automatic selection of correct one if we have multiple is not straight forward
	return nil
	/*
		if qq.ProgressIndicator == nil {
			qq.ProgressIndicator = &ProgressIndicator{}
		}
		return qq.ProgressIndicator
	*/
}

func (qq *AppBar) GetSearch() *Search {
	return qq.Search
}

func (qq *AppBar) IsSearch() bool {
	_, isSearch := qq.Title.(*Search)
	return isSearch
}
