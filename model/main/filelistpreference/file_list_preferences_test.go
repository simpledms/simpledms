package filelistpreference

import "testing"

func TestFileListPreferencesDefaults(t *testing.T) {
	preferences := NewFileListPreferences()

	if preferences.ViewMode != FileListViewModeList {
		t.Fatalf("expected list view mode, got %q", preferences.ViewMode)
	}
	for _, column := range DefaultFileListColumns() {
		if !preferences.HasBuiltInColumn(column) {
			t.Fatalf("expected default column %q", column)
		}
	}
}

func TestFileListPreferencesInvalidValuesFallback(t *testing.T) {
	preferences := NewFileListPreferencesFromValue(FileListPreferences{
		ViewMode:       FileListViewMode("invalid"),
		BuiltInColumns: []FileListColumn{FileListColumn("invalid")},
	})

	if preferences.ViewMode != FileListViewModeList {
		t.Fatalf("expected list view mode fallback, got %q", preferences.ViewMode)
	}
	if !preferences.HasBuiltInColumn(FileListColumnName) {
		t.Fatal("expected default columns after invalid input")
	}
}

func TestFileListPreferencesToggleColumns(t *testing.T) {
	preferences := NewFileListPreferences()
	preferences.ToggleBuiltInColumn(FileListColumnSize)
	if preferences.HasBuiltInColumn(FileListColumnSize) {
		t.Fatal("expected size column to be disabled")
	}
	preferences.ToggleBuiltInColumn(FileListColumnSize)
	if !preferences.HasBuiltInColumn(FileListColumnSize) {
		t.Fatal("expected size column to be enabled")
	}
}

func TestFileListPreferencesSpaceColumns(t *testing.T) {
	preferences := NewFileListPreferences()
	preferences.SetSpaceTags("space-a", true)
	preferences.ToggleSpacePropertyID("space-a", 42)
	preferences.ToggleSpaceTagGroupID("space-a", 7)

	spaceColumns := preferences.SpaceColumnsFor("space-a")
	if !spaceColumns.ShowTags {
		t.Fatal("expected tags to be visible")
	}
	if !spaceColumns.HasPropertyID(42) {
		t.Fatal("expected property 42 to be selected")
	}
	if !spaceColumns.HasTagGroupID(7) {
		t.Fatal("expected tag group 7 to be selected")
	}
}
