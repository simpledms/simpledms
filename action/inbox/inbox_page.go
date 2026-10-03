package inbox

// package action

import (
	"net/http"
	"net/url"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
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
		true,
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

// Lifecycle notifications query the next valid selection in the current filtered Inbox.
func (qq *InboxPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state, err := qq.prepareState(rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.render(rw, req, ctx, state)
}

func (qq *InboxPage) render(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	state *InboxPageState,
) error {

	selectedFileID := ""

	if current, err := url.Parse(req.Header.Get("HX-Current-URL")); err == nil {
		selectedFileID = strings.TrimPrefix(current.Path,
			route.InboxRoot(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID))
		if strings.Contains(selectedFileID, "/") {
			selectedFileID = ""
		}
	}
	if selectedFileID != "" && !qq.actions.ListFilesPartial.filesQuery(ctx, state).
		Where(file.PublicID(entx.NewCIText(selectedFileID))).ExistX(ctx) {
		selectedFileID = ""
		// OnlyX does not support Limit; query a slice to select the next result.
		files := qq.actions.ListFilesPartial.filesQuery(ctx, state).Limit(1).AllX(ctx)
		if len(files) > 0 {
			selectedFileID = files[0].PublicID.String()
		}
	}
	currentURL := route.InboxRootWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID)
	if selectedFileID != "" {
		currentURL = route.InboxWithState(state)(
			ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, selectedFileID,
		)
	}
	rw.Header().Set("HX-Replace-Url", currentURL)
	rw.Header().Set("HX-Retarget", "#innerContent")
	rw.Header().Set("HX-Reswap", "morph:innerHTML")
	view, err := qq.Widget(ctx, state, selectedFileID)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		view,
	)
}

// TODO with and without selection together?
// TODO not nice that url params are already read from URL and passed in in addition to req, could be confusing
func (qq *InboxPage) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	selectedFileID string,
) (*widget.ListDetailLayout, error) {
	// TODO handle selection
	// TODO use in MoveFileCmd / AssignFileCmd, initial render
	state, err := qq.prepareState(rw, req, ctx)
	if err != nil {
		return nil, err
	}
	return qq.Widget(ctx, state, selectedFileID)
}

// Staged uploads are consumed by ConsumeUploadsCmd, never while rendering a query.
func (qq *InboxPage) prepareState(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) (*InboxPageState, error) {
	state := autil.StateX[InboxPageState](rw, req)
	if err := req.ParseForm(); err != nil {
		return nil, err
	}
	if req.PostForm.Has("SearchQuery") {
		state.SearchQuery = req.PostForm.Get("SearchQuery")
	}
	if req.PostForm.Has("SourceValues") {
		state.SourceValues = req.PostForm["SourceValues"]
	}
	state.UploadToken = ""
	if _, err := state.sources(); err != nil {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid source filter.")
	}

	// TODO is this the correct place?
	// TODO why is this necessary?
	/* commented on 28.01.2026 because it kept side_sheet param in URL alive when switching
	from other pages to inbox
	newURL := route.InboxRootWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID)
	if selectedFileID != "" {
		newURL = route.InboxWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, selectedFileID)
	}*/
	// rw.Header().Set("HX-Replace-Url", newURL)

	// rw.Header().Set("HX-Retarget", "#innerContent")
	// rw.Header().Set("HX-Reswap", "innerHTML")

	return state, nil
}

func (qq *InboxPage) Widget(
	ctx ctxx.Context,
	state *InboxPageState,
	selectedFileID string,
) (*widget.ListDetailLayout, error) {
	listDetailLayout := qq.actions.ListFilesPartial.Widget(
		ctx,
		state,
		selectedFileID,
	)

	if selectedFileID != "" {
		filex := qq.infra.FileRepo.GetX(ctx, selectedFileID)
		detail, err := qq.actions.FilePartial.Widget(ctx, state, filex)
		if err != nil {
			return nil, err
		}
		listDetailLayout.Detail = detail
	}

	return listDetailLayout, nil
}
