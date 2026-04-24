package file

import "time"

type FileDTO struct {
	ID             int64
	PublicID       string
	ParentID       int64
	SpaceID        int64
	Name           string
	Notes          string
	IsDirectory    bool
	IsInInbox      bool
	DocumentTypeID int64
	CreatedAt      time.Time
	ModifiedAt     *time.Time
	DeletedAt      time.Time
	OcrContent     string
	OcrSuccessAt   *time.Time
}
