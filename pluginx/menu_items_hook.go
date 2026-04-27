package pluginx

import (
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type ExtendMenuItemsHook interface {
	ExtendMenuItems(ctx ctxx.Context, items []*wx.MenuItem) []*wx.MenuItem
}
