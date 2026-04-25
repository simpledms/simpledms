package browse

import (
	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
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

func (qq *FileDetailsSideSheetPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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
) *widget.Dialog {
	// if listDirState.OpenDialog == qq.ID() {
	// TODO remove state from URL
	// return &wx.View{}
	// }

	return &widget.Dialog{
		Widget: widget.Widget[widget.Dialog]{
			ID: qq.ID(),
		},
		Headline:                        widget.T("Details"),
		IsOpenOnLoadOnExtraLargeScreens: true,
		// allows for quick back and forth on mobile devices
		KeepInDOMOnClose: true,
		IsOpenOnLoad:     state.ActiveSideSheet == qq.ID(),
		Layout:           widget.DialogLayoutSideSheet,
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
