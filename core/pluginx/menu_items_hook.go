package pluginx

import (
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
)

type ExtendMenuItemsHook interface {
	ExtendMenuItems(ctx ctxx.Context, items []*wx.MenuItem) []*wx.MenuItem
}
