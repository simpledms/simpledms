package file

type FileWithChildrenDTO struct {
	FileDTO
	ChildDirectoryCount int64
	ChildFileCount      int64
}
