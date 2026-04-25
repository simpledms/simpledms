package inbox

// package action

import (
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/uix/route"
)

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

func (qq *FileTabsPartial) Data(fileID string, activeTab string) *FileTabsPartialData {
	return &FileTabsPartialData{
		FileID:    fileID,
		ActiveTab: activeTab,
	}
}

func (qq *FileTabsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileTabsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	state := autil.StateX[InboxPageState](rw, req)
	state.ActiveTab = data.ActiveTab

	rw.Header().Set("HX-Push-Url", route.InboxWithState(state)(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, data.FileID))

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, state, data.FileID, nil),
	)
}

func (qq *FileTabsPartial) Widget(
	ctx ctxx.Context,
	state *InboxPageState,
	fileID string,
	nullableFile *filemodel.File,
) *widget.TabBar {
	var activeTabContent *widget.ScrollableContent
	tabsID := autil.GenerateID("showFileTabs")
	activeTab := strings.ToLower(state.ActiveTab)

	if nullableFile == nil {
		nullableFile = qq.infra.FileRepo.GetX(ctx, fileID)
	}

	switch activeTab {
	case "", "metadata":
		activeTabContent = qq.actions.FileMetadataPartial.Widget(
			ctx,
			qq.actions.FileMetadataPartial.Data(fileID),
		)
	case "move":
		if ctx.SpaceCtx().Space.IsFolderMode {
			activeTabContent = &widget.ScrollableContent{
				MarginY: true,
				GapY:    true,
				Children: []widget.IWidget{
					&widget.ListItem{
						Headline: widget.T("Select destination manually"),
						Type:     widget.ListItemTypeHelper,
						HTMXAttrs: qq.actions.MoveFileCmd.ModalLinkAttrs(
							qq.actions.MoveFileCmd.Data(nullableFile.Data.PublicID.String(), ""),
							"#innerContent",
						), /*.SetHxHeaders(autil.QueryHeader(
							qq.actions.InboxPage.Endpoint(),
							qq.actions.InboxPage.Data(),
						)),*/
					},
					widget.H(widget.HeadingTypeTitleMd, widget.T("Suggestions based on filename")),
					qq.actions.ListInboxAssignmentSuggestionsPartial.Widget(ctx, nullableFile.Data.ID),
				},
			}
		}
	case "tags":
		activeTabContent = qq.actions.Browse.Tagging.AssignedTags.List.ListView(
			ctx,
			qq.actions.Browse.Tagging.AssignedTags.List.Data(fileID),
		)
	case "fields":
		// TODO
		activeTabContent = qq.actions.Browse.FilePropertiesPartial.Widget(
			ctx,
			qq.actions.Browse.FilePropertiesPartial.Data(fileID),
		)
	case "info":
		// TODO
		activeTabContent = qq.actions.Browse.FileInfoPartial.Widget(
			ctx,
			qq.actions.Browse.FileInfoPartial.Data(fileID),
		)
	}

	var tabs []*widget.Tab

	tabs = append(tabs, &widget.Tab{
		Label: widget.T("Metadata"),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(fileID, "")),
			HxTarget: "#" + tabsID,
			HxSwap:   "outerHTML",
		},
		IncreasedHeight: true,
	},
	)

	if ctx.SpaceCtx().Space.IsFolderMode {
		tabs = append(tabs, &widget.Tab{
			Label: widget.T("Move"),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxVals:   util.JSON(qq.Data(fileID, "move")),
				HxTarget: "#" + tabsID,
				HxSwap:   "outerHTML",
			},
			IncreasedHeight: true,
		})
	}

	tabs = append(tabs, []*widget.Tab{
		{
			Label: widget.T("Tags"),
			Badge: qq.actions.Browse.Tagging.AssignedTags.Count.Badge(ctx, fileID),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxVals:   util.JSON(qq.Data(fileID, "tags")),
				HxTarget: "#" + tabsID,
				HxSwap:   "outerHTML",
			},
			IncreasedHeight: true,
		},
		{
			Label: widget.T("Fields"),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxVals:   util.JSON(qq.Data(fileID, "fields")),
				HxTarget: "#" + tabsID,
				HxSwap:   "outerHTML",
			},
			IncreasedHeight: true,
		},
		{
			Label: widget.T("Info"),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:   qq.Endpoint(),
				HxVals:   util.JSON(qq.Data(fileID, "info")),
				HxTarget: "#" + tabsID,
				HxSwap:   "outerHTML",
			},
			IncreasedHeight: true,
		},
	}...)

	return &widget.TabBar{
		Widget: widget.Widget[widget.TabBar]{
			ID: tabsID,
		},
		ActiveTab:        activeTab,
		IsFlowing:        true,
		Tabs:             tabs,
		ActiveTabContent: activeTabContent,
	}
}
