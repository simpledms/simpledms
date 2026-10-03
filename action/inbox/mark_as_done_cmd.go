package inbox

// package action

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	filingmodel "github.com/simpledms/simpledms/model/tenant/filing"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type MarkAsDoneCmdData struct {
	FileID string
}

type MarkAsDoneCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	// inboxDir   *ent.File
	// storageDir *ent.File
}

func NewMarkAsDoneCmd(infra *common.Infra, actions *Actions) *MarkAsDoneCmd {
	config := actionx.NewConfig(
		actions.Route("mark-as-done-cmd"),
		false,
	).EnableCommittedResponse()
	return &MarkAsDoneCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		// inboxDir:   inboxDir,
		// storageDir: storageDir,
	}
}

func (qq *MarkAsDoneCmd) Data(fileID string) *MarkAsDoneCmdData {
	return &MarkAsDoneCmdData{
		FileID: fileID,
	}
}

func (qq *MarkAsDoneCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[MarkAsDoneCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	fileData, err := filingmodel.NewFilingService(qq.infra.FileSystem()).CompleteInboxFile(
		ctx, data.FileID,
	)
	if err != nil {
		log.Println(err)
		return err
	}
	filex := qq.infra.FileRepo.GetWithParentX(ctx, fileData.PublicID.String())
	// assignment = assignment.Update().SetIsInInbox(false).SaveX(ctx)

	// FIXME not correct, should not be opened in parent dir, but flat
	action := &widget.Link{
		Href:  route.BrowseFile(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, filex.Parent(ctx).Data.PublicID.String(), filex.Data.PublicID.String()),
		Child: widget.T("Open file"),
	}

	rw.AddRenderables(
		widget.NewSnackbarf("Marked file «%s» as done.", filex.Data.Name).WithAction(action),
	)
	rw.Header().Set("HX-Trigger", event.InboxChanged.String())

	return nil
}
