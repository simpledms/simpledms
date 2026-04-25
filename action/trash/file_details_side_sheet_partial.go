package trash

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
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
