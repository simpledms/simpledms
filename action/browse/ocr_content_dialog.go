package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

// OCRContentDialogData identifies the file whose OCR content is shown.
type OCRContentDialogData struct {
	FileID string
}

// OCRContentDialog renders raw OCR content in a read-only dialog.
type OCRContentDialog struct {
	infra *common.Infra
	*actionx.Config
}

// NewOCRContentDialog creates the OCR content dialog action.
func NewOCRContentDialog(infra *common.Infra, actions *Actions) *OCRContentDialog {
	return &OCRContentDialog{
		infra:  infra,
		Config: actionx.NewConfig(actions.Route("ocr-content-dialog"), true),
	}
}

// Data returns request data for the given file.
func (qq *OCRContentDialog) Data(fileID string) *OCRContentDialogData {
	return &OCRContentDialogData{
		FileID: fileID,
	}
}

// ModalLinkAttrs returns attributes that load the dialog for the given file.
func (qq *OCRContentDialog) ModalLinkAttrs(fileID string) widget.HTMXAttrs {
	return widget.HTMXAttrs{
		HxPost:        qq.Endpoint(),
		HxVals:        util.JSON(qq.Data(fileID)),
		LoadInPopover: true,
	}
}

// Handler renders the requested file's OCR content.
func (qq *OCRContentDialog) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[OCRContentDialogData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetWithDeletedX(ctx, data.FileID)
	return qq.infra.Renderer().Render(
		rw,
		ctx,
		&widget.Dialog{
			Layout:       widget.DialogLayoutStable,
			Width:        widget.DialogWidthWide,
			Headline:     widget.Tu("OCR"),
			IsOpenOnLoad: true,
			Child: &widget.TextArea{
				Value:      filex.Data.OcrContent,
				Rows:       20,
				StyleType:  widget.TextAreaStyleTypeFullHeight,
				IsReadonly: true,
			},
		},
	)
}
