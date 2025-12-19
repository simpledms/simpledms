package partial

import wx "github.com/simpledms/simpledms/ui/widget"

type Basex struct {
}

func NewBase(title *wx.Text, child wx.IWidget) *wx.Base {
	return &wx.Base{
		Title: title,
		Content: []wx.IWidget{
			child,
		},
		Children: []wx.IWidget{
			&wx.Container{
				Widget: wx.Widget[wx.Container]{
					ID: "popovers",
				},
			},
		},
	}
}
