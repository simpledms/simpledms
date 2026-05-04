package filelistpreference

type FileListColumn string

const (
	FileListColumnName         FileListColumn = "name"
	FileListColumnDocumentType FileListColumn = "document_type"
	FileListColumnMetadata     FileListColumn = "metadata"
	FileListColumnDate         FileListColumn = "date"
	FileListColumnSize         FileListColumn = "size"
)

func DefaultFileListColumns() []FileListColumn {
	return []FileListColumn{
		FileListColumnName,
		FileListColumnDocumentType,
		FileListColumnMetadata,
		FileListColumnDate,
		FileListColumnSize,
	}
}

func FileListColumnString(value string) (FileListColumn, bool) {
	column := FileListColumn(value)
	return column, column.IsValid()
}

func (qq FileListColumn) IsValid() bool {
	switch qq {
	case FileListColumnName,
		FileListColumnDocumentType,
		FileListColumnMetadata,
		FileListColumnDate,
		FileListColumnSize:
		return true
	default:
		return false
	}
}

func (qq FileListColumn) String() string {
	return string(qq)
}
