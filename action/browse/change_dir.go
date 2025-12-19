package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ChangeDirData struct {
	DirID string
}

type ChangeDir struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewChangeDir(infra *common.Infra, actions *Actions) *ChangeDir {
	config := actionx.NewConfig(
		actions.Route("change-dir"),
		true,
	)
	return &ChangeDir{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ChangeDir) Data(dirID string) *ChangeDirData {
	return &ChangeDirData{
		DirID: dirID,
	}
}

func (qq *ChangeDir) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ChangeDirData](rw, req, ctx)
	if err != nil {
		return err
	}

	qq.infra.Renderer().RenderX(rw, ctx, qq.actions.ListDir.WidgetHandler(
		rw,
		req,
		ctx,
		data.DirID,
		"",
	))
	return nil
}
