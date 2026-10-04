package browse

import (
	"html/template"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

// tab IDs must match the IDs TabBar derives from the untranslated tab labels
const (
	filterTabDocumentType = "document-type"
	filterTabFields       = "fields"
	filterTabTags         = "tags"
)

type FilterTabsPartialData struct {
	CurrentDirID string
	ActiveTab    string
}

type FilterTabsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFilterTabsPartial(infra *common.Infra, actions *Actions) *FilterTabsPartial {
	return &FilterTabsPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("filter-tabs-partial"),
			true,
		),
	}
}

func (qq *FilterTabsPartial) Data(currentDirID, activeTab string) *FilterTabsPartialData {
	return &FilterTabsPartialData{
		CurrentDirID: currentDirID,
		ActiveTab:    activeTab,
	}
}

func (qq *FilterTabsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FilterTabsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirPartialState](rw, req)

	if tabID, isBadge := strings.CutPrefix(req.Header.Get("HX-Target"), qq.badgeIDPrefix()); isBadge {
		return qq.infra.Renderer().Render(rw, ctx, qq.badge(state, data.CurrentDirID, tabID))
	}

	state.FilterTab = data.ActiveTab

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, state, data.CurrentDirID),
	)
}

func (qq *FilterTabsPartial) Widget(
	ctx ctxx.Context,
	state *ListDirPartialState,
	currentDirID string,
) *widget.TabBar {
	activeTab := state.FilterTab
	var activeTabContent renderable.Renderable

	switch activeTab {
	case filterTabFields:
		activeTabContent = qq.actions.ListFilterPropertiesPartial.Widget(
			ctx,
			qq.actions.ListFilterPropertiesPartial.Data(currentDirID, 0),
			state,
		)
	case filterTabTags:
		activeTabContent = qq.actions.ListFilterTagsPartial.Widget(
			ctx,
			currentDirID,
			state.CheckedTagIDs,
		)
	default:
		activeTab = filterTabDocumentType
		activeTabContent = qq.actions.DocumentTypeFilterPartial.Widget(
			ctx,
			qq.actions.DocumentTypeFilterPartial.Data(currentDirID),
			state,
		)
	}

	return &widget.TabBar{
		Widget: widget.Widget[widget.TabBar]{
			ID: qq.ID(),
		},
		ActiveTab: activeTab,
		Tabs: []*widget.Tab{
			qq.tab(widget.T("Document type"), state, currentDirID, filterTabDocumentType),
			qq.tab(widget.T("Fields"), state, currentDirID, filterTabFields),
			qq.tab(widget.T("Tags"), state, currentDirID, filterTabTags),
		},
		ActiveTabContent: &widget.ScrollableContent{
			MarginY:  true,
			Children: activeTabContent,
		},
	}
}

func (qq *FilterTabsPartial) tab(
	label *widget.Text,
	state *ListDirPartialState,
	currentDirID string,
	tabID string,
) *widget.Tab {
	return &widget.Tab{
		Label: label,
		Badge: qq.badge(state, currentDirID, tabID),
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(qq.Data(currentDirID, tabID)),
			HxTarget: "#" + qq.ID(),
			HxSwap:   "outerHTML",
			// kept client-side to preserve the current path, for example a selected file;
			// tabID is one of the constants above and thus safe to embed
			HxOn: &widget.HxOn{
				Event:   "click",
				Handler: template.JS("_setQueryParamValue('filter_tab', '" + tabID + "')"),
			},
		},
		IncreasedHeight: true,
	}
}

// badge indicates the active filters of a tab; it refreshes itself instead of the whole
// tab bar to preserve the tab content, for example focused field filter inputs
func (qq *FilterTabsPartial) badge(
	state *ListDirPartialState,
	currentDirID string,
	tabID string,
) *widget.Badge {
	var count int
	var isSmall bool
	var changedEvent event.Event
	switch tabID {
	case filterTabFields:
		count = len(state.PropertyValues)
		changedEvent = event.PropertyFilterChanged
	case filterTabTags:
		count = len(state.CheckedTagIDs)
		changedEvent = event.FilterTagsChanged
	default:
		tabID = filterTabDocumentType
		// only one document type can be selected, so a count would carry no information
		isSmall = true
		if state.DocumentTypeID != 0 {
			count = 1
		}
		changedEvent = event.DocumentTypeFilterChanged
	}

	id := qq.badgeIDPrefix() + tabID
	return &widget.Badge{
		Widget: widget.Widget[widget.Badge]{
			ID: id,
		},
		Value:          count,
		IsInline:       true,
		IsSmall:        isSmall,
		IsHiddenIfZero: true,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:    qq.Endpoint(),
			HxVals:    util.JSON(qq.Data(currentDirID, "")),
			HxTarget:  "#" + id,
			HxSwap:    "outerHTML",
			HxTrigger: event.HxTrigger(changedEvent),
		},
	}
}

func (qq *FilterTabsPartial) badgeIDPrefix() string {
	return "filterTabBadge-"
}

func (qq *FilterTabsPartial) ID() string {
	return "filterTabs"
}
