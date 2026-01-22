package browse

import (
	"context"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/db/enttenant"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
)

type ShowFileSheetPartialData struct {
}

type ShowFileSheetPartialState struct {
}

type ShowFileSheetPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewShowFileSheetPartial(infra *common.Infra, actions *Actions) *ShowFileSheetPartial {
	return &ShowFileSheetPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("show-file-sheet"),
			true,
		),
	}
}

func (qq *ShowFileSheetPartial) Widget(
	ctx context.Context,
	tx *enttenant.Tx,
	state *ShowFileSheetPartialState,
	filex *enttenant.File,
) *wx.Column {
	return nil
}
