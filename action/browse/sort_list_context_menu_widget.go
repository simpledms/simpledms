package browse

import (
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type SortListContextMenuWidget struct{}

func NewSortListContextMenuWidget() *SortListContextMenuWidget {
	return &SortListContextMenuWidget{}
}

func (qq *SortListContextMenuWidget) Widget(ctx ctxx.Context, state *ListDirPartialState) *wx.Menu {
	var items []*wx.MenuItem
	if state.hasActiveSearch() {
		items = append(
			items,
			&wx.MenuItem{
				Label:          wx.T("Best match"),
				RadioGroupName: "SortBy",
				RadioValue:     sortByRank,
				IsSelected:     state.SortBy == sortByRank,
				HTMXAttrs: wx.HTMXAttrs{
					HxOn: event.SortByUpdated.UnsafeHxOnWithQueryParamAndValue(
						"click",
						"sort_by",
						sortByRank,
					),
				},
			},
			&wx.MenuItem{IsDivider: true},
		)
	}

	items = append(items,
		qq.sortMenuItem(wx.T("Newest first"), sortByNewestFirst, state.SortBy == sortByNewestFirst),
		qq.sortMenuItem(wx.T("Oldest first"), sortByOldestFirst, state.SortBy == sortByOldestFirst),
		qq.sortMenuItem(
			wx.T("Sort by name"),
			sortByName,
			state.SortBy == sortByName || state.SortBy == "",
		),
	)

	return &wx.Menu{
		Widget: wx.Widget[wx.Menu]{
			ID: "sortBy",
		},
		Position: wx.PositionLeft,
		Items:    items,
	}
}

func (qq *SortListContextMenuWidget) sortMenuItem(
	label *wx.Text,
	sortBy string,
	isSelected bool,
) *wx.MenuItem {
	return &wx.MenuItem{
		Label:          label,
		RadioGroupName: "SortBy",
		RadioValue:     sortBy,
		IsSelected:     isSelected,
		HTMXAttrs: wx.HTMXAttrs{
			HxOn: event.SortByUpdated.UnsafeHxOnWithQueryParamAndValue(
				"click",
				"sort_by",
				sortBy,
			),
		},
	}
}
