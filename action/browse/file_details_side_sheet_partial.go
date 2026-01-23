package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileDetailsSideSheetPartialData struct {
	CurrentDirID string
	FileID       string
}

type FileDetailsSideSheetPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileDetailsSideSheetPartial(infra *common.Infra, actions *Actions) *FileDetailsSideSheetPartial {
	config := actionx.NewConfig(actions.Route("file-details-side-sheet-partial"), true)
	return &FileDetailsSideSheetPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileDetailsSideSheetPartial) Data(currentDirID string, fileID string) *FileDetailsSideSheetPartialData {
	return &FileDetailsSideSheetPartialData{
		CurrentDirID: currentDirID,
		FileID:       fileID,
	}
}

func (qq *FileDetailsSideSheetPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileDetailsSideSheetPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[FilePreviewPartialState](rw, req)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data, state),
	)
}

func (qq *FileDetailsSideSheetPartial) Widget(
	ctx ctxx.Context,
	data *FileDetailsSideSheetPartialData,
	state *FilePreviewPartialState,
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
		Child: qq.actions.FileTabsPartial.Widget(
			ctx,
			state,
			data.CurrentDirID,
			data.FileID,
		),
	}

}

func (qq *FileDetailsSideSheetPartial) ID() string {
	return "fileDetailsSideSheet"
}
