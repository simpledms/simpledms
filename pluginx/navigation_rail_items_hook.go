package pluginx

import (
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type ExtendNavigationRailItemsHook interface {
	ExtendNavigationRailItems(
		ctx ctxx.Context,
		items []*wx.NavigationRailItem,
	) []*wx.NavigationRailItem
}
