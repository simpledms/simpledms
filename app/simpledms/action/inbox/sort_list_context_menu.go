package inbox

import (
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/event"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type SortListContextMenu struct {
	actions *Actions
}

func NewSortListContextMenu(actions *Actions) *SortListContextMenu {
	return &SortListContextMenu{
		actions: actions,
	}
}

func (qq *SortListContextMenu) Widget(ctx ctxx.Context, state *ListFilesState) *wx.Menu {
	return &wx.Menu{
		Widget: wx.Widget[wx.Menu]{
			ID: "sortBy",
		},
		Position: wx.PositionLeft,
		Items: []*wx.MenuItem{
			{
				Label:          wx.T("Newest first"),
				RadioGroupName: "SortBy",
				RadioValue:     "newestFirst",
				IsSelected:     state.SortBy == "newestFirst" || state.SortBy == "",
				HTMXAttrs: wx.HTMXAttrs{
					HxOn: event.SortByUpdated.UnsafeHxOnWithQueryParamAndValue(
						"click",
						"sort_by",
						"newestFirst",
					),
				},
			},
			{
				Label:          wx.T("Oldest first"),
				RadioGroupName: "SortBy",
				RadioValue:     "oldestFirst",
				IsSelected:     state.SortBy == "oldestFirst",
				HTMXAttrs: wx.HTMXAttrs{
					HxOn: event.SortByUpdated.UnsafeHxOnWithQueryParamAndValue(
						"click",
						"sort_by",
						"oldestFirst",
					),
				},
			},
			{
				Label:          wx.T("Sort by name"),
				RadioGroupName: "SortBy",
				RadioValue:     "name",
				IsSelected:     state.SortBy == "name",
				HTMXAttrs: wx.HTMXAttrs{
					HxOn: event.SortByUpdated.UnsafeHxOnWithQueryParamAndValue(
						"click",
						"sort_by",
						"name",
					),
				},
			},
		},
	}
}
