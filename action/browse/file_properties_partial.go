package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FilePropertiesPartialData struct {
	FileID string
}

type FilePropertiesPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFilePropertiesPartial(infra *common.Infra, actions *Actions) *FilePropertiesPartial {
	config := actionx.NewConfig(
		actions.Route("file-properties"),
		true,
	)
	return &FilePropertiesPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FilePropertiesPartial) Data(fileID string) *FilePropertiesPartialData {
	return &FilePropertiesPartialData{
		FileID: fileID,
	}
}

func (qq *FilePropertiesPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FilePropertiesPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FilePropertiesPartial) Widget(ctx ctxx.Context, data *FilePropertiesPartialData) *wx.ScrollableContent {
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

func (qq *FilePropertiesPartial) ID() string {
	return "fileProperties"
}
