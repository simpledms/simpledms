package browse

import (
	"fmt"
	"slices"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/attribute"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/common/attributetype"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DocumentTypeFilterData struct {
	CurrentDirID string
}

type DocumentTypeFilterState struct {
	DocumentTypeID int64 `url:"document_type_id,omitempty"`
	// CheckedTagIDs  []int `url:"tag_ids,omitempty"` // shared with ListFilterTagsState
}

type DocumentTypeFilter struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDocumentTypeFilter(infra *common.Infra, actions *Actions) *DocumentTypeFilter {
	return &DocumentTypeFilter{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("document-type-filter"),
			true,
		),
	}
}

func (qq *DocumentTypeFilter) Data(currentDirID string) *DocumentTypeFilterData {
	return &DocumentTypeFilterData{
		CurrentDirID: currentDirID,
	}
}

func (qq *DocumentTypeFilter) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DocumentTypeFilterData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirState](rw, req)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		autil.WrapWidget(
			wx.T("Document type | Filter"),
			nil,
			qq.Widget(ctx, data, state),
			actionx.ResponseWrapperDialog,
			wx.DialogLayoutDefault,
		),
	)
}

func (qq *DocumentTypeFilter) Widget(
	ctx ctxx.Context,
	data *DocumentTypeFilterData,
	state *ListDirState,
) renderable.Renderable {
	// TODO show only document types that exist in current folder or subfolder
	// TODO only show tags that are in use...
	documentTypes := ctx.SpaceCtx().Space.QueryDocumentTypes().Order(documenttype.ByName()).AllX(ctx)

	if len(documentTypes) == 0 {
		return &wx.EmptyState{
			Headline: wx.T("No document types available yet."),
			Actions: []wx.IWidget{
				&wx.Button{
					Icon:  wx.NewIcon("category"),
					Label: wx.T("Manage document types"),
					HTMXAttrs: wx.HTMXAttrs{
						HxGet: route.ManageDocumentTypes(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
					},
				},
			},
		}
	}

	var documentTypeChips []*wx.FilterChip
	var attributeBlocks []wx.IWidget

	for _, documentType := range documentTypes {
		// if selected, just show selected one, if nothing selected, show all
		if state.DocumentTypeID == 0 || state.DocumentTypeID == documentType.ID {
			trailingIcon := ""
			if state.DocumentTypeID == documentType.ID {
				trailingIcon = "close"
			}
			// TODO make it a InputChip instead of adding a `close` TrailingIcon?
			//		or at least make Icon and IconButton?
			documentTypeChips = append(documentTypeChips, &wx.FilterChip{
				Name:         "DocumentTypeID", // must match name in struct
				Label:        wx.Tu(documentType.Name),
				IsChecked:    documentType.ID == state.DocumentTypeID,
				Value:        fmt.Sprintf("%d", documentType.ID),
				TrailingIcon: trailingIcon,
				HTMXAttrs: wx.HTMXAttrs{
					HxPost: qq.actions.ToggleDocumentTypeFilter.Endpoint(),
					HxVals: util.JSON(qq.actions.ToggleDocumentTypeFilter.Data(data.CurrentDirID, documentType.ID)),
					// HxSwap: "none",
					HxHeaders: autil.QueryHeader(
						qq.Endpoint(),
						qq.Data(data.CurrentDirID),
					),
					HxTarget: "#" + qq.ID(),
					HxSelect: "#" + qq.ID(),
					HxSwap:   "outerHTML",
				},
			})
		}

		if documentType.ID == state.DocumentTypeID {
			attributeBlocks = append(attributeBlocks, &wx.Column{
				NoOverflowHidden: true,
				GapYSize:         wx.Gap1,
				Children: []wx.IWidget{
					wx.NewLabel(wx.LabelTypeLg, wx.T("Fields")),
					qq.actions.ListFilterProperties.Widget(
						ctx,
						qq.actions.ListFilterProperties.Data(data.CurrentDirID, documentType.ID),
						state,
					),
				},
			})

			// TODO ordering
			attributesx := documentType.QueryAttributes().Where(attribute.TypeEQ(attributetype.Tag)).AllX(ctx)
			var tagGroupAttributes []wx.IWidget
			for _, attributex := range attributesx {
				tagGroupAttributes = append(
					tagGroupAttributes,
					qq.attributeBlock(ctx, data, state, attributex),
				)
			}
			tagGroupAttributes = append(tagGroupAttributes, wx.NewLabel(wx.LabelTypeLg, wx.T("Tag groups")))
			if len(tagGroupAttributes) == 1 {
				tagGroupAttributes = append(tagGroupAttributes, wx.T("No tag groups available."))
			}
			attributeBlocks = append(attributeBlocks, &wx.Column{
				GapYSize:         wx.Gap1,
				NoOverflowHidden: true,
				Children:         tagGroupAttributes,
			})
		}
	}

	return &wx.Container{
		Widget: wx.Widget[wx.Container]{
			ID: qq.ID(),
		},
		GapY: true,
		Child: []wx.IWidget{
			&wx.Column{
				GapYSize:         wx.Gap2,
				NoOverflowHidden: true,
				Children: []wx.IWidget{
					&wx.Label{
						Text: wx.T("Document type"),
						Type: wx.LabelTypeLg,
					},
					&wx.Container{
						Gap:   true,
						Child: documentTypeChips,
					},
				},
			},
			attributeBlocks,
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxOn: event.DocumentTypeFilterChanged.HxOn("change"),
		},
	}
}

func (qq *DocumentTypeFilter) attributeBlock(
	ctx ctxx.Context,
	data *DocumentTypeFilterData,
	state *ListDirState,
	attributex *enttenant.Attribute,
) *wx.Column {
	// TODO not efficient; do one query one layer above?
	//		is it possible to query all and filter down on demand? or implement helper to split
	// TODO show selected first
	tags := ctx.SpaceCtx().Space.QueryTags().Order(tag.ByName()).Where(tag.GroupID(attributex.TagID)).AllX(ctx)
	var chips []wx.IWidget

	for _, tagx := range tags {
		icon := "label"
		if tagx.Type == tagtype.Super {
			icon = "label_important"
		}
		chips = append(chips, &wx.FilterChip{
			Label:       wx.Tu(tagx.Name),
			LeadingIcon: icon,
			IsChecked:   slices.Contains(state.CheckedTagIDs, int(tagx.ID)),
			HTMXAttrs: wx.HTMXAttrs{
				HxPost: qq.actions.ToggleTagFilter.Endpoint(),
				HxVals: util.JSON(qq.actions.ToggleTagFilter.Data(data.CurrentDirID, tagx.ID)),
				HxSwap: "none",
			},
		})
	}

	return &wx.Column{
		GapYSize: wx.Gap2,
		Children: []wx.IWidget{
			&wx.Label{
				Text: wx.Tu(attributex.Name),
				Type: wx.LabelTypeLg,
			},
			&wx.Container{
				Child: chips,
				Gap:   true,
			},
		},
	}
}

func (qq *DocumentTypeFilter) ID() string {
	return "documentTypeFilter"
}
