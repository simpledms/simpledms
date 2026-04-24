package file

import (
	"errors"
	"log"
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/util/e"
)

type FileVersionFromInboxService struct{}

func NewFileVersionFromInboxService() *FileVersionFromInboxService {
	return &FileVersionFromInboxService{}
}

func (qq *FileVersionFromInboxService) MergeFromInbox(
	ctx ctxx.Context,
	sourceFilePublicID string,
	targetFilePublicID string,
) (*FileDTO, error) {
	if sourceFilePublicID == "" || targetFilePublicID == "" {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source and target files are required.")
	}

	repos := NewEntSpaceFileRepositoryFactory().ForSpaceX(ctx)

	sourceFile, err := repos.Read.FileByPublicID(ctx, sourceFilePublicID)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source and target files are required.")
		}
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not read source file.")
	}

	targetFile, err := repos.Read.FileByPublicID(ctx, targetFilePublicID)
	if err != nil {
		if enttenant.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source and target files are required.")
		}
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not read target file.")
	}

	if sourceFile.ID == targetFile.ID {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source and target must be different files.")
	}

	if !sourceFile.IsInInbox {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File must be in inbox.")
	}

	if sourceFile.IsDirectory || targetFile.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot merge directories.")
	}

	if !sourceFile.DeletedAt.IsZero() {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Source file is deleted.")
	}

	err = repos.Write.MergeInboxFileVersion(ctx, sourceFile, targetFile)
	if err != nil {
		var httpErr *e.HTTPError
		if errors.As(err, &httpErr) {
			return nil, err
		}
		log.Println(err)
		return nil, e.NewHTTPErrorf(http.StatusInternalServerError, "Could not merge inbox file.")
	}

	return repos.Read.FileByIDX(ctx, targetFile.ID), nil
}
