package inbox

import (
	"fmt"
	"sort"
	"strings"

	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/attribute"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/enttenant/tagassignment"
	"github.com/simpledms/simpledms/model/main/common/attributetype"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	"github.com/simpledms/simpledms/model/main/filelistpreference"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/fileutil"
	"github.com/simpledms/simpledms/util/timex"
)

func (qq *FilesListPartial) fileTable(
	ctx ctxx.Context,
	data *FilesListPartialData,
	files []*enttenant.File,
	preferences *filelistpreference.FileListPreferences,
) *wx.Table {
	spaceColumns := preferences.SpaceColumnsFor(ctx.SpaceCtx().SpaceID)
	propertyColumns := qq.fileTablePropertyColumns(ctx, spaceColumns.PropertyIDs)
	tagGroupColumns := qq.fileTableTagGroupColumns(ctx, spaceColumns.TagGroupIDs)
	columnData := qq.fileTableColumnData(
		ctx,
		files,
		preferences,
		spaceColumns,
		propertyColumns,
		tagGroupColumns,
	)
	columns := qq.fileTableColumns(preferences, spaceColumns, propertyColumns, tagGroupColumns)
	rows := make([]*wx.TableRow, 0, len(files))
	for _, filex := range files {
		rows = append(rows, qq.fileTableRow(
			ctx,
			data,
			filex,
			preferences,
			spaceColumns,
			propertyColumns,
			tagGroupColumns,
			columnData,
		))
	}

	return &wx.Table{
		Columns:      columns,
		Rows:         rows,
		HideOnMobile: true,
	}
}

func (qq *FilesListPartial) fileTableColumns(
	preferences *filelistpreference.FileListPreferences,
	spaceColumns *filelistpreference.SpaceFileListColumns,
	propertyColumns []*enttenant.Property,
	tagGroupColumns []*enttenant.Tag,
) []*wx.TableColumn {
	columns := []*wx.TableColumn{}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnName) {
		columns = append(columns, &wx.TableColumn{Label: wx.T("Name")})
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnDocumentType) {
		columns = append(columns, &wx.TableColumn{Label: wx.T("Type")})
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnMetadata) {
		columns = append(columns, &wx.TableColumn{Label: wx.T("Metadata")})
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnDate) {
		columns = append(columns, &wx.TableColumn{Label: wx.T("Date")})
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnSize) {
		columns = append(columns, &wx.TableColumn{Label: wx.T("Size")})
	}
	if spaceColumns.ShowTags {
		columns = append(columns, &wx.TableColumn{Label: wx.T("Tags")})
	}
	for _, propertyx := range propertyColumns {
		columns = append(columns, &wx.TableColumn{Label: wx.Tu(propertyx.Name)})
	}
	for _, tagGroup := range tagGroupColumns {
		columns = append(columns, &wx.TableColumn{Label: wx.Tu(tagGroup.Name)})
	}
	return columns
}

func (qq *FilesListPartial) fileTableRow(
	ctx ctxx.Context,
	data *FilesListPartialData,
	filex *enttenant.File,
	preferences *filelistpreference.FileListPreferences,
	spaceColumns *filelistpreference.SpaceFileListColumns,
	propertyColumns []*enttenant.Property,
	tagGroupColumns []*enttenant.Tag,
	columnData *fileTableColumnData,
) *wx.TableRow {
	cells := []*wx.TableCell{}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnName) {
		cells = append(cells, &wx.TableCell{Child: qq.fileTableNameCell(filex)})
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnDocumentType) {
		cells = append(cells, &wx.TableCell{Child: wx.Tu(columnData.documentTypes[filex.DocumentTypeID])})
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnMetadata) {
		cells = append(cells, &wx.TableCell{Child: wx.Tu(columnData.metadata[filex.ID])})
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnDate) {
		cells = append(cells, &wx.TableCell{Child: wx.Tu(timex.NewDateTime(filex.CreatedAt).String(ctx.MainCtx().LanguageBCP47))})
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnSize) {
		cells = append(cells, &wx.TableCell{Child: wx.Tu(columnData.sizes[filex.ID])})
	}
	if spaceColumns.ShowTags {
		cells = append(cells, &wx.TableCell{Child: wx.Tu(strings.Join(columnData.tags[filex.ID], ", "))})
	}
	for _, propertyx := range propertyColumns {
		cells = append(cells, &wx.TableCell{Child: wx.Tu(columnData.properties[filex.ID][propertyx.ID])})
	}
	for _, tagGroup := range tagGroupColumns {
		cells = append(cells, &wx.TableCell{Child: wx.Tu(strings.Join(columnData.tagGroups[filex.ID][tagGroup.ID], ", "))})
	}

	return &wx.TableRow{
		HTMXAttrs:   qq.fileTableRowHTMXAttrs(ctx, filex),
		Cells:       cells,
		ContextMenu: NewFileContextMenuWidget(qq.actions).Widget(ctx, filex),
		IsSelected:  filex.PublicID.String() == data.SelectedFileID,
	}
}

