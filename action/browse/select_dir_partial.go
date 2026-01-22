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

type SelectDirPartialData struct {
	CurrentDirID int64
}

type SelectDirPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSelectDirPartial(infra *common.Infra, actions *Actions) *SelectDirPartial {
	return &SelectDirPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("select-dir"),
			true, // TODO is this correct in the context it is used?
		),
	}
}

func (qq *SelectDirPartial) Data(currentDirID int64) *SelectDirPartialData {
	return &SelectDirPartialData{
		CurrentDirID: currentDirID,
	}
}

func (qq *SelectDirPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	_, err := autil.FormData[SelectDirPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx),
	)
}

func (qq *SelectDirPartial) Widget(ctx context.Context) *wx.List {
	return &wx.List{}
}
