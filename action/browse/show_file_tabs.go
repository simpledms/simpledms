package browse

// package action

import (
	"log"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ShowFileTabsData struct {
	CurrentDirID string
	FileID       string
	ActiveTab    string
}

// TODO rename to ShowFileTabs?
type ShowFileTabs struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewShowFileTabs(infra *common.Infra, actions *Actions) *ShowFileTabs {
	return &ShowFileTabs{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("show-file-tabs"),
			true,
		),
	}
}

func (qq *ShowFileTabs) Data(currentDirID, fileID, activeTab string) *ShowFileTabsData {
	return &ShowFileTabsData{
		CurrentDirID: currentDirID,
		FileID:       fileID,
		ActiveTab:    activeTab,
	}
}

func (qq *ShowFileTabs) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ShowFileTabsData](rw, req, ctx)
	if err != nil {
		return err
	}

	state := autil.StateX[FilePreviewState](rw, req)
	state.ActiveTab = data.ActiveTab

	rw.Header().Set("HX-Push-Url", route.BrowseFileWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, data.CurrentDirID, data.FileID))

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, state, data.CurrentDirID, data.FileID),
	)
}

// TODO enum instead of tabName
//
// nullableFile can be provided used if already loaded
// not necessary for all tabs, thus optional
func (qq *ShowFileTabs) Widget(
	ctx ctxx.Context,
	state *FilePreviewState,
	dirID string,
	fileID string,
) *wx.TabBar {
	var activeTabContent *wx.ScrollableContent

	activeTab := strings.ToLower(state.ActiveTab)

	switch activeTab {
	case "metadata", "":
		activeTabContent = qq.actions.FileAttributes.Widget(
			ctx,
			qq.actions.FileAttributes.Data(fileID),
		)
	case "properties":
		// TODO
		activeTabContent = qq.actions.FileProperties.Widget(
			ctx,
			qq.actions.FileProperties.Data(fileID),
		)
	case "tags":
		activeTabContent = qq.actions.Tagging.AssignedTags.List.ListView(
			ctx,
			qq.actions.Tagging.AssignedTags.List.Data(fileID),
		)
	case "info":
		// TODO
		activeTabContent = qq.actions.FileInfo.Widget(
			ctx,
			qq.actions.FileInfo.Data(fileID),
		)
	case "versions":
		// TODO
		activeTabContent = qq.actions.FileVersions.Widget(
			ctx,
			qq.actions.FileVersions.Data(fileID),
		)
	default:
		log.Println("tab name not supported")
		// FIXME fatal error or just continue?
		// 		raise BadRequest error?
		panic("Tab name not supported.") // log.Fatalln is not recoverable
		// return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Tab name not supported.")
	}

	tabsID := autil.GenerateID("showFileTabs")
	return &wx.TabBar{
		Widget: wx.Widget[wx.TabBar]{
			ID: tabsID,
		},
		ActiveTab: activeTab,
		IsFlowing: true,
		Tabs: []*wx.Tab{
			{
				Label: wx.T("Metadata"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxVals:   util.JSON(qq.Data(dirID, fileID, "metadata")),
					HxTarget: "#" + tabsID,
					HxSwap:   "outerHTML",
				},
				IncreasedHeight: true,
			},
			{
				Label: wx.T("Tags"),
				Badge: qq.actions.Tagging.AssignedTags.Count.Badge(ctx, fileID),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxVals:   util.JSON(qq.Data(dirID, fileID, "tags")),
					HxTarget: "#" + tabsID,
					HxSwap:   "outerHTML",
				},
				IncreasedHeight: true,
			},
			{
				Label: wx.T("Fields"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxVals:   util.JSON(qq.Data(dirID, fileID, "properties")),
					HxTarget: "#" + tabsID,
					HxSwap:   "outerHTML",
				},
				IncreasedHeight: true,
			},
			{
				Label: wx.T("Info"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxVals:   util.JSON(qq.Data(dirID, fileID, "info")),
					HxTarget: "#" + tabsID,
					HxSwap:   "outerHTML",
				},
				IncreasedHeight: true,
			},
			{
				Label: wx.T("Versions"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxVals:   util.JSON(qq.Data(dirID, fileID, "versions")),
					HxTarget: "#" + tabsID,
					HxSwap:   "outerHTML",
				},
				IncreasedHeight: true,
			},
		},
		ActiveTabContent: activeTabContent,
	}
}
