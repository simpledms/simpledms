package partial

import (
	"github.com/simpledms/simpledms/core/ui/widget"
)

type Basex struct {
}

func NewBase(title *widget.Text, child widget.IWidget) *widget.Base {
	return &widget.Base{
		Title: title,
		Content: []widget.IWidget{
			child,
		},
		Children: []widget.IWidget{
			&widget.Container{
				Widget: widget.Widget[widget.Container]{
					ID: "popovers",
				},
			},
		},
	}
}
