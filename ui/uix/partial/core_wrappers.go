package partial

import (
	corectxx "github.com/marcobeierer/go-core/ctxx"
	corepartial "github.com/marcobeierer/go-core/ui/uix/partial"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/simpledms/simpledms/common"
)

func NewBase(title *widget.Text, child widget.IWidget) *widget.Base {
	return corepartial.NewBase(title, child)
}

func NewFullscreenPopoverDialogAppBar(
	title string,
	closePopoverTarget string,
	submitBtnLabel string,
) *widget.AppBar {
	return corepartial.NewFullscreenPopoverDialogAppBar(title, closePopoverTarget, submitBtnLabel)
}

func NewFullscreenDialogAppBar(
	title *widget.Text,
	closeButtonHref string,
	actions []widget.IWidget,
) *widget.AppBar {
	return corepartial.NewFullscreenDialogAppBar(title, closeButtonHref, actions)
}

func NewMainMenu(ctx corectxx.Context, infra *common.Infra) *widget.IconButton {
	return corepartial.NewMainMenu(ctx, infra.CoreInfra())
}
