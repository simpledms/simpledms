package pluginx

import (
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
)

// ExtendNavigationRailFooterItemsHook extends navigation rail footer items.
type ExtendNavigationRailFooterItemsHook interface {
	ExtendNavigationRailFooterItems(
		ctx ctxx.Context,
		items []*wx.NavigationRailItem,
		active string,
	) []*wx.NavigationRailItem
}
