package browse

import (
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ShowFileInfoData struct {
	CurrentPath string
	Filename    string
}

type ShowFileInfo struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewShowFileInfo(infra *common.Infra, actions *Actions) *ShowFileInfo {
	return &ShowFileInfo{
		infra,
		actions,
		actionx.NewConfig(
			actions.Route("show-file-info"),
			true,
		),
	}
}

func (qq *ShowFileInfo) Data(currentPath, filename string) *ShowFileInfoData {
	return &ShowFileInfoData{
		CurrentPath: currentPath,
		Filename:    filename,
	}
}

func (qq *ShowFileInfo) Widget() *wx.Column {
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

func (qq *ShowFileInfo) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	qq.infra.Renderer().RenderX(rw, ctx, qq.Widget())
	return nil
}
