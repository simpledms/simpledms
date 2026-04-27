package trash

import (
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

type FileTabsPartialState struct {
	ActiveTab       string `url:"tab,omitempty"`
	ActiveSideSheet string `url:"side_sheet,omitempty"`
}

type FileTabsPartialData struct {
	FileID    string
	ActiveTab string
}

type FileTabsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileTabsPartial(infra *common.Infra, actions *Actions) *FileTabsPartial {
	return &FileTabsPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("file-tabs-partial"),
			true,
		),
	}
}

func (qq *FileTabsPartial) Data(fileID, activeTab string) *FileTabsPartialData {
	return &FileTabsPartialData{
		FileID:    fileID,
		ActiveTab: activeTab,
	}
}

func (qq *FileTabsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileTabsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	state := autil.StateX[FileTabsPartialState](rw, req)
	state.ActiveTab = data.ActiveTab

	rw.Header().Set("HX-Push-Url", route.TrashFileWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, data.FileID))

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, state, data.FileID),
	)
}

func (qq *FileTabsPartial) Widget(
	ctx ctxx.Context,
	state *FileTabsPartialState,
	fileID string,
) *wx.TabBar {
	activeTab := strings.ToLower(state.ActiveTab)
	var activeTabContent wx.IWidget

	switch activeTab {
	case "metadata", "":
		activeTabContent = qq.actions.FileMetadataPartial.Widget(ctx, qq.actions.FileMetadataPartial.Data(fileID))
	case "tags":
		activeTabContent = qq.actions.FileTagsPartial.Widget(ctx, qq.actions.FileTagsPartial.Data(fileID))
	case "fields":
		activeTabContent = qq.actions.Browse.FilePropertiesPartial.Widget(
			ctx,
			qq.actions.Browse.FilePropertiesPartial.Data(fileID),
		)
	case "info":
		activeTabContent = qq.actions.FileInfoPartial.Widget(
			ctx,
			qq.actions.FileInfoPartial.Data(fileID),
		)
	case "versions":
		activeTabContent = qq.actions.Browse.FileVersionsPartial.Widget(
			ctx,
			qq.actions.Browse.FileVersionsPartial.Data(fileID),
		)
	default:
		activeTabContent = qq.actions.FileMetadataPartial.Widget(ctx, qq.actions.FileMetadataPartial.Data(fileID))
	}

	tabsID := autil.GenerateID("trashFileTabs")
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
					HxVals:   util.JSON(qq.Data(fileID, "metadata")),
					HxTarget: "#" + tabsID,
					HxSwap:   "outerHTML",
				},
				IncreasedHeight: true,
			},
			{
				Label: wx.T("Tags"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxVals:   util.JSON(qq.Data(fileID, "tags")),
					HxTarget: "#" + tabsID,
					HxSwap:   "outerHTML",
				},
				IncreasedHeight: true,
			},
			{
				Label: wx.T("Fields"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxVals:   util.JSON(qq.Data(fileID, "fields")),
					HxTarget: "#" + tabsID,
					HxSwap:   "outerHTML",
				},
				IncreasedHeight: true,
			},
			{
				Label: wx.T("Info"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxVals:   util.JSON(qq.Data(fileID, "info")),
					HxTarget: "#" + tabsID,
					HxSwap:   "outerHTML",
				},
				IncreasedHeight: true,
			},
			{
				Label: wx.T("Versions"),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:   qq.Endpoint(),
					HxVals:   util.JSON(qq.Data(fileID, "versions")),
					HxTarget: "#" + tabsID,
					HxSwap:   "outerHTML",
				},
				IncreasedHeight: true,
			},
		},
		ActiveTabContent: activeTabContent,
	}
}