func (qq *FilesListPartial) fileTableNameCell(filex *enttenant.File) wx.IWidget {
	return &wx.Row{
		Children: []wx.IWidget{
			wx.NewIcon("description"),
			wx.Tu(filex.Name),
		},
	}
}

func (qq *FilesListPartial) fileTableRowHTMXAttrs(ctx ctxx.Context, filex *enttenant.File) wx.HTMXAttrs {
	return wx.HTMXAttrs{
		HxTarget: "#details",
		HxSwap:   "outerHTML",
		HxGet: route.Inbox(
			ctx.TenantCtx().TenantID,
			ctx.SpaceCtx().SpaceID,
			filex.PublicID.String(),
		),
		HxHeaders: autil.PreserveStateHeader(),
	}
}

type fileTableColumnData struct {
	documentTypes map[int64]string
	metadata      map[int64]string
	sizes         map[int64]string
	tags          map[int64][]string
	tagGroups     map[int64]map[int64][]string
	properties    map[int64]map[int64]string
}

func (qq *FilesListPartial) fileTableColumnData(
	ctx ctxx.Context,
	files []*enttenant.File,
	preferences *filelistpreference.FileListPreferences,
	spaceColumns *filelistpreference.SpaceFileListColumns,
	propertyColumns []*enttenant.Property,
	tagGroupColumns []*enttenant.Tag,
) *fileTableColumnData {
	data := &fileTableColumnData{
		documentTypes: map[int64]string{},
		metadata:      map[int64]string{},
		sizes:         map[int64]string{},
		tags:          map[int64][]string{},
		tagGroups:     map[int64]map[int64][]string{},
		properties:    map[int64]map[int64]string{},
	}
	ids := fileIDs(files)
	if len(ids) == 0 {
		return data
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnDocumentType) {
		data.documentTypes = fileTableDocumentTypes(ctx, files)
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnMetadata) {
		data.metadata = fileTableMetadata(ctx, files)
	}
	if preferences.HasBuiltInColumn(filelistpreference.FileListColumnSize) {
		data.sizes = fileTableSizes(ctx, ids)
	}
	if spaceColumns.ShowTags {
		data.tags = fileTableTags(ctx, ids)
	}
	if len(propertyColumns) > 0 {
		data.properties = fileTableProperties(ctx, ids, propertyIDs(propertyColumns))
	}
	if len(tagGroupColumns) > 0 {
		data.tagGroups = fileTableTagValuesByGroup(ctx, ids, tagIDs(tagGroupColumns))
	}
	return data
}

type fileTableAttributeValue struct {
	label string
	value string
}

func fileTableMetadata(ctx ctxx.Context, files []*enttenant.File) map[int64]string {
	fileIDs := fileIDs(files)
	documentTypeIDs := fileDocumentTypeIDs(files)
	if len(fileIDs) == 0 || len(documentTypeIDs) == 0 {
		return map[int64]string{}
	}

	attributes := ctx.SpaceCtx().TTx.Attribute.Query().
		Where(attribute.DocumentTypeIDIn(documentTypeIDs...), attribute.IsDisabled(false)).
		WithProperty().
		WithTag().
		Order(attribute.ByID()).
		AllX(ctx)

	propertyIDs := attributePropertyIDs(attributes)
	tagGroupIDs := attributeTagGroupIDs(attributes)
	propertyValues := map[int64]map[int64]string{}
	tagValues := map[int64]map[int64][]string{}
	if len(propertyIDs) > 0 {
		propertyValues = fileTableProperties(ctx, fileIDs, propertyIDs)
	}
	if len(tagGroupIDs) > 0 {
		tagValues = fileTableTagValuesByGroup(ctx, fileIDs, tagGroupIDs)
	}

	attributesByDocumentType := map[int64][]*enttenant.Attribute{}
	for _, attributex := range attributes {
		attributesByDocumentType[attributex.DocumentTypeID] = append(attributesByDocumentType[attributex.DocumentTypeID], attributex)
	}

	result := map[int64]string{}
	for _, filex := range files {
		values := make([]string, 0, len(attributesByDocumentType[filex.DocumentTypeID]))
		for _, attributex := range attributesByDocumentType[filex.DocumentTypeID] {
			attributeValue := fileTableAttributeValueFor(
				filex.ID,
				attributex,
				propertyValues,
				tagValues,
			)
			if attributeValue.value == "" {
				continue
			}
			values = append(values, attributeValue.label+": "+attributeValue.value)
		}
		result[filex.ID] = strings.Join(values, " · ")
	}

	return result
}

