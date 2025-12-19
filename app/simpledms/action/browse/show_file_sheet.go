package browse

import (
	"context"

	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/enttenant"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
)

type ShowFileSheetData struct {
}

type ShowFileSheetState struct {
}

type ShowFileSheet struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewShowFileSheet(infra *common.Infra, actions *Actions) *ShowFileSheet {
	return &ShowFileSheet{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("show-file-sheet"),
			true,
		),
	}
}

func (qq *ShowFileSheet) Widget(
	ctx context.Context,
	tx *enttenant.Tx,
	state *ShowFileSheetState,
	filex *enttenant.File,
) *wx.Column {
	return nil
}
