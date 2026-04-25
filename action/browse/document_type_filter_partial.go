package browse

import (
	"fmt"
	"slices"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	actionx2 "github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/attribute"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tenant/common/attributetype"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type DocumentTypeFilterPartialData struct {
	CurrentDirID string
}

type DocumentTypeFilterPartialState struct {
	DocumentTypeID int64 `url:"document_type_id,omitempty"`
	// CheckedTagIDs  []int `url:"tag_ids,omitempty"` // shared with ListFilterTagsPartialState
}

type DocumentTypeFilterPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
}

func NewDocumentTypeFilterPartial(infra *common.Infra, actions *Actions) *DocumentTypeFilterPartial {
	return &DocumentTypeFilterPartial{
		infra:   infra,
		actions: actions,
		Config: actionx2.NewConfig(
			actions.Route("document-type-filter-partial"),
			true,
		),
	}
}

func (qq *DocumentTypeFilterPartial) Data(currentDirID string) *DocumentTypeFilterPartialData {
	return &DocumentTypeFilterPartialData{
		CurrentDirID: currentDirID,
	}
}

func (qq *DocumentTypeFilterPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DocumentTypeFilterPartialData](rw, req, ctx)
	if err != nil {
		return err
	}
	state := autil.StateX[ListDirPartialState](rw, req)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		autil.WrapWidget(
			widget.T("Document type | Filter"),
			nil,
			qq.Widget(ctx, data, state),
			actionx2.ResponseWrapperDialog,
			widget.DialogLayoutDefault,
		),
	)
}

func (qq *DocumentTypeFilterPartial) Widget(
	ctx ctxx.Context,
	data *DocumentTypeFilterPartialData,
	state *ListDirPartialState,
) renderable.Renderable {
	// TODO show only document types that exist in current folder or subfolder
	// TODO only show tags that are in use...
	documentTypes := ctx.SpaceCtx().Space.QueryDocumentTypes().Order(documenttype.ByName()).AllX(ctx)

	if len(documentTypes) == 0 {
		return &widget.EmptyState{
			Headline: widget.T("No document types available yet."),
			Actions: []widget.IWidget{
				&widget.Button{
					Icon:  widget.NewIcon("category"),
					Label: widget.T("Manage document types"),
					HTMXAttrs: widget.HTMXAttrs{
						HxGet: route.ManageDocumentTypes(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
					},
				},
			},
		}
	}

	var documentTypeChips []*widget.FilterChip
	var attributeBlocks []widget.IWidget

	for _, documentType := range documentTypes {
		// if selected, just show selected one, if nothing selected, show all
		if state.DocumentTypeID == 0 || state.DocumentTypeID == documentType.ID {
			trailingIcon := ""
			if state.DocumentTypeID == documentType.ID {
				trailingIcon = "close"
			}
			// TODO make it a InputChip instead of adding a `close` TrailingIcon?
			//		or at least make Icon and IconButton?
			documentTypeChips = append(documentTypeChips, &widget.FilterChip{
				Name:         "DocumentTypeID", // must match name in struct
				Label:        widget.Tu(documentType.Name),
				IsChecked:    documentType.ID == state.DocumentTypeID,
				Value:        fmt.Sprintf("%d", documentType.ID),
				TrailingIcon: trailingIcon,
				HTMXAttrs: widget.HTMXAttrs{
					HxPost: qq.actions.ToggleDocumentTypeFilterCmd.Endpoint(),
					HxVals: util.JSON(qq.actions.ToggleDocumentTypeFilterCmd.Data(data.CurrentDirID, documentType.ID)),
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
			attributeBlocks = append(attributeBlocks, &widget.Column{
				NoOverflowHidden: true,
				GapYSize:         widget.Gap1,
				Children: []widget.IWidget{
					widget.NewLabel(widget.LabelTypeLg, widget.T("Fields")),
					qq.actions.ListFilterPropertiesPartial.Widget(
						ctx,
						qq.actions.ListFilterPropertiesPartial.Data(data.CurrentDirID, documentType.ID),
						state,
					),
				},
			})

			// TODO ordering
			attributesx := documentType.QueryAttributes().Where(attribute.TypeEQ(attributetype.Tag)).AllX(ctx)
			var tagGroupAttributes []widget.IWidget
			for _, attributex := range attributesx {
				tagGroupAttributes = append(
					tagGroupAttributes,
					qq.attributeBlock(ctx, data, state, attributex),
				)
			}
			if len(tagGroupAttributes) == 0 {
				tagGroupAttributes = append(
					tagGroupAttributes,
					widget.NewLabel(widget.LabelTypeLg, widget.T("Tag groups")),
					widget.T("No tag groups available."),
				)
			}
			attributeBlocks = append(attributeBlocks, &widget.Column{
				GapYSize:         widget.Gap4,
				NoOverflowHidden: true,
				Children:         tagGroupAttributes,
			})
		}
	}

	return &widget.Container{
		Widget: widget.Widget[widget.Container]{
			ID: qq.ID(),
		},
		GapY: true,
		Child: []widget.IWidget{
			&widget.Column{
				GapYSize:         widget.Gap2,
				NoOverflowHidden: true,
				Children: []widget.IWidget{
					&widget.Label{
						Text: widget.T("Document type"),
						Type: widget.LabelTypeLg,
					},
					&widget.Container{
						Gap:   true,
						Child: documentTypeChips,
					},
				},
			},
			attributeBlocks,
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxOn: event.DocumentTypeFilterChanged.HxOn("change"),
		},
	}
}

func (qq *DocumentTypeFilterPartial) attributeBlock(
	ctx ctxx.Context,
	data *DocumentTypeFilterPartialData,
	state *ListDirPartialState,
	attributex *enttenant.Attribute,
) *widget.Column {
	// TODO not efficient; do one query one layer above?
	//		is it possible to query all and filter down on demand? or implement helper to split
	// TODO show selected first
	tags := ctx.SpaceCtx().Space.QueryTags().Order(tag.ByName()).Where(tag.GroupID(attributex.TagID)).AllX(ctx)
	var chips []widget.IWidget

	for _, tagx := range tags {
		icon := "label"
		if tagx.Type == tagtype.Super {
			icon = "label_important"
		}
		chips = append(chips, &widget.FilterChip{
			Label:       widget.Tu(tagx.Name),
			LeadingIcon: icon,
			IsChecked:   slices.Contains(state.CheckedTagIDs, int(tagx.ID)),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost: qq.actions.ToggleTagFilterCmd.Endpoint(),
				HxVals: util.JSON(qq.actions.ToggleTagFilterCmd.Data(data.CurrentDirID, tagx.ID)),
				HxSwap: "none",
			},
		})
	}

	return &widget.Column{
		GapYSize:         widget.Gap2,
		NoOverflowHidden: true,
		Children: []widget.IWidget{
			&widget.Label{
				Text: widget.Tu(attributex.Name),
				Type: widget.LabelTypeLg,
			},
			&widget.Container{
				Child: chips,
				Gap:   true,
			},
		},
	}
}

func (qq *DocumentTypeFilterPartial) ID() string {
	return "documentTypeFilter"
}
