package widget

import (
	"strings"
)

// TODO rename to Tabs?, maybe refactor is necessary
type TabBar struct {
	Widget[TabBar]
	Tabs        []*Tab // TODO Tabs or Children
	IsSecondary bool
	IsFlowing   bool // TODO find a better name; IsFloating?
	ActiveTab   string
	// TODO ScrollableContent should be implicit, always automatic and not responsibiliy of caller
	ActiveTabContent *ScrollableContent
	NoContentMargin  bool
}

func (qq *TabBar) GetClass() string {
	var classes []string
	if !qq.IsSecondary {
		// classes = append(classes, "min")
	}
	return strings.Join(classes, " ")
}

func (qq *TabBar) GetTabs() []*Tab {
	for qi, tab := range qq.Tabs {
		tabID := strings.ToLower(tab.Label.StringUntranslated())
		tabID = strings.ReplaceAll(tabID, " ", "-")
		tab.IsActive = (qi == 0 && qq.ActiveTab == "") || qq.ActiveTab == tabID
		tab.IsFlowing = qq.IsFlowing
	}
	return qq.Tabs
}
