package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ChangeDirCmdData struct {
	DirID string
}

type ChangeDirCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewChangeDirCmd(infra *common.Infra, actions *Actions) *ChangeDirCmd {
	config := actionx.NewConfig(
		actions.Route("change-dir-cmd"),
		true,
	)
	return &ChangeDirCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ChangeDirCmd) Data(dirID string) *ChangeDirCmdData {
	return &ChangeDirCmdData{
		DirID: dirID,
	}
}

func (qq *ChangeDirCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ChangeDirCmdData](rw, req, ctx)
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
