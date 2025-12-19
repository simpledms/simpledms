package partial

import (
	wx "github.com/simpledms/simpledms/ui/widget"
)

// TODO widget or block?
func NewFullscreenPopoverDialogAppBar(title string, closePopoverTarget string, submitBtnLabel string) *wx.AppBar {
	var actions []wx.IWidget
	if submitBtnLabel != "" {
		actions = append(actions, &wx.Button{
			Type:  "submit",
			Label: wx.T(submitBtnLabel),
		})
	}

	return &wx.AppBar{
		Leading: &wx.IconButton{
			Icon:                "close",
			PopoverTarget:       closePopoverTarget,
			PopoverTargetAction: "hide",
		},
		Title: &wx.AppBarTitle{
			Text: wx.T(title), // TODO add filename
		},
		Actions: actions,
	}
}

func NewFullscreenDialogAppBar(title *wx.Text, closeButtonHref string, actions []wx.IWidget) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.IconButton{
			Icon: "close",
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: closeButtonHref,
			},
		},
		Title: &wx.AppBarTitle{
			Text: title,
		},
		Actions: actions,
	}
}
