package pluginx

import (
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
)

type ExtendNavigationRailItemsHook interface {
	ExtendNavigationRailItems(
		ctx ctxx.Context,
		items []*wx.NavigationRailItem,
	) []*wx.NavigationRailItem
}