func fileTableAttributeValueFor(
	fileID int64,
	attributex *enttenant.Attribute,
	propertyValues map[int64]map[int64]string,
	tagValues map[int64]map[int64][]string,
) *fileTableAttributeValue {
	if attributex.Type == attributetype.Field && attributex.Edges.Property != nil {
		return &fileTableAttributeValue{
			label: attributex.Edges.Property.Name,
			value: propertyValues[fileID][attributex.PropertyID],
		}
	}
	if attributex.Type == attributetype.Tag {
		return &fileTableAttributeValue{
			label: attributex.Name,
			value: strings.Join(tagValues[fileID][attributex.TagID], ", "),
		}
	}
	return &fileTableAttributeValue{}
}

func fileDocumentTypeIDs(files []*enttenant.File) []int64 {
	ids := make([]int64, 0, len(files))
	for _, filex := range files {
		if filex.DocumentTypeID == 0 {
			continue
		}
		ids = append(ids, filex.DocumentTypeID)
	}
	sort.Slice(ids, func(qi int, qj int) bool { return ids[qi] < ids[qj] })
	return slicesCompact(ids)
}

func attributePropertyIDs(attributes []*enttenant.Attribute) []int64 {
	ids := make([]int64, 0, len(attributes))
	for _, attributex := range attributes {
		if attributex.Type != attributetype.Field || attributex.PropertyID == 0 {
			continue
		}
		ids = append(ids, attributex.PropertyID)
	}
	sort.Slice(ids, func(qi int, qj int) bool { return ids[qi] < ids[qj] })
	return slicesCompact(ids)
}

func attributeTagGroupIDs(attributes []*enttenant.Attribute) []int64 {
	ids := make([]int64, 0, len(attributes))
	for _, attributex := range attributes {
		if attributex.Type != attributetype.Tag || attributex.TagID == 0 {
			continue
		}
		ids = append(ids, attributex.TagID)
	}
	sort.Slice(ids, func(qi int, qj int) bool { return ids[qi] < ids[qj] })
	return slicesCompact(ids)
}

func (qq *FilesListPartial) fileTablePropertyColumns(
	ctx ctxx.Context,
	selectedPropertyIDs []int64,
) []*enttenant.Property {
	if len(selectedPropertyIDs) == 0 {
		return []*enttenant.Property{}
	}
	return ctx.SpaceCtx().TTx.Property.Query().
		Where(property.IDIn(selectedPropertyIDs...)).
		Order(property.ByName()).
		AllX(ctx)
}

func (qq *FilesListPartial) fileTableTagGroupColumns(
	ctx ctxx.Context,
	selectedTagGroupIDs []int64,
) []*enttenant.Tag {
	if len(selectedTagGroupIDs) == 0 {
		return []*enttenant.Tag{}
	}
	return ctx.SpaceCtx().TTx.Tag.Query().
		Where(tag.IDIn(selectedTagGroupIDs...), tag.TypeEQ(tagtype.Group)).
		Order(tag.ByName()).
		AllX(ctx)
}

func fileIDs(files []*enttenant.File) []int64 {
	ids := make([]int64, 0, len(files))
	for _, filex := range files {
		if filex.IsDirectory {
			continue
		}
		ids = append(ids, filex.ID)
	}
	return ids
}

func propertyIDs(properties []*enttenant.Property) []int64 {
	ids := make([]int64, 0, len(properties))
	for _, propertyx := range properties {
		ids = append(ids, propertyx.ID)
	}
	return ids
}

func tagIDs(tags []*enttenant.Tag) []int64 {
	ids := make([]int64, 0, len(tags))
	for _, tagx := range tags {
		ids = append(ids, tagx.ID)
	}
	return ids
}

func fileTableDocumentTypes(ctx ctxx.Context, files []*enttenant.File) map[int64]string {
	documentTypeIDs := []int64{}
	for _, filex := range files {
		if filex.DocumentTypeID != 0 {
			documentTypeIDs = append(documentTypeIDs, filex.DocumentTypeID)
		}
	}
	sort.Slice(documentTypeIDs, func(qi int, qj int) bool { return documentTypeIDs[qi] < documentTypeIDs[qj] })
	documentTypeIDs = slicesCompact(documentTypeIDs)
	if len(documentTypeIDs) == 0 {
		return map[int64]string{}
	}
	documentTypes := ctx.SpaceCtx().TTx.DocumentType.Query().Where(documenttype.IDIn(documentTypeIDs...)).AllX(ctx)
	result := map[int64]string{}
	for _, documentTypex := range documentTypes {
		result[documentTypex.ID] = documentTypex.Name
	}
	return result
}

