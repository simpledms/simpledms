package file

import "github.com/simpledms/simpledms/db/enttenant"

func entFileToDTO(filex *enttenant.File) *FileDTO {
	if filex == nil {
		return nil
	}

	return &FileDTO{
		ID:             filex.ID,
		PublicID:       filex.PublicID.String(),
		ParentID:       filex.ParentID,
		SpaceID:        filex.SpaceID,
		Name:           filex.Name,
		Notes:          filex.Notes,
		IsDirectory:    filex.IsDirectory,
		IsInInbox:      filex.IsInInbox,
		DocumentTypeID: filex.DocumentTypeID,
		CreatedAt:      filex.CreatedAt,
		ModifiedAt:     filex.ModifiedAt,
		DeletedAt:      filex.DeletedAt,
		OcrContent:     filex.OcrContent,
		OcrSuccessAt:   filex.OcrSuccessAt,
	}
}

func entFileToWithChildrenDTO(filex *enttenant.File) *FileWithChildrenDTO {
	if filex == nil {
		return nil
	}

	fileDTO := entFileToDTO(filex)

	var childDirectoryCount int64
	var childFileCount int64
	for _, child := range filex.Edges.Children {
		if child.IsDirectory {
			childDirectoryCount++
			continue
		}
		childFileCount++
	}

	return &FileWithChildrenDTO{
		FileDTO:             *fileDTO,
		ChildDirectoryCount: childDirectoryCount,
		ChildFileCount:      childFileCount,
	}
}

func entFileToWithParentDTO(filex *enttenant.File) *FileWithParentDTO {
	if filex == nil {
		return nil
	}

	return &FileWithParentDTO{
		FileDTO: *entFileToDTO(filex),
		Parent:  entFileToDTO(filex.Edges.Parent),
	}
}
