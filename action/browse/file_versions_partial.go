package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileVersionsPartialData struct {
	FileID string
}

type FileVersionsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileVersionsPartial(infra *common.Infra, actions *Actions) *FileVersionsPartial {
	config := actionx.NewConfig(
		actions.Route("file-versions"),
		true,
	)
	return &FileVersionsPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileVersionsPartial) Data(fileID string) *FileVersionsPartialData {
	return &FileVersionsPartialData{
		FileID: fileID,
	}
}

func (qq *FileVersionsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FileVersionsPartial) Widget(ctx ctxx.Context, data *FileVersionsPartialData) *wx.ScrollableContent {
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

func (qq *FileVersionsPartial) ID() string {
	return "fileVersions"
}
