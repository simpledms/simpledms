package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileVersionsData struct {
	FileID string
}

type FileVersions struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileVersions(infra *common.Infra, actions *Actions) *FileVersions {
	config := actionx.NewConfig(
		actions.Route("file-versions"),
		true,
	)
	return &FileVersions{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileVersions) Data(fileID string) *FileVersionsData {
	return &FileVersionsData{
		FileID: fileID,
	}
}

func (qq *FileVersions) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionsData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FileVersions) Widget(ctx ctxx.Context, data *FileVersionsData) *wx.ScrollableContent {
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

func (qq *FileVersions) ID() string {
	return "fileVersions"
}
