package browse

import (
	"fmt"
	"log"

	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/filesearch"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/model/common/attributetype"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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
		actions.Route("file-attributes"),
		false,
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

func (qq *FileAttributesPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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
) *wx.ScrollableContent {
	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.FilePropertiesID(),
		},
		GapY:     true,
		Children: qq.Content(ctx, data),
		MarginY:  true,
	}
}

func (qq *FileAttributesPartial) Content(
	ctx ctxx.Context,
	data *FileAttributesPartialData,
) wx.IWidget {
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

	log.Println(suggestedDocumentTypes)

	var suggestedDocumentTypeIDs []int64
	for _, documentType := range suggestedDocumentTypes {
		suggestedDocumentTypeIDs = append(suggestedDocumentTypeIDs, documentType.ID)
	}

	documentTypes := ctx.SpaceCtx().Space.QueryDocumentTypes().
		Where(documenttype.IDNotIn(suggestedDocumentTypeIDs...)).
		Order(documenttype.ByName()).
		AllX(ctx)

	if len(documentTypes) == 0 && len(suggestedDocumentTypes) == 0 {
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
	var attributeBlocks []*wx.Column

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

	return []wx.IWidget{
		&wx.Column{
			GapYSize:         wx.Gap2,
			NoOverflowHidden: true,
			AutoHeight:       true,
			HTMXAttrs: wx.HTMXAttrs{
				// TODO only reload affected tag group
				HxTrigger: event.TagUpdated.Handler(), // TODO only update if ID is identical
				HxPost:    qq.Endpoint(),
				HxVals:    util.JSON(qq.Data(filex.Data.PublicID.String())),
				HxTarget:  "#" + qq.FilePropertiesID(),
				HxSwap:    "outerHTML",
			},
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
	}
}

// TODO refactor should, just return one and caller should append
func (qq *FileAttributesPartial) documentTypeBadge(
	ctx ctxx.Context,
	data *FileAttributesPartialData,
	filex *model.File,
	documentType *enttenant.DocumentType,
	isSuggested bool,
	documentTypeChips []*wx.FilterChip,
	tagAssignmentsMap map[int64]bool,
	attributeBlocks []*wx.Column,
) ([]*wx.FilterChip, []*wx.Column) {
	// if selected, just show selected one, if nothing selected, show all
	if filex.Data.DocumentTypeID == 0 || filex.Data.DocumentTypeID == documentType.ID {
		trailingIcon := ""
		if filex.Data.DocumentTypeID == documentType.ID {
			trailingIcon = "close"
		}
		// TODO make it a InputChip instead of adding a `close` TrailingIcon?
		//		or at least make Icon and IconButton?
		documentTypeChips = append(documentTypeChips, &wx.FilterChip{
			Label:        wx.Tu(documentType.Name),
			IsChecked:    documentType.ID == filex.Data.DocumentTypeID,
			IsSuggestion: isSuggested,
			TrailingIcon: trailingIcon,
			HTMXAttrs: wx.HTMXAttrs{
				HxPost:   qq.actions.SelectDocumentTypePartial.Endpoint(),
				HxVals:   util.JSON(qq.actions.SelectDocumentTypePartial.Data(data.FileID, documentType.ID)),
				HxTarget: "#" + qq.FilePropertiesID(),
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
			var block *wx.Column
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
	filex *model.File,
	attributex *enttenant.Attribute,
) *wx.Column {
	var field wx.IWidget

	htmxAttrsFn := func(hxTrigger string) wx.HTMXAttrs {
		return wx.HTMXAttrs{
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

	var defaultValue string
	if nilableAssignment != nil {
		switch attributex.Edges.Property.Type {
		case fieldtype.Text:
			defaultValue = nilableAssignment.TextValue
		case fieldtype.Number:
			defaultValue = fmt.Sprintf("%d", nilableAssignment.NumberValue)
		case fieldtype.Money:
			// TODO
			val := float64(nilableAssignment.NumberValue) / 100.0
			defaultValue = fmt.Sprintf("%.2f", val)
		case fieldtype.Date:
			// TODO
			defaultValue = nilableAssignment.DateValue.Format("2006-01-02")
		}
	}

	switch attributex.Edges.Property.Type {
	case fieldtype.Text:
		field = &wx.TextField{
			Label:        wx.Tu(attributex.Edges.Property.Name),
			Name:         "TextValue",
			Type:         "text",
			DefaultValue: defaultValue,
			HTMXAttrs:    htmxAttrsFn("change, input delay:1000ms"),
		}
	case fieldtype.Number:
		field = &wx.TextField{
			Label:        wx.Tu(attributex.Edges.Property.Name),
			Name:         "NumberValue",
			Type:         "number",
			DefaultValue: defaultValue,
			// `change` event doesn't work because a change is triggered all the time a user uses arrow increase/decrease
			HTMXAttrs: htmxAttrsFn("input delay:1000ms"),
		}
	case fieldtype.Money:
		field = &wx.TextField{
			Label:        wx.Tu(attributex.Edges.Property.Name),
			Name:         "MoneyValue",
			Type:         "number",
			Step:         "0.01",
			DefaultValue: defaultValue,
			// `change` event doesn't work because a change is triggered all the time a user uses arrow increase/decrease
			HTMXAttrs: htmxAttrsFn("input delay:1000ms"),
		}
	case fieldtype.Date:
		field = &wx.TextField{
			Label:        wx.Tu(attributex.Edges.Property.Name),
			Name:         "DateValue",
			Type:         "date",
			DefaultValue: defaultValue,
			// short delay because going quickly up and down on day or month or year triggers change event
			HTMXAttrs: htmxAttrsFn("change delay:250ms"),
		}
	case fieldtype.Checkbox:
		// TODO cannot handle nil value; is this okay?
		isChecked := false
		if nilableAssignment != nil {
			isChecked = nilableAssignment.BoolValue
		}
		field = &wx.Checkbox{
			Label:     wx.Tu(attributex.Edges.Property.Name),
			Name:      "CheckboxValue",
			IsChecked: isChecked,
			HTMXAttrs: htmxAttrsFn("change"),
		}
	default:
		log.Println("unknown property type: ", attributex.Edges.Property.Type)
		return &wx.Column{} // TODO is there a better option
	}

	return &wx.Column{
		GapYSize:         wx.Gap2,
		NoOverflowHidden: true,
		AutoHeight:       true,
		Children: []wx.IWidget{
			/*&wx.Label{
				Text: wx.Tu(attributex.Edges.Property.Name),
				Type: wx.LabelTypeLg,
			},*/
			&wx.Container{
				Child: field,
				Gap:   true,
			},
		},
	}
}

func (qq *FileAttributesPartial) tagGroupAttributeBlock(
	ctx ctxx.Context,
	tagAssignmentsMap map[int64]bool,
	filex *model.File,
	attributex *enttenant.Attribute,
) *wx.Column {
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

	var chips []wx.IWidget

	chips = append(chips, &wx.AssistChip{
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

	return &wx.Column{
		GapYSize:         wx.Gap2,
		NoOverflowHidden: true,
		AutoHeight:       true,
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

func (qq *FileAttributesPartial) tagBadge(
	tagx *enttenant.Tag,
	chips []wx.IWidget,
	tagAssignmentsMap map[int64]bool,
	filex *model.File,
	isSuggested bool,
) []wx.IWidget {
	icon := "label"
	if tagx.Type == tagtype.Super {
		icon = "label_important"
	}
	chips = append(chips, &wx.FilterChip{
		Label:        wx.Tu(tagx.Name),
		LeadingIcon:  icon,
		Value:        fmt.Sprintf("%d", tagx.ID),
		IsChecked:    tagAssignmentsMap[tagx.ID],
		IsSuggestion: isSuggested,
		HTMXAttrs: wx.HTMXAttrs{
			HxPost: qq.actions.Tagging.ToggleFileTagCmd.Endpoint(),
			HxVals: util.JSON(qq.actions.Tagging.ToggleFileTagCmd.Data(filex.Data.ID, tagx.ID)),
			HxSwap: "none",
		},
	})
	return chips
}

func (qq *FileAttributesPartial) FilePropertiesID() string {
	return "fileProperties"
}
