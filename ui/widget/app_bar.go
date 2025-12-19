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
	if qq.Search == nil {
		return qq.Search
	}
	if qq.Search.LeadingIconButton == nil {
		qq.Search.LeadingIconButton = qq.Leading
		qq.Search.LeadingIconButtonAltMobile = qq.LeadingAltMobile
		// qq.Leading = nil // if not reset, prevents rendering of menus...
	}
	// TODO find a better solution, maybe add PrimaryAction?
	//
	// disabled on 26.07.2025 because of conflicts when a button with context menu
	// is rendered twice (in search bar and next to search bar). New layout (rendering
	// all actions next to search bar) is according to material 3 expressive
	/*if qq.Search.TrailingIconButton == nil && len(qq.Actions) > 0 {
		lastAction := qq.Actions[len(qq.Actions)-1]
		if lastActionBtn, ok := lastAction.(*IconButton); ok {
			qq.Search.TrailingIconButton = lastActionBtn
		}
	}*/
	return qq.Search
}

func (qq *AppBar) IsSearch() bool {
	_, isSearch := qq.Title.(*Search)
	return isSearch
}
