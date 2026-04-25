package browse

import (
	"fmt"
	"log"

	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/model/common/fieldtype"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/filesearch"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/tenant/common/attributetype"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type FileAttributesPartialData struct {
	FileID string
	// DocumentTypeID int64
}

type FileAttributesPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileAttributesPartial(infra *common.Infra, actions *Actions) *FileAttributesPartial {
	config := actionx.NewConfig(
		actions.Route("file-attributes-partial"),
		true,
	)
	return &FileAttributesPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FileAttributesPartial) Data(fileID string) *FileAttributesPartialData {
	return &FileAttributesPartialData{
		FileID: fileID,
		// DocumentTypeID: documentTypeID,
	}
}

func (qq *FileAttributesPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileAttributesPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FileAttributesPartial) Widget(
	ctx ctxx.Context,
	data *FileAttributesPartialData,
) *widget.ScrollableContent {
	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.FileAttributesID(),
		},
		GapY:     true,
		Children: qq.Content(ctx, data),
		MarginY:  true,
	}
}

func (qq *FileAttributesPartial) Content(
	ctx ctxx.Context,
	data *FileAttributesPartialData,
) widget.IWidget {
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	suggestedDocumentTypes := ctx.SpaceCtx().Space.QueryDocumentTypes().
		Order(documenttype.ByName()).
		Where(
			func(qs *sql.Selector) {
				fileSearchTable := sql.Table(filesearch.Table)
				qs.Where(
					sql.Exists(
						sql.Select(fileSearchTable.C(filesearch.FieldRowid)).From(fileSearchTable).
							Where(
								sql.And(
									// Rowid is internal id
									sql.EQ(fileSearchTable.C(filesearch.FieldRowid), filex.Data.ID),
									sql.ExprP(
										fileSearchTable.C(filesearch.FieldFileSearches)+" MATCH "+
											`'"' || replace(`+qs.C(documenttype.FieldName)+`, '"', '""') || '"'`,
									),
									// sql.EQ(
									// fileSearchTable.C(filesearch.FieldFileSearches),
									// `'"' || replace(`+qs.C(documenttype.FieldName)+`, '"', '""') || '"'`,
									// ),
									sql.LT(fileSearchTable.C(filesearch.FieldRank), 0), // TODO what is a good threshold?
								),
							),
					),
				)
			},
		).
		// Limit(3). // would need order by filesearch.FieldRank
		AllX(ctx)

	var suggestedDocumentTypeIDs []int64
	for _, documentType := range suggestedDocumentTypes {
		suggestedDocumentTypeIDs = append(suggestedDocumentTypeIDs, documentType.ID)
	}

	documentTypes := ctx.SpaceCtx().Space.QueryDocumentTypes().
		Where(documenttype.IDNotIn(suggestedDocumentTypeIDs...)).
		Order(documenttype.ByName()).
		AllX(ctx)

	if len(documentTypes) == 0 && len(suggestedDocumentTypes) == 0 {
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
	var attributeBlocks []*widget.Column

	tagAssignmentsMap := make(map[int64]bool)
	for _, tagAssignment := range filex.Data.QueryTagAssignment().AllX(ctx) {
		tagAssignmentsMap[tagAssignment.TagID] = true
	}

	for _, documentType := range suggestedDocumentTypes {
		documentTypeChips, attributeBlocks = qq.documentTypeBadge(
			ctx,
			data,
			filex,
			documentType,
			true,
			documentTypeChips,
			tagAssignmentsMap,
			attributeBlocks,
		)
	}
	for _, documentType := range documentTypes {
		documentTypeChips, attributeBlocks = qq.documentTypeBadge(
			ctx,
			data,
			filex,
			documentType,
			false,
			documentTypeChips,
			tagAssignmentsMap,
			attributeBlocks,
		)
	}

	return []widget.IWidget{
		&widget.Column{
			GapYSize:         widget.Gap2,
			NoOverflowHidden: true,
			AutoHeight:       true,
			HTMXAttrs: widget.HTMXAttrs{
				// TODO only reload affected tag group
				// TODO only update if ID is identical
				HxTrigger: events.HxTrigger(event.TagUpdated),
				HxPost:    qq.Endpoint(),
				HxVals:    util.JSON(qq.Data(filex.Data.PublicID.String())),
				HxTarget:  "#" + qq.FileAttributesID(),
				HxSwap:    "outerHTML",
			},
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
	}
}

// TODO refactor should, just return one and caller should append
func (qq *FileAttributesPartial) documentTypeBadge(
	ctx ctxx.Context,
	data *FileAttributesPartialData,
	filex *filemodel.File,
	documentType *enttenant.DocumentType,
	isSuggested bool,
	documentTypeChips []*widget.FilterChip,
	tagAssignmentsMap map[int64]bool,
	attributeBlocks []*widget.Column,
) ([]*widget.FilterChip, []*widget.Column) {
	// if selected, just show selected one, if nothing selected, show all
	if filex.Data.DocumentTypeID == 0 || filex.Data.DocumentTypeID == documentType.ID {
		trailingIcon := ""
		if filex.Data.DocumentTypeID == documentType.ID {
			trailingIcon = "close"
		}
		// TODO make it a InputChip instead of adding a `close` TrailingIcon?
		//		or at least make Icon and IconButton?
		documentTypeChips = append(documentTypeChips, &widget.FilterChip{
			Label:        widget.Tu(documentType.Name),
			IsChecked:    documentType.ID == filex.Data.DocumentTypeID,
			IsSuggestion: isSuggested,
			TrailingIcon: trailingIcon,
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:   qq.actions.SelectDocumentTypePartial.Endpoint(),
				HxVals:   util.JSON(qq.actions.SelectDocumentTypePartial.Data(data.FileID, documentType.ID)),
				HxTarget: "#" + qq.FileAttributesID(),
				HxSwap:   "outerHTML",
				HxHeaders: autil.QueryHeader(
					qq.Endpoint(),
					qq.Data(data.FileID),
				),
			},
		})
	}

	if documentType.ID == filex.Data.DocumentTypeID {
		// TODO ordering
		attributes := documentType.QueryAttributes().WithProperty().AllX(ctx)
		for _, attributex := range attributes {
			var block *widget.Column
			switch attributex.Type {
			case attributetype.Field:
				block = qq.propertyAttributeBlock(ctx, filex, attributex)
			case attributetype.Tag:
				block = qq.tagGroupAttributeBlock(ctx, tagAssignmentsMap, filex, attributex)
			default:
				log.Println("unknown attribute type: ", attributex.Type)
				continue
			}

			attributeBlocks = append(
				attributeBlocks,
				block,
			)
		}
	}
	return documentTypeChips, attributeBlocks
}

func (qq *FileAttributesPartial) propertyAttributeBlock(
	ctx ctxx.Context,
	filex *filemodel.File,
	attributex *enttenant.Attribute,
) *widget.Column {
	htmxAttrsFn := func(hxTrigger string) widget.HTMXAttrs {
		return widget.HTMXAttrs{
			HxTrigger: hxTrigger,
			HxPost:    qq.actions.SetFilePropertyCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.SetFilePropertyCmd.Data(filex.Data.PublicID.String(), attributex.Edges.Property.ID)),
			HxInclude: "this",
		}
	}

	nilableAssignment, err := ctx.SpaceCtx().TTx.FilePropertyAssignment.Query().
		Where(
			filepropertyassignment.PropertyID(attributex.Edges.Property.ID),
			filepropertyassignment.FileID(filex.Data.ID),
		).Only(ctx)
	if err != nil && !enttenant.IsNotFound(err) {
		panic(err)
	}

	field, found := fieldByProperty(attributex.Edges.Property, nilableAssignment, htmxAttrsFn)
	if !found {
		log.Println("unknown property type: ", attributex.Edges.Property.Type)
		return &widget.Column{}
	}

	children := []widget.IWidget{
		&widget.Container{
			Child: field,
			Gap:   true,
		},
	}

	if attributex.Edges.Property.Type == fieldtype.Date {
		hasDateValue := false
		if nilableAssignment != nil && !nilableAssignment.DateValue.IsZero() {
			hasDateValue = true
		}
		fieldID := field.(widget.IWidgetWithID).GetID()
		dateSuggestionsWidget := NewDateSuggestionsWidget(filex, fieldID, attributex.Edges.Property.ID)
		children = append(children, dateSuggestionsWidget.Widget(
			ctx,
			!hasDateValue,
			"",
		))
	}

	return &widget.Column{
		GapYSize:         widget.Gap2,
		NoOverflowHidden: true,
		AutoHeight:       true,
		Children:         children,
	}
}

