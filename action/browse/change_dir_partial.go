package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ChangeDirPartialData struct {
	DirID string
}

type ChangeDirPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewChangeDirPartial(infra *common.Infra, actions *Actions) *ChangeDirPartial {
	config := actionx.NewConfig(
		actions.Route("change-dir"),
		true,
	)
	return &ChangeDirPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ChangeDirPartial) Data(dirID string) *ChangeDirPartialData {
	return &ChangeDirPartialData{
		DirID: dirID,
	}
}

func (qq *ChangeDirPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ChangeDirPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	qq.infra.Renderer().RenderX(rw, ctx, qq.actions.ListDirPartial.WidgetHandler(
		rw,
		req,
		ctx,
		data.DirID,
		"",
	))
	return nil
}
