package inbox

import (
	"log"
	"net/http"
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/txx"
)

type ConsumeUploadsCmd struct {
	infra *common.Infra
	*actionx.Config
}

func NewConsumeUploadsCmd(infra *common.Infra) *ConsumeUploadsCmd {
	return &ConsumeUploadsCmd{
		infra: infra,
		Config: actionx.NewConfig(
			"/org/{tenant_id}/space/{space_id}/inbox/consume-uploads", false,
		).EnableManualTxManagement(),
	}
}

func (qq *ConsumeUploadsCmd) Handler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	data, err := autil.FormData[ConsumeUploadsCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	// Finish conversion even if the upload client disconnects. The ingestion service
	// claims each account-owned staged file and commits before returning.
	ctx = ctxx.WithoutCancel(ctx.SpaceCtx())
	tmpFiles, err := ctx.MainCtx().UnsafeMainDB().ReadOnlyConn.TemporaryFile.Query().Where(
		temporaryfile.OwnerID(ctx.MainCtx().Account.ID),
		temporaryfile.UploadToken(data.UploadToken),
		temporaryfile.ConvertedToStoredFileAtIsNil(),
		temporaryfile.ExpiresAtGT(time.Now()),
	).All(ctx)
	if err != nil {
		log.Println(err)
		return err
	}
	rootID, err := txx.WithTenantReadSpaceTx(ctx.SpaceCtx(),
		func(readCtx *ctxx.SpaceContext) (int64, error) {
			return readCtx.SpaceRootDir().ID, nil
		})
	if err != nil {
		return err
	}
	for _, tmpFile := range tmpFiles {
		if _, err := qq.infra.FileSystem().PreparePersistingTemporaryAccountFile(
			ctx, tmpFile, rootID, true,
		); err != nil {
			log.Println(err)
			return err
		}
	}
	// Context-selection exception: navigate only after all conversions have committed.
	destination := route.InboxRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID)
	if !ctx.VisitorCtx().IsHTMXRequest {
		http.Redirect(rw, req.Request, destination, http.StatusSeeOther)
		return nil
	}
	rw.Header().Set("HX-Location", destination)
	rw.Header().Set("HX-Trigger", event.InboxChanged.String())
	if len(tmpFiles) == 0 {
		rw.AddRenderables(widget.NewSnackbarf("No new files found."))
	} else {
		rw.AddRenderables(widget.NewSnackbarf("Files uploaded successfully."))
	}
	return nil
}