func (qq *FileAttributesPartial) tagGroupAttributeBlock(
	ctx ctxx.Context,
	tagAssignmentsMap map[int64]bool,
	filex *filemodel.File,
	attributex *enttenant.Attribute,
) *widget.Column {
	suggestedTags := ctx.SpaceCtx().Space.QueryTags().
		Order(tag.ByName()).
		Where(
			func(qs *sql.Selector) {
				fileSearchTable := sql.Table(filesearch.Table)
				qs.Where(
					sql.Exists(
						sql.Select(qs.C(filesearch.FieldRowid)).From(fileSearchTable).
							Where(
								sql.And(
									// Rowid is internal id
									sql.EQ(fileSearchTable.C(filesearch.FieldRowid), filex.Data.ID),
									sql.EQ(tag.FieldGroupID, attributex.TagID),
									sql.ExprP(
										fileSearchTable.C(filesearch.FieldFileSearches)+" MATCH "+
											`'"' || replace(`+qs.C(tag.FieldName)+`, '"', '""') || '"'`,
									),
									sql.LT(fileSearchTable.C(filesearch.FieldRank), 0),
								),
							),
					),
				)
			},
		).
		AllX(ctx)

	var suggestedTagIDs []int64
	for _, tagx := range suggestedTags {
		suggestedTagIDs = append(suggestedTagIDs, tagx.ID)
	}

	// TODO not efficient; do one query one layer above?
	//		is it possible to query all and filter down on demand? or implement helper to split
	// TODO show selected first
	tags := ctx.SpaceCtx().Space.QueryTags().
		Order(tag.ByName()).
		Where(tag.GroupID(attributex.TagID), tag.IDNotIn(suggestedTagIDs...)).
		AllX(ctx)

	var chips []widget.IWidget

	chips = append(chips, &widget.AssistChip{
		// Label:        wx.T("Add"),
		LeadingIcon: "add",
		HTMXAttrs: qq.actions.Tagging.AssignedTags.CreateAndAssignTagCmd.ModalLinkAttrs(
			qq.actions.Tagging.AssignedTags.CreateAndAssignTagCmd.Data(filex.Data.PublicID.String(), attributex.TagID),
			"",
		),
	})

	for _, tagx := range suggestedTags {
		chips = qq.tagBadge(tagx, chips, tagAssignmentsMap, filex, true)
	}
	for _, tagx := range tags {
		chips = qq.tagBadge(tagx, chips, tagAssignmentsMap, filex, false)
	}

	// attributeBlockID := fmt.Sprintf("attributeBlock-%d", attributex.ID)

	return &widget.Column{
		GapYSize:         widget.Gap2,
		NoOverflowHidden: true,
		AutoHeight:       true,
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

func (qq *FileAttributesPartial) tagBadge(
	tagx *enttenant.Tag,
	chips []widget.IWidget,
	tagAssignmentsMap map[int64]bool,
	filex *filemodel.File,
	isSuggested bool,
) []widget.IWidget {
	icon := "label"
	if tagx.Type == tagtype.Super {
		icon = "label_important"
	}
	chips = append(chips, &widget.FilterChip{
		Label:        widget.Tu(tagx.Name),
		LeadingIcon:  icon,
		Value:        fmt.Sprintf("%d", tagx.ID),
		IsChecked:    tagAssignmentsMap[tagx.ID],
		IsSuggestion: isSuggested,
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.actions.Tagging.ToggleFileTagCmd.Endpoint(),
			HxVals: util.JSON(qq.actions.Tagging.ToggleFileTagCmd.Data(filex.Data.ID, tagx.ID)),
			HxSwap: "none",
		},
	})
	return chips
}

func (qq *FileAttributesPartial) FileAttributesID() string {
	return "fileAttributes"
}
