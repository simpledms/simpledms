package filelistpreference

import "slices"

type FileListPreferences struct {
	ViewMode       FileListViewMode                `json:"view_mode,omitempty"`
	BuiltInColumns []FileListColumn                `json:"built_in_columns,omitempty"`
	SpaceColumns   map[string]SpaceFileListColumns `json:"space_columns,omitempty"`
}

func DefaultFileListPreferences() FileListPreferences {
	preferences := FileListPreferences{
		ViewMode:       FileListViewModeList,
		BuiltInColumns: DefaultFileListColumns(),
		SpaceColumns:   map[string]SpaceFileListColumns{},
	}
	preferences.Normalize()
	return preferences
}

func NewFileListPreferences() *FileListPreferences {
	preferences := DefaultFileListPreferences()
	return &preferences
}

func NewFileListPreferencesFromValue(value FileListPreferences) *FileListPreferences {
	value.Normalize()
	return &value
}

func (qq *FileListPreferences) Normalize() {
	if !qq.ViewMode.IsValid() {
		qq.ViewMode = FileListViewModeList
	}
	qq.BuiltInColumns = normalizeFileListColumns(qq.BuiltInColumns)
	if len(qq.BuiltInColumns) == 0 {
		qq.BuiltInColumns = DefaultFileListColumns()
	}
	if qq.SpaceColumns == nil {
		qq.SpaceColumns = map[string]SpaceFileListColumns{}
	}
}

func (qq *FileListPreferences) IsTable() bool {
	return qq.ViewMode == FileListViewModeTable
}

func (qq *FileListPreferences) SetViewMode(viewMode FileListViewMode) {
	if !viewMode.IsValid() {
		return
	}
	qq.ViewMode = viewMode
}

func (qq *FileListPreferences) HasBuiltInColumn(column FileListColumn) bool {
	return slices.Contains(qq.BuiltInColumns, column)
}

func (qq *FileListPreferences) ToggleBuiltInColumn(column FileListColumn) {
	if !column.IsValid() {
		return
	}

	for qi, existingColumn := range qq.BuiltInColumns {
		if existingColumn == column {
			qq.BuiltInColumns = slices.Delete(qq.BuiltInColumns, qi, qi+1)
			qq.Normalize()
			return
		}
	}

	qq.BuiltInColumns = append(qq.BuiltInColumns, column)
	qq.Normalize()
}

func (qq *FileListPreferences) SpaceColumnsFor(spaceID string) *SpaceFileListColumns {
	qq.Normalize()
	spaceColumns := qq.SpaceColumns[spaceID]
	return &spaceColumns
}

func (qq *FileListPreferences) SetSpaceTags(spaceID string, showTags bool) {
	if spaceID == "" {
		return
	}
	qq.Normalize()
	spaceColumns := qq.SpaceColumns[spaceID]
	spaceColumns.ShowTags = showTags
	qq.SpaceColumns[spaceID] = spaceColumns
}

func (qq *FileListPreferences) ToggleSpacePropertyID(spaceID string, propertyID int64) {
	if spaceID == "" || propertyID <= 0 {
		return
	}
	qq.Normalize()
	spaceColumns := qq.SpaceColumns[spaceID]
	spaceColumns.TogglePropertyID(propertyID)
	qq.SpaceColumns[spaceID] = spaceColumns
}

func (qq *FileListPreferences) ToggleSpaceTagGroupID(spaceID string, tagGroupID int64) {
	if spaceID == "" || tagGroupID <= 0 {
		return
	}
	qq.Normalize()
	spaceColumns := qq.SpaceColumns[spaceID]
	spaceColumns.ToggleTagGroupID(tagGroupID)
	qq.SpaceColumns[spaceID] = spaceColumns
}

func normalizeFileListColumns(columns []FileListColumn) []FileListColumn {
	validColumns := make([]FileListColumn, 0, len(columns))
	seenColumns := map[FileListColumn]struct{}{}
	for _, column := range columns {
		if !column.IsValid() {
			continue
		}
		if _, found := seenColumns[column]; found {
			continue
		}
		seenColumns[column] = struct{}{}
		validColumns = append(validColumns, column)
	}
	return validColumns
}
