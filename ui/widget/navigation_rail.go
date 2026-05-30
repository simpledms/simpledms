package widget

const (
	MaxCompactNavigationRailItems   = 4
	MaxCollapsedNavigationRailItems = 7
)

type NavigationRail struct {
	Widget[NavigationRail]
	HTMXAttrs
	MenuBtn        *IconButton
	FABs           []*FloatingActionButton
	Action         IWidget
	ActionExpanded IWidget
	CompactItems   []*NavigationRailItem
	TopItems       []*NavigationRailItem
	Items          []*NavigationRailItem
	FooterItems    []*NavigationRailItem
	Destinations   []*NavigationDestination

	activeValue string
}

func (qq *NavigationRail) GetItems() []*NavigationRailItem {
	if len(qq.Items) > 0 {
		return qq.Items
	}

	items := make([]*NavigationRailItem, 0, len(qq.Destinations))
	for _, destination := range qq.Destinations {
		items = append(items, NewNavigationRailItemFromDestination(destination))
	}
	return items
}

func (qq *NavigationRail) CollapsedItems() []*NavigationRailItem {
	return qq.limitedItems(MaxCollapsedNavigationRailItems)
}

func (qq *NavigationRail) ExpandedItems() []*NavigationRailItem {
	items := qq.GetItems()
	expandedItems := make([]*NavigationRailItem, 0, len(items))
	for _, item := range items {
		if item == nil || item.IsCollapsedOnly {
			continue
		}
		expandedItems = append(expandedItems, item)
	}
	return expandedItems
}

func (qq *NavigationRail) CompactNavigationItems() []*NavigationRailItem {
	if len(qq.CompactItems) > 0 {
		return qq.CompactItems
	}
	return qq.limitedItems(MaxCompactNavigationRailItems)
}

func (qq *NavigationRail) limitedItems(limit int) []*NavigationRailItem {
	items := make([]*NavigationRailItem, 0, limit)
	for _, item := range qq.GetItems() {
		if item == nil || item.IsGroup() || item.IsSubheader || item.IsExpandedOnly {
			continue
		}
		items = append(items, item)
		if len(items) == limit {
			break
		}
	}
	return items
}

func (qq *NavigationRail) SetActiveValue(value string) {
	if value == "" || qq.activeValue == value {
		return
	}
	qq.activeValue = value
	qq.clearActiveItems()
	qq.setActiveValueOnItems(value, qq.GetItems())
	qq.setActiveValueOnItems(value, qq.CompactItems)
	qq.setActiveValueOnItems(value, qq.TopItems)
	qq.setActiveValueOnItems(value, qq.FooterItems)
}

func (qq *NavigationRail) clearActiveItems() {
	for _, item := range qq.GetItems() {
		item.ClearActive()
	}
	for _, item := range qq.CompactItems {
		item.ClearActive()
	}
	for _, item := range qq.TopItems {
		item.ClearActive()
	}
	for _, item := range qq.FooterItems {
		item.ClearActive()
	}
}

func (qq *NavigationRail) setActiveValueOnItems(value string, items []*NavigationRailItem) bool {
	for _, item := range items {
		if item.SetActiveValue(value) {
			return true
		}
	}
	return false
}
