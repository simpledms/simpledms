package browse

import (
	"context"

	"github.com/marcobeierer/go-core/common"
	wx "github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/simpledms/simpledms/db/enttenant"
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
