package inbox

// package action

import (
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ShowFileTabsData struct {
	FileID    string
	ActiveTab string
}

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

func (qq *ShowFileTabs) Data(fileID string, activeTab string) *ShowFileTabsData {
	return &ShowFileTabsData{
		FileID:    fileID,
		ActiveTab: activeTab,
	}
}

func (qq *ShowFileTabs) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ShowFileTabsData](rw, req, ctx)
	if err != nil {
		return err
	}

	state := autil.StateX[PageState](rw, req)
	state.ActiveTab = data.ActiveTab

	rw.Header().Set("HX-Push-Url", route.InboxWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, data.FileID))

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, state, data.FileID, nil),
	)
}

func (qq *ShowFileTabs) Widget(
	ctx ctxx.Context,
	state *PageState,
	fileID string,
	nullableFile *model.File,
) *wx.TabBar {
	var activeTabContent *wx.ScrollableContent
	tabsID := autil.GenerateID("showFileTabs")
	activeTab := strings.ToLower(state.ActiveTab)

	if nullableFile == nil {
		nullableFile = qq.infra.FileRepo.GetX(ctx, fileID)
	}

	switch activeTab {
	case "", "metadata":
		activeTabContent = qq.actions.FileMetadata.Widget(
			ctx,
			qq.actions.FileMetadata.Data(fileID),
		)
	case "move":
		if ctx.SpaceCtx().Space.IsFolderMode {
			activeTabContent = &wx.ScrollableContent{
				MarginY: true,
				GapY:    true,
				Children: []wx.IWidget{
					&wx.ListItem{
						Headline: wx.T("Select destination manually"),
						Type:     wx.ListItemTypeHelper,
						HTMXAttrs: qq.actions.MoveFile.ModalLinkAttrs(
							qq.actions.MoveFile.Data(nullableFile.Data.PublicID.String(), ""),
							"#innerContent",
						).SetHxHeaders(autil.QueryHeader(
							qq.actions.Page.Endpoint(),
							qq.actions.Page.Data(),
						)),
					},
					wx.H(wx.HeadingTypeTitleMd, wx.T("Suggestions based on filename")),
					qq.actions.ListInboxAssignmentSuggestions.Widget(ctx, nullableFile.Data.ID),
				},
			}
		}
	case "tags":
		activeTabContent = qq.actions.Browse.Tagging.AssignedTags.List.ListView(
			ctx,
			qq.actions.Browse.Tagging.AssignedTags.List.Data(fileID),
		)
	case "properties":
		// TODO
		activeTabContent = qq.actions.Browse.FileProperties.Widget(
			ctx,
			qq.actions.Browse.FileProperties.Data(fileID),
		)
	case "info":
		// TODO
		activeTabContent = qq.actions.Browse.FileInfo.Widget(
			ctx,
			qq.actions.Browse.FileInfo.Data(fileID),
		)
	}

	var tabs []*wx.Tab

	tabs = append(tabs, &wx.Tab{
		Label: wx.T("Metadata"),
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(fileID, "")),
			HxTarget: "#" + tabsID,
			HxSwap:   "outerHTML",
		},
		IncreasedHeight: true,
	},
	)

	if ctx.SpaceCtx().Space.IsFolderMode {
		tabs = append(tabs, &wx.Tab{
			Label: wx.T("Move"),
			HTMXAttrs: wx.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxVals:   util.JSON(qq.Data(fileID, "move")),
				HxTarget: "#" + tabsID,
				HxSwap:   "outerHTML",
			},
			IncreasedHeight: true,
		})
	}

	tabs = append(tabs, []*wx.Tab{
		{
			Label: wx.T("Tags"),
			Badge: qq.actions.Browse.Tagging.AssignedTags.Count.Badge(ctx, fileID),
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
				HxVals:   util.JSON(qq.Data(fileID, "properties")),
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
	}...)

	return &wx.TabBar{
		Widget: wx.Widget[wx.TabBar]{
			ID: tabsID,
		},
		ActiveTab:        activeTab,
		IsFlowing:        true,
		Tabs:             tabs,
		ActiveTabContent: activeTabContent,
	}
}
