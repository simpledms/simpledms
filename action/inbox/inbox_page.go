package inbox

// package action

import (
	"log"
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain/temporaryfile"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

// TODO necessary?
type InboxPageData struct {
}

type InboxPageState struct {
	UploadToken string `url:"upload_token,omitempty"`
	FilesListPartialState
	FilePartialState
}

// TODO rename to PageContent?
type InboxPage struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewInboxPage(infra *common.Infra, actions *Actions) *InboxPage {
	config := actionx.NewConfig(
		actions.Route("inbox-page"),
		false,
	)
	return &InboxPage{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *InboxPage) Data() *InboxPageData {
	return &InboxPageData{}
}

// used in Query, for example MarkAsDoneCmd
// TODO refactor, legacy code
func (qq *InboxPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[InboxPageState](rw, req)

	selectedFileID := ""

	// duplicate in AssignFileCmd and MoveFileCmd
	// select next file in queue
	//
	// OnlyX or OnlyID doesn't work with Limit, returns error if multiple before Limit is applied
	files := qq.actions.ListFilesPartial.filesQuery(ctx, state).Limit(1).AllX(ctx)
	if len(files) == 0 {
		selectedFileID = ""
	} else {
		selectedFileID = files[0].PublicID.String()
	}

	// TODO necessary?
	rw.Header().Set("HX-Retarget", "#innerContent")
	rw.Header().Set("HX-Reswap", "innerHTML")

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.actions.InboxPage.WidgetHandler(rw, req, ctx, selectedFileID),
	)
}

// TODO with and without selection together?
// TODO not nice that url params are already read from URL and passed in in addition to req, could be confusing
func (qq *InboxPage) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	selectedFileID string,
) *wx.ListDetailLayout {
	// TODO handle selection
	// TODO use in MoveFileCmd / AssignFileCmd, initial render

	state := autil.StateX[InboxPageState](rw, req)

	// in WidgetHandler because it used legacy InboxPage style where WidgetHandler is called in page.Inbox
	if state.UploadToken != "" {
		// must be first before the data gets read
		err := qq.processTemporaryFiles(rw, ctx, state)
		if err != nil {
			panic(err)
		}

		state.UploadToken = ""
		// remove Upload Token from URL to prevent "now files found" message on reload
		// TODO doesn't work on errors; okay or not?
		rw.Header().Set("HX-Replace-Url", route.InboxRootWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID))

		// TODO select first file?
	} else {
		// TODO is this the correct place?
		// TODO why is this necessary?
		/* commented on 28.01.2026 because it kept side_sheet param in URL alive when switching
		from other pages to inbox
		newURL := route.InboxRootWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID)
		if selectedFileID != "" {
			newURL = route.InboxWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, selectedFileID)
		}*/
		// rw.Header().Set("HX-Replace-Url", newURL)
	}

	// rw.Header().Set("HX-Retarget", "#innerContent")
	// rw.Header().Set("HX-Reswap", "innerHTML")

	return qq.Widget(ctx, state, selectedFileID)
}

func (qq *InboxPage) Widget(ctx ctxx.Context, state *InboxPageState, selectedFileID string) *wx.ListDetailLayout {
	listDetailLayout := qq.actions.ListFilesPartial.Widget(
		ctx,
		state,
		selectedFileID,
	)

	if selectedFileID != "" {
		filex := qq.infra.FileRepo.GetX(ctx, selectedFileID)
		listDetailLayout.Detail = qq.actions.FilePartial.Widget(ctx, state, filex)
	}

	return listDetailLayout
}

func (qq *InboxPage) processTemporaryFiles(rw httpx.ResponseWriter, ctx ctxx.Context, state *InboxPageState) error {
	tmpFiles := ctx.MainCtx().Account.QueryTemporaryFiles().Where(
		temporaryfile.UploadToken(state.UploadToken),
		temporaryfile.ConvertedToStoredFileAtIsNil(),
		temporaryfile.ExpiresAtGT(time.Now()),
	).AllX(ctx)

	if len(tmpFiles) == 0 {
		rw.AddRenderables(wx.NewSnackbarf("No new files found."))
		return nil
	}

	for _, tmpFile := range tmpFiles {
		_, err := qq.infra.FileSystem().PreparePersistingTemporaryAccountFile(
			ctx,
			tmpFile,
			ctx.SpaceCtx().SpaceRootDir().ID,
			true,
		)
		if err != nil {
			log.Println("error saving tmp file", err)
			return err
		}
	}

	rw.AddRenderables(wx.NewSnackbarf("Files uploaded successfully."))
	return nil
}
