package inbox

import (
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type SortListContextMenuWidget struct {
	actions *Actions
}

func NewSortListContextMenuWidget(actions *Actions) *SortListContextMenuWidget {
	return &SortListContextMenuWidget{
		actions: actions,
	}
}

func (qq *SortListContextMenuWidget) Widget(ctx ctxx.Context, state *FilesListPartialState) *widget.Menu {
	return &widget.Menu{
		Widget: widget.Widget[widget.Menu]{
			ID: "sortBy",
		},
		Position: widget.PositionLeft,
		Items: []*widget.MenuItem{
			{
				Label:          widget.T("Newest first"),
				RadioGroupName: "SortBy",
				RadioValue:     "newestFirst",
				IsSelected:     state.SortBy == "newestFirst" || state.SortBy == "",
				HTMXAttrs: widget.HTMXAttrs{
					HxOn: event.SortByUpdated.UnsafeHxOnWithQueryParamAndValue(
						"click",
						"sort_by",
						"newestFirst",
					),
				},
			},
			{
				Label:          widget.T("Oldest first"),
				RadioGroupName: "SortBy",
				RadioValue:     "oldestFirst",
				IsSelected:     state.SortBy == "oldestFirst",
				HTMXAttrs: widget.HTMXAttrs{
					HxOn: event.SortByUpdated.UnsafeHxOnWithQueryParamAndValue(
						"click",
						"sort_by",
						"oldestFirst",
					),
				},
			},
			{
				Label:          widget.T("Sort by name"),
				RadioGroupName: "SortBy",
				RadioValue:     "name",
				IsSelected:     state.SortBy == "name",
				HTMXAttrs: widget.HTMXAttrs{
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
