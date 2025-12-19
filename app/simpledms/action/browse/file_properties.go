package browse

import (
	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FilePropertiesData struct {
	FileID string
}

type FileProperties struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileProperties(infra *common.Infra, actions *Actions) *FileProperties {
	config := actionx.NewConfig(
		actions.Route("file-properties"),
		true,
	)
	return &FileProperties{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileProperties) Data(fileID string) *FilePropertiesData {
	return &FilePropertiesData{
		FileID: fileID,
	}
}

func (qq *FileProperties) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FilePropertiesData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FileProperties) Widget(ctx ctxx.Context, data *FilePropertiesData) *wx.ScrollableContent {
	// filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.ID(),
		},
		GapY:     true,
		Children: wx.T("Coming soon."),
		MarginY:  true,
	}
}

func (qq *FileProperties) ID() string {
	return "fileProperties"
}
