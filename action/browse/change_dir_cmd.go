package browse

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
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

func (qq *ChangeDirCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
