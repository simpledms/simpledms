package file

type BrowseFileQueryResultDTO struct {
	CurrentDir *FileDTO
	Children   []*FileWithChildrenDTO
	HasMore    bool
}
