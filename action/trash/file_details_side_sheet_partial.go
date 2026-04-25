package trash

import (
	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type FileDetailsSideSheetPartialData struct {
	FileID string
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

func (qq *FileDetailsSideSheetPartial) Data(fileID string) *FileDetailsSideSheetPartialData {
	return &FileDetailsSideSheetPartialData{
		FileID: fileID,
	}
}

func (qq *FileDetailsSideSheetPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileDetailsSideSheetPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[FileTabsPartialState](rw, req)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data, state),
	)
}

func (qq *FileDetailsSideSheetPartial) Widget(
	ctx ctxx.Context,
	data *FileDetailsSideSheetPartialData,
	state *FileTabsPartialState,
) *widget.Dialog {
	return &widget.Dialog{
		Widget: widget.Widget[widget.Dialog]{
			ID: qq.ID(),
		},
		Headline:                        widget.T("Details"),
		IsOpenOnLoadOnExtraLargeScreens: true,
		KeepInDOMOnClose:                true,
		IsOpenOnLoad:                    state.ActiveSideSheet == qq.ID(),
		Layout:                          widget.DialogLayoutSideSheet,
		Child:                           qq.actions.FileTabsPartial.Widget(ctx, state, data.FileID),
	}
}

func (qq *FileDetailsSideSheetPartial) ID() string {
	return "trashFileDetailsSideSheet"
}
