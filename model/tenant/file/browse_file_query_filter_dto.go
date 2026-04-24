package file

type BrowseFileQueryFilterDTO struct {
	CurrentDirPublicID string
	SearchQuery        string
	DocumentTypeID     int64
	CheckedTagIDs      []int
	HideDirectories    bool
	HideFiles          bool
	IsRecursive        bool
	Offset             int
	Limit              int
	PropertyFilters    []BrowsePropertyFilterDTO
}
