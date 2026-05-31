package pluginx

import (
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
)

// ExtendNavigationRailFooterItemsHook extends navigation rail footer items.
type ExtendNavigationRailFooterItemsHook interface {
	ExtendNavigationRailFooterItems(
		ctx ctxx.Context,
		items []*wx.NavigationRailItem,
		active string,
	) []*wx.NavigationRailItem
}
