package partial

import (
	"github.com/simpledms/simpledms/core/ui/widget"
)

// TODO widget or block?
func NewFullscreenPopoverDialogAppBar(title string, closePopoverTarget string, submitBtnLabel string) *widget.AppBar {
	var actions []widget.IWidget
	if submitBtnLabel != "" {
		actions = append(actions, &widget.Button{
			Type:  "submit",
			Label: widget.T(submitBtnLabel),
		})
	}

	return &widget.AppBar{
		Leading: &widget.IconButton{
			Icon:                "close",
			Tooltip:             widget.T("Close"),
			PopoverTarget:       closePopoverTarget,
			PopoverTargetAction: "hide",
		},
		Title: &widget.AppBarTitle{
			Text: widget.T(title), // TODO add filename
		},
		Actions: actions,
	}
}

func NewFullscreenDialogAppBar(title *widget.Text, closeButtonHref string, actions []widget.IWidget) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.IconButton{
			Icon:    "close",
			Tooltip: widget.T("Close"),
			HTMXAttrs: widget.HTMXAttrs{
				HxGet: closeButtonHref,
			},
		},
		Title: &widget.AppBarTitle{
			Text: title,
		},
		Actions: actions,
	}
}
