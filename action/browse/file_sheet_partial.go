package browse

import (
	"context"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/db/enttenant"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
)

type FileSheetPartialData struct {
}

type FileSheetPartialState struct {
}

type FileSheetPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileSheetPartial(infra *common.Infra, actions *Actions) *FileSheetPartial {
	return &FileSheetPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("file-sheet-partial"),
			true,
		),
	}
}

func (qq *FileSheetPartial) Widget(
	ctx context.Context,
	tx *enttenant.Tx,
	state *FileSheetPartialState,
	filex *enttenant.File,
) *wx.Column {
	return nil
}
