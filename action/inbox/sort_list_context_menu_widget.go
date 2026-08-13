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
	var items []*widget.MenuItem
	if state.hasActiveSearch() {
		items = append(
			items,
			&widget.MenuItem{
				Label:          widget.T("Best match"),
				RadioGroupName: "SortBy",
				RadioValue:     sortByRank,
				IsSelected:     state.SortBy == sortByRank,
				HTMXAttrs: widget.HTMXAttrs{
					HxOn: event.SortByUpdated.UnsafeHxOnWithQueryParamAndValue(
						"click",
						"sort_by",
						sortByRank,
					),
				},
			},
			&widget.MenuItem{IsDivider: true},
		)
	}

	items = append(items,
		qq.sortMenuItem(
			widget.T("Newest first"),
			sortByNewestFirst,
			state.SortBy == sortByNewestFirst || state.SortBy == "",
		),
		qq.sortMenuItem(widget.T("Oldest first"), sortByOldestFirst, state.SortBy == sortByOldestFirst),
		qq.sortMenuItem(widget.T("Sort by name"), sortByName, state.SortBy == sortByName),
	)

	return &widget.Menu{
		Widget: widget.Widget[widget.Menu]{
			ID: "sortBy",
		},
		Position: widget.PositionLeft,
		Items:    items,
	}
}

func (qq *SortListContextMenuWidget) sortMenuItem(
	label *widget.Text,
	sortBy string,
	isSelected bool,
) *widget.MenuItem {
	return &widget.MenuItem{
		Label:          label,
		RadioGroupName: "SortBy",
		RadioValue:     sortBy,
		IsSelected:     isSelected,
		HTMXAttrs: widget.HTMXAttrs{
			HxOn: event.SortByUpdated.UnsafeHxOnWithQueryParamAndValue(
				"click",
				"sort_by",
				sortBy,
			),
		},
	}
}
