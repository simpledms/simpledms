package browse

import (
	"log"
	"net/http"
	"net/url"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type FilePreviewStatusPartialData struct {
	CurrentDirID  string
	FileID        string
	VersionNumber string
	PreviewTab    string
	PushURL       bool
}

type FilePreviewStatusPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFilePreviewStatusPartial(infra *common.Infra, actions *Actions) *FilePreviewStatusPartial {
	return &FilePreviewStatusPartial{
		infra:   infra,
		actions: actions,
		Config:  actionx.NewConfig(actions.Route("file-preview-status-partial"), true),
	}
}

func (qq *FilePreviewStatusPartial) Data(
	currentDirID string,
	fileID string,
	versionNumber string,
	previewTab string,
) *FilePreviewStatusPartialData {
	return &FilePreviewStatusPartialData{
		CurrentDirID:  currentDirID,
		FileID:        fileID,
		VersionNumber: versionNumber,
		PreviewTab:    previewTab,
	}
}

func (qq *FilePreviewStatusPartial) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[FilePreviewStatusPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	activeTab := data.PreviewTab
	if data.CurrentDirID != "" {
		state := autil.StateX[FilePreviewPartialState](rw, req)
		if activeTab == "" {
			activeTab = state.PreviewTab
		}
		state.PreviewTab = activeTab
		rw.Header().Set("HX-Push-Url", route.BrowseFileWithState(state)(
			ctx.TenantCtx().TenantID,
			ctx.SpaceCtx().SpaceID,
			data.CurrentDirID,
			data.FileID,
		))
	} else if data.PushURL {
		currentURLValue := req.Header.Get("HX-Current-URL")
		if currentURLValue != "" {
			currentURL, err := url.Parse(currentURLValue)
			if err != nil {
				log.Println(err)
			} else {
				query := currentURL.Query()
				query.Set("preview_tab", activeTab)
				currentURL.RawQuery = query.Encode()
				currentURL.Scheme = ""
				currentURL.Host = ""
				rw.Header().Set("HX-Push-Url", currentURL.String())
			}
		}
	}
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	source, err := qq.actions.FilePreviewPartial.versionSource(ctx, filex, data.VersionNumber)
	if err != nil {
		log.Println(err)
		return e.NewHTTPErrorf(http.StatusNotFound, "version not found")
	}
	view, _, err := qq.actions.FilePreviewPartial.PreviewWidget(
		ctx,
		filex,
		source,
		data.VersionNumber,
		data.CurrentDirID,
		activeTab,
	)
	if err != nil {
		return err
	}
	qq.infra.Renderer().RenderX(rw, ctx, view)
	return nil
}
