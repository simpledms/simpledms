package browse

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ShowFileInfoPartialData struct {
	CurrentPath string
	Filename    string
}

type ShowFileInfoPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewShowFileInfoPartial(infra *common.Infra, actions *Actions) *ShowFileInfoPartial {
	return &ShowFileInfoPartial{
		infra,
		actions,
		actionx.NewConfig(
			actions.Route("show-file-info"),
			true,
		),
	}
}

func (qq *ShowFileInfoPartial) Data(currentPath, filename string) *ShowFileInfoPartialData {
	return &ShowFileInfoPartialData{
		CurrentPath: currentPath,
		Filename:    filename,
	}
}

func (qq *ShowFileInfoPartial) Widget() *wx.Column {
	return &wx.Column{
		Children: []wx.IWidget{
			wx.T("File Name"),
			wx.T("Size"),
			/*
				Text{"Name:"},
				Text{"Size:"},
				Text{"Created At:"},
				Text{"Modified At:"},
				Text{"Permissions:"},
				Text{"Owner:"},
				Text{"Group:"},
				Text{"Links:"},
				Text{"SHA256:"},
				Text{Label: "Content:"}, */
		},
	}
}

func (qq *ShowFileInfoPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	qq.infra.Renderer().RenderX(rw, ctx, qq.Widget())
	return nil
}
