package inbox

// package action

import (
	"log"
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/entmain/temporaryfile"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

// TODO necessary?
type PageData struct {
}

type PageState struct {
	UploadToken string `url:"upload_token,omitempty"`
	ListFilesState
	ShowFileState
}

// TODO rename to PageContent?
type Page struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewPage(infra *common.Infra, actions *Actions) *Page {
	config := actionx.NewConfig(
		actions.Route("page"),
		false,
	)
	return &Page{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *Page) Data() *PageData {
	return &PageData{}
}

// used in Query, for example MarkAsDone
// TODO refactor, legacy code
func (qq *Page) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[PageState](rw, req)

	selectedFileID := ""

	// duplicate in AssignFile and MoveFile
	// select next file in queue
	//
	// OnlyX or OnlyID doesn't work with Limit, returns error if multiple before Limit is applied
	files := qq.actions.ListFiles.filesQuery(ctx, state).Limit(1).AllX(ctx)
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
		qq.actions.Page.WidgetHandler(rw, req, ctx, selectedFileID),
	)
}

// TODO with and without selection together?
// TODO not nice that url params are already read from URL and passed in in addition to req, could be confusing
func (qq *Page) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	selectedFileID string,
) *wx.ListDetailLayout {
	// TODO handle selection
	// TODO use in MoveFile / AssignFile, initial render

	state := autil.StateX[PageState](rw, req)

	// in WidgetHandler because it used legacy Page style where WidgetHandler is called in page.Inbox
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
		newURL := route.InboxRootWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID)
		if selectedFileID != "" {
			newURL = route.InboxWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, selectedFileID)
		}
		rw.Header().Set("HX-Replace-Url", newURL)
	}

	// rw.Header().Set("HX-Retarget", "#innerContent")
	// rw.Header().Set("HX-Reswap", "innerHTML")

	return qq.Widget(ctx, state, selectedFileID)
}

func (qq *Page) Widget(ctx ctxx.Context, state *PageState, selectedFileID string) *wx.ListDetailLayout {
	listDetailLayout := qq.actions.ListFiles.Widget(
		ctx,
		state,
		selectedFileID,
	)

	if selectedFileID != "" {
		filex := qq.infra.FileRepo.GetX(ctx, selectedFileID)
		listDetailLayout.Detail = qq.actions.ShowFile.Widget(ctx, state, filex)
	}

	return listDetailLayout
}

func (qq *Page) processTemporaryFiles(rw httpx.ResponseWriter, ctx ctxx.Context, state *PageState) error {
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
