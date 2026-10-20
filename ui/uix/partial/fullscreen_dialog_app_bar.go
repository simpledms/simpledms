package partial

import (
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ui/uix/event"
)

func NewFullscreenDialogAppBar(title *widget.Text, closeButtonHref string, actions []widget.IWidget) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.IconButton{
			Icon:    "close",
			Tooltip: widget.T("Close"),
			HTMXAttrs: widget.HTMXAttrs{
				HxGet: closeButtonHref,
				HxOn:  event.DetailsClosed.HxOn("click"),
			},
		},
		Title: &widget.AppBarTitle{
			Text: title,
		},
		Actions: actions,
	}
}
