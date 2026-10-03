package inbox

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/space"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type TransferFileCmd struct {
	*actionx.Config
	transfers *filemodel.InboxTransferService
}

func NewTransferFileCmd(actions *Actions) *TransferFileCmd {
	return &TransferFileCmd{
		Config: actionx.NewConfig(actions.Route("transfer-file-cmd"), false).
			EnableCommittedResponse(),
		transfers: filemodel.NewInboxTransferService(),
	}
}

func (qq *TransferFileCmd) Handler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	data, err := autil.FormData[TransferFileCmdData](rw, req, ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	destination, err := qq.transfers.Transfer(
		ctx, data.FileID, data.DestinationSpaceID, data.Message,
	)
	if err != nil {
		return err
	}
	canOpenDestination, err := ctx.TenantCtx().TTx.Space.Query().
		Where(space.ID(destination.ID)).Exist(ctx)
	if err != nil {
		log.Printf("check transferred file destination access: %v", err)
		return err
	}
	snackbar := widget.NewSnackbarf("Moved to the Inbox of «%s».", destination.Name)
	if canOpenDestination {
		snackbar.WithAction(&widget.Link{
			Href: route.Inbox(
				ctx.TenantCtx().TenantID, destination.PublicID.String(), data.FileID,
			),
			Child: widget.T("Open file"),
		})
	}
	rw.AddRenderables(snackbar)
	rw.Header().Set("HX-Trigger", event.FileMoved.String())
	return nil
}
