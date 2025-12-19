package browse

import (
	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileDetailsSideSheetData struct {
	CurrentDirID string
	FileID       string
}

type FileDetailsSideSheet struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileDetailsSideSheet(infra *common.Infra, actions *Actions) *FileDetailsSideSheet {
	config := actionx.NewConfig(actions.Route("file-details-side-sheet"), true)
	return &FileDetailsSideSheet{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileDetailsSideSheet) Data(currentDirID string, fileID string) *FileDetailsSideSheetData {
	return &FileDetailsSideSheetData{
		CurrentDirID: currentDirID,
		FileID:       fileID,
	}
}

func (qq *FileDetailsSideSheet) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileDetailsSideSheetData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[FilePreviewState](rw, req)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data, state),
	)
}

func (qq *FileDetailsSideSheet) Widget(
	ctx ctxx.Context,
	data *FileDetailsSideSheetData,
	state *FilePreviewState,
) *wx.Dialog {
	// if listDirState.OpenDialog == qq.ID() {
	// TODO remove state from URL
	// return &wx.View{}
	// }

	return &wx.Dialog{
		Widget: wx.Widget[wx.Dialog]{
			ID: qq.ID(),
		},
		Headline:                        wx.T("Details"),
		IsOpenOnLoadOnExtraLargeScreens: true,
		// allows for quick back and forth on mobile devices
		KeepInDOMOnClose: true,
		IsOpenOnLoad:     state.ActiveSideSheet == qq.ID(),
		Layout:           wx.DialogLayoutSideSheet,
		Child: qq.actions.ShowFileTabs.Widget(
			ctx,
			state,
			data.CurrentDirID,
			data.FileID,
		),
	}

}

func (qq *FileDetailsSideSheet) ID() string {
	return "fileDetailsSideSheet"
}
