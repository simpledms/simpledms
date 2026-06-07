package pluginx

import (
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type ExtendNavigationDestinationsHook interface {
	ExtendNavigationDestinations(
		ctx ctxx.Context,
		destinations []*wx.NavigationDestination,
	) []*wx.NavigationDestination
}
