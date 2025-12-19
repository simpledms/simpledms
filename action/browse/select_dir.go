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

type SelectDirData struct {
	CurrentDirID int64
}

type SelectDir struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSelectDir(infra *common.Infra, actions *Actions) *SelectDir {
	return &SelectDir{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("select-dir"),
			true, // TODO is this correct in the context it is used?
		),
	}
}

func (qq *SelectDir) Data(currentDirID int64) *SelectDirData {
	return &SelectDirData{
		CurrentDirID: currentDirID,
	}
}

func (qq *SelectDir) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	_, err := autil.FormData[SelectDirData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx),
	)
}

func (qq *SelectDir) Widget(ctx context.Context) *wx.List {
	return &wx.List{}
}
