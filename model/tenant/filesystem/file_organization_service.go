package filesystem

import (
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/util/e"
)

// FileOrganizationService applies existing filesystem rules to scoped filed entries.
type FileOrganizationService struct {
	fileSystem *S3FileSystem
}

// NewFileOrganizationService reuses the existing filesystem for logical organization.
func NewFileOrganizationService(fileSystem *S3FileSystem) *FileOrganizationService {
	return &FileOrganizationService{
		fileSystem: fileSystem,
	}
}

// Rename changes only a filed entry's logical name.
func (qq *FileOrganizationService) Rename(
	ctx ctxx.Context, publicID, filename string,
) (*enttenant.File, error) {
	filex, err := qq.filedEntry(ctx, publicID)
	if err != nil {
		return nil, err
	}
	renamed, err := qq.fileSystem.Rename(ctx, filemodel.NewFile(filex), filename)
	if err != nil {
		return nil, err
	}
	return renamed.Data, nil
}

// Move changes location/name in the caller's transaction without completing an Inbox file.
func (qq *FileOrganizationService) Move(
	ctx ctxx.Context, publicID, destinationID, filename, newDirectoryName string,
) (*enttenant.File, error) {
	filex, err := qq.filedEntry(ctx, publicID)
	if err != nil {
		return nil, err
	}
	destination, err := filemodel.NewFileReader().Get(ctx, destinationID)
	if err != nil {
		return nil, err
	}
	moved, err := qq.fileSystem.Move(
		ctx, filemodel.NewFile(destination), filemodel.NewFile(filex), filename, newDirectoryName,
	)
	if err != nil {
		return nil, err
	}
	return moved.Data, nil
}

func (qq *FileOrganizationService) filedEntry(
	ctx ctxx.Context, publicID string,
) (*enttenant.File, error) {
	filex, err := filemodel.NewFileReader().Get(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if filex.IsInInbox {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File must be filed before organization.")
	}
	if filex.IsRootDir {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot organize the Space root folder.")
	}
	return filex, nil
}
