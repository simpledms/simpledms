package widget

const defaultNavigationRailTargetID = "navigationRailAndBar"

type NavigationRailToggle struct {
	Widget[NavigationRailToggle]

	TargetID string
	Tooltip  *Text
}

func (qq *NavigationRailToggle) GetTargetID() string {
	if qq.TargetID != "" {
		return qq.TargetID
	}
	return defaultNavigationRailTargetID
}

func (qq *NavigationRailToggle) GetTooltip() *Text {
	if qq.Tooltip != nil {
		return qq.Tooltip
	}
	return T("Open main menu")
}
