package browse

// package action

import (
	"context"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type SelectDirCmdData struct {
	CurrentDirID int64
}

type SelectDirCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSelectDirCmd(infra *common.Infra, actions *Actions) *SelectDirCmd {
	return &SelectDirCmd{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("select-dir-cmd"),
			true, // TODO is this correct in the context it is used?
		),
	}
}

func (qq *SelectDirCmd) Data(currentDirID int64) *SelectDirCmdData {
	return &SelectDirCmdData{
		CurrentDirID: currentDirID,
	}
}

func (qq *SelectDirCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	_, err := autil.FormData[SelectDirCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx),
	)
}

func (qq *SelectDirCmd) Widget(ctx context.Context) *wx.List {
	return &wx.List{}
}
