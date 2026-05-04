package filelistpreference

type FileListViewMode string

const (
	FileListViewModeList  FileListViewMode = "list"
	FileListViewModeTable FileListViewMode = "table"
)

func FileListViewModeString(value string) FileListViewMode {
	viewMode := FileListViewMode(value)
	if viewMode.IsValid() {
		return viewMode
	}
	return FileListViewModeList
}

func (qq FileListViewMode) IsValid() bool {
	return qq == FileListViewModeList || qq == FileListViewModeTable
}

func (qq FileListViewMode) String() string {
	return string(qq)
}
