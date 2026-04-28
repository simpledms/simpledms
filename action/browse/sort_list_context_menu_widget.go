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
				IsSelected:     state.SortBy == "newestFirst",
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
				IsSelected:     state.SortBy == "name" || state.SortBy == "",
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
