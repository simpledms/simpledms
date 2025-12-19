package model

import (
	securejoin "github.com/cyphar/filepath-securejoin"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/util/fileutil"
)

type StoredFile struct {
	Data *enttenant.StoredFile // TODO make privat?
}

func NewStoredFile(data *enttenant.StoredFile) *StoredFile {
	return &StoredFile{
		Data: data,
	}
}

func (qq *StoredFile) ObjectNameWithPrefix() (string, error) {
	if qq.Data.CopiedToFinalDestinationAt == nil {
		// fallback in case it is not moved to final destination yet
		return qq.UnsafeTempObjectNameWithPrefix()
	}

	return qq.UnsafeFinalObjectNameWithPrefix()
}

// Unsafe because in most cases you want to use ObjectNameWithPrefix instead,
// especially when reading files
func (qq *StoredFile) UnsafeTempObjectNameWithPrefix() (string, error) {
	return securejoin.SecureJoin(qq.Data.TemporaryStoragePath, qq.Data.TemporaryStorageFilename)

}

// Unsafe because in most cases you want to use ObjectNameWithPrefix instead,
// especially when reading files
func (qq *StoredFile) UnsafeFinalObjectNameWithPrefix() (string, error) {
	return securejoin.SecureJoin(qq.Data.StoragePath, qq.Data.StorageFilename)
}

func (qq *StoredFile) SizeString() string {
	// return fmt.Sprintf("%.f kB", math.Ceil(float64(qq.Data.Size)/1014))
	return fileutil.FormatSize(qq.Data.Size)
}

func (qq *StoredFile) IsZIPArchive() bool {
	return qq.Data.MimeType == "application/zip"
}

func (qq *StoredFile) IsMovedToFinalDestination() bool {
	return qq.Data.CopiedToFinalDestinationAt != nil
}
