package inbox

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type TransferFileDialog struct {
	*actionx.Config
	infra     *common.Infra
	actions   *Actions
	transfers *filemodel.InboxTransferService
}

func NewTransferFileDialog(infra *common.Infra, actions *Actions) *TransferFileDialog {
	return &TransferFileDialog{
		Config:    actionx.NewConfig(actions.Route("transfer-file-dialog"), true),
		infra:     infra,
		actions:   actions,
		transfers: filemodel.NewInboxTransferService(),
	}
}

func (qq *TransferFileDialog) Data(fileID string) *TransferFileDialogData {
	return &TransferFileDialogData{
		FileID: fileID,
	}
}

func (qq *TransferFileDialog) Handler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	data, err := autil.FormData[TransferFileDialogData](rw, req, ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	destinations, err := qq.transfers.Destinations(ctx, data.FileID)
	if err != nil {
		return err
	}
	dialog := &widget.Dialog{
		Widget: widget.Widget[widget.Dialog]{
			ID: "transferFileDialog",
		},
		Headline:     widget.T("Move to another Inbox"),
		IsOpenOnLoad: true,
	}
	if len(destinations) == 0 {
		dialog.Child = widget.NewBody(widget.BodyTypeMd,
			widget.T("No other Inbox is available. You need write access to another Space, "+
				"or its Inbox must accept transfers."))
		return qq.infra.Renderer().Render(rw, ctx, dialog)
	}
	options := []*widget.SelectOption{
		{
			Value: "",
			Label: widget.T("Choose an Inbox"),
		},
	}
	for _, destination := range destinations {
		options = append(options, &widget.SelectOption{
			Value: destination.PublicID.String(),
			Label: widget.Tu(destination.Name),
		})
	}
	form := &widget.Form{
		Widget: widget.Widget[widget.Form]{
			ID: "transferFileForm",
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.actions.TransferFileCmd.Endpoint(),
			HxVals: util.JSON(data),
			// Preserve the selected destination and message after a rejected request.
			HxSwap: "none",
		},
		Children: []widget.IWidget{
			&widget.SelectField{
				Widget: widget.Widget[widget.SelectField]{
					ID: "transferDestination",
				},
				Name:       "DestinationSpaceID",
				Label:      widget.T("Destination Inbox"),
				Options:    options,
				IsRequired: true,
			},
			widget.NewBody(widget.BodyTypeSm,
				widget.T("You can choose other Spaces in this organization where you have write access, "+
					"or whose Inboxes accept transfers.")),
			&widget.TextArea{
				Widget: widget.Widget[widget.TextArea]{
					ID: "transferMessage",
				},
				Name:      "Message",
				Label:     widget.T("Message (optional)"),
				Rows:      3,
				StyleType: widget.TextAreaStyleTypeCompact,
			},
			widget.NewBody(widget.BodyTypeSm,
				widget.T("Your message will be saved as a note with your name.")),
			widget.NewBody(widget.BodyTypeMd,
				widget.T("Moving clears the document type, tags, and custom fields. "+
					"Versions and notes are kept.")),
			widget.NewBody(widget.BodyTypeMd,
				widget.T("If you cannot open the destination Space, "+
					"you will lose access to this file after moving it.")),
		},
	}
	dialog.FormID = form.ID
	dialog.SubmitLabel = widget.T("Move")
	dialog.Child = form
	return qq.infra.Renderer().Render(rw, ctx, dialog)
}