func fileTableSizes(ctx ctxx.Context, fileIDs []int64) map[int64]string {
	versions := ctx.SpaceCtx().TTx.FileVersion.Query().
		Where(fileversion.FileIDIn(fileIDs...)).
		WithStoredFile().
		Order(fileversion.ByFileID(sql.OrderAsc()), fileversion.ByVersionNumber(sql.OrderDesc())).
		AllX(ctx)
	result := map[int64]string{}
	for _, version := range versions {
		if _, found := result[version.FileID]; found {
			continue
		}
		if version.Edges.StoredFile == nil {
			continue
		}
		result[version.FileID] = fileutil.FormatSize(version.Edges.StoredFile.Size)
	}
	return result
}

func fileTableTags(ctx ctxx.Context, fileIDs []int64) map[int64][]string {
	assignments := ctx.SpaceCtx().TTx.TagAssignment.Query().
		Where(tagassignment.FileIDIn(fileIDs...)).
		WithTag(func(query *enttenant.TagQuery) {
			query.Order(tag.ByName())
		}).
		AllX(ctx)
	result := map[int64][]string{}
	for _, assignment := range assignments {
		if assignment.Edges.Tag == nil {
			continue
		}
		result[assignment.FileID] = append(result[assignment.FileID], assignment.Edges.Tag.Name)
	}
	for fileID := range result {
		sort.Strings(result[fileID])
	}
	return result
}

func fileTableTagValuesByGroup(
	ctx ctxx.Context,
	fileIDs []int64,
	tagGroupIDs []int64,
) map[int64]map[int64][]string {
	assignments := ctx.SpaceCtx().TTx.TagAssignment.Query().
		Where(
			tagassignment.FileIDIn(fileIDs...),
			tagassignment.HasTagWith(tag.GroupIDIn(tagGroupIDs...)),
		).
		WithTag().
		AllX(ctx)
	result := map[int64]map[int64][]string{}
	for _, assignment := range assignments {
		if assignment.Edges.Tag == nil {
			continue
		}
		if result[assignment.FileID] == nil {
			result[assignment.FileID] = map[int64][]string{}
		}
		result[assignment.FileID][assignment.Edges.Tag.GroupID] = append(
			result[assignment.FileID][assignment.Edges.Tag.GroupID],
			assignment.Edges.Tag.Name,
		)
	}
	for fileID := range result {
		for tagGroupID := range result[fileID] {
			sort.Strings(result[fileID][tagGroupID])
		}
	}
	return result
}

func fileTableProperties(ctx ctxx.Context, fileIDs []int64, propertyIDs []int64) map[int64]map[int64]string {
	assignments := ctx.SpaceCtx().TTx.FilePropertyAssignment.Query().
		Where(
			filepropertyassignment.FileIDIn(fileIDs...),
			filepropertyassignment.PropertyIDIn(propertyIDs...),
		).
		WithProperty().
		AllX(ctx)
	result := map[int64]map[int64]string{}
	for _, assignment := range assignments {
		if assignment.Edges.Property == nil {
			continue
		}
		if result[assignment.FileID] == nil {
			result[assignment.FileID] = map[int64]string{}
		}
		result[assignment.FileID][assignment.PropertyID] = fileTablePropertyValue(ctx, assignment)
	}
	return result
}

func fileTablePropertyValue(ctx ctxx.Context, assignment *enttenant.FilePropertyAssignment) string {
	switch assignment.Edges.Property.Type {
	case fieldtype.Text:
		return assignment.TextValue
	case fieldtype.Number:
		return fmt.Sprintf("%d", assignment.NumberValue)
	case fieldtype.Money:
		return fmt.Sprintf("%.2f", float64(assignment.NumberValue)/100.0)
	case fieldtype.Date:
		if assignment.DateValue.IsZero() {
			return ""
		}
		return assignment.DateValue.String(ctx.MainCtx().LanguageBCP47)
	case fieldtype.Checkbox:
		if assignment.BoolValue {
			return wx.T("Yes").String(ctx)
		}
		return wx.T("No").String(ctx)
	default:
		return ""
	}
}

func slicesCompact(ids []int64) []int64 {
	if len(ids) == 0 {
		return ids
	}
	writeIndex := 1
	for readIndex := 1; readIndex < len(ids); readIndex++ {
		if ids[readIndex] != ids[readIndex-1] {
			ids[writeIndex] = ids[readIndex]
			writeIndex++
		}
	}
	return ids[:writeIndex]
}
