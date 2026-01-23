package browse

import (
	"maps"
	"slices"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type ListFilterTagsPartialData struct {
	CurrentDirID string
}

type ListFilterTagsPartialState struct {
	// int instead of int64 because of query (sql.InInts)
	// TODO is this okay? should be if we run on 64 bit system
	CheckedTagIDs []int `url:"tag_ids,omitempty"` // shared with DocumentTypeFilterPartialState
}

type ListFilterTagsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewListFilterTagsPartial(infra *common.Infra, actions *Actions) *ListFilterTagsPartial {
	return &ListFilterTagsPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("list-filter-tags-partial"),
			true,
		),
	}
}

func (qq *ListFilterTagsPartial) Data(currentDirID string) *ListFilterTagsPartialData {
	return &ListFilterTagsPartialData{
		CurrentDirID: currentDirID,
	}
}

func (qq *ListFilterTagsPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[ListFilterTagsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListFilterTagsPartialState](rw, req)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		autil.WrapWidget(
			wx.T("Tags | Filter"),
			nil,
			qq.Widget(ctx, data.CurrentDirID, state.CheckedTagIDs),
			actionx.ResponseWrapperDialog,
			wx.DialogLayoutDefault,
		),
	)
}

// TODO duplicate in search
func (qq *ListFilterTagsPartial) Widget(
	ctx ctxx.Context,
	currentDirID string,
	checkedTagIDs []int,
) renderable.Renderable {
	/* deactivated on 26.02.25 because it also comes with some problems, for example if
	folder is changed, a selected tag might get out of scope and is no longer deselectable;
	also did not play nicely with document type filters

	// TODO is this faster than old solution?
	// TODO can be simplified when edges work on resolvedTagAssignment
	fileInfoView := sql.Table(fileinfo.Table)
	tagsInScope := ctx.TenantCtx().TTx.Tag.Query().
		Where(func(qss *sql.Selector) {
			resolvedTagAssignmentView := sql.Table(resolvedtagassignment.Table)
			qss.Where(
				sql.In(
					qss.C(tag.FieldID),
					sql.Select(resolvedTagAssignmentView.C(resolvedtagassignment.FieldTagID)).
						From(resolvedTagAssignmentView).
						Where(sql.In(
							resolvedTagAssignmentView.C(resolvedtagassignment.FieldFileID),
							// subquery to select all tags of files in search scope
							sql.Select(fileInfoView.C(fileinfo.FieldFileID)).
								From(fileInfoView).
								Where(sql.And(
									sqljson.ValueContains(fileInfoView.C(fileinfo.FieldPath), currentDirID),
									sql.NEQ(fileInfoView.C(fileinfo.FieldFileID), currentDirID),
								)),
						)),
				))
		}).
		WithGroup().
		Order(tag.ByName()).
		Unique(true).
		AllX(ctx)
	*/

	tagsInScope := ctx.SpaceCtx().Space.QueryTags().
		WithGroup().
		Order(tag.ByName()).
		Unique(true).
		AllX(ctx)

	if len(tagsInScope) == 0 {
		return &wx.EmptyState{
			Headline: wx.T("No tags available yet."),
			Actions: []wx.IWidget{
				&wx.Button{
					Icon:  wx.NewIcon("label"),
					Label: wx.T("Manage tags"),
					HTMXAttrs: wx.HTMXAttrs{
						HxGet: route.ManageTags(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
					},
				},
			},
		}
	}

	// var chips []*Chip
	var chips []*wx.FilterChip
	groups := map[string][]*wx.FilterChip{}

	for _, tagx := range tagsInScope {
		icon := "label"
		if tagx.Type == tagtype.Super {
			icon = "label_important"
		}

		// TODO indicate if a composed tag, by color?
		chip := &wx.FilterChip{
			Label:       wx.Tf(tagx.Name),
			LeadingIcon: icon,
			IsChecked:   slices.Contains(checkedTagIDs, int(tagx.ID)), // TODO cast okay?
			HTMXAttrs: wx.HTMXAttrs{
				HxPost: qq.actions.ToggleTagFilterCmd.Endpoint(),
				HxVals: util.JSON(qq.actions.ToggleTagFilterCmd.Data(currentDirID, tagx.ID)),
				HxSwap: "none",
			},
		}

		if tagx.Edges.Group != nil {
			groupName := tagx.Edges.Group.Name
			groupChips, found := groups[groupName]
			if !found {
				groupChips = []*wx.FilterChip{}
			}
			groupChips = append(groupChips, chip)
			groups[groupName] = groupChips
		} else {
			chips = append(chips, chip)
		}
	}

	children := []wx.IWidget{
		&wx.Container{
			Child: chips,
		},
	}

	for _, groupKey := range slices.Sorted(maps.Keys(groups)) {
		groupChips := groups[groupKey]

		children = append(children, &wx.Container{
			Child: []wx.IWidget{
				wx.H(wx.HeadingTypeTitleMd, wx.Tu(groupKey)),
				groupChips,
			},
		})
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: "filterTags",
		},
		// HTMXAttrs: wx.HTMXAttrs{
		// HxOn: event.FilterTagsChanged.HxOn("change"),
		// },
		GapY:  true,
		Child: children,
	}
}
