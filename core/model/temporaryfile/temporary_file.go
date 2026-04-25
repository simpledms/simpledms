package temporaryfile

import (
	"log"
	"net/http"

	securejoin "github.com/cyphar/filepath-securejoin"

	"github.com/simpledms/simpledms/core/util/e"
	"github.com/simpledms/simpledms/core/util/fileutil"
	"github.com/simpledms/simpledms/db/entmain"
)

type TemporaryFile struct {
	Data *entmain.TemporaryFile // TODO make privat?
}

func NewTemporaryFile(data *entmain.TemporaryFile) *TemporaryFile {
	return &TemporaryFile{
		Data: data,
	}
}

func (qq *TemporaryFile) ObjectNameWithPrefix() (string, error) {
	storagePath := qq.Data.StoragePath
	storageFilename := qq.Data.StorageFilename

	path, err := securejoin.SecureJoin(storagePath, storageFilename)
	if err != nil {
		log.Println(err)
		return "", e.NewHTTPErrorf(http.StatusInternalServerError, "")
	}

	return path, nil
}

func (qq *TemporaryFile) SizeString() string {
	return fileutil.FormatSize(qq.Data.Size)
	// return fmt.Sprintf("%.f kB", math.Ceil(float64(qq.Data.Size)/1014))
}
