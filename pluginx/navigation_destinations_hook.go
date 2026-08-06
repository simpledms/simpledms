package pluginx

import (
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
)

type ExtendNavigationDestinationsHook interface {
	ExtendNavigationDestinations(
		ctx ctxx.Context,
		destinations []*wx.NavigationDestination,
	) []*wx.NavigationDestination
}
