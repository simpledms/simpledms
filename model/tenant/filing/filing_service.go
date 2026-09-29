package filing

import (
	"log"
	"net/http"
	"path/filepath"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entquery"
	"github.com/simpledms/simpledms/db/enttenant"
	dbfile "github.com/simpledms/simpledms/db/enttenant/file"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/model/tenant/filesystem"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/filenamex"
)

// FilingService owns the atomic transition from Inbox to filed state.
type FilingService struct {
	fileSystem *filesystem.S3FileSystem
}

// NewFilingService creates a filing service using the existing filesystem rules.
func NewFilingService(fileSystem *filesystem.S3FileSystem) *FilingService {
	return &FilingService{fileSystem: fileSystem}
}

// CompleteInboxFile completes an Inbox document without changing its parent.
func (qq *FilingService) CompleteInboxFile(
	ctx ctxx.Context,
	fileID string,
) (*enttenant.File, error) {
	filex, err := qq.inboxDocument(ctx, fileID)
	if err != nil {
		return nil, err
	}
	filex, err = filex.Update().SetIsInInbox(false).Save(ctx)
	if err != nil {
		log.Println(err)
	}
	return filex, err
}

// FileInboxDocument moves and completes one Inbox document in the same transaction.
func (qq *FilingService) FileInboxDocument(
	ctx ctxx.Context,
	fileID string,
	destinationID string,
	filename string,
	newDirectoryName string,
) (*enttenant.File, error) {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Folder mode is not enabled.")
	}
	filex, err := qq.inboxDocument(ctx, fileID)
	if err != nil {
		return nil, err
	}
	destination, err := filemodel.NewFileReader().Get(ctx, destinationID)
	if err != nil {
		return nil, err
	}
	if !destination.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Destination is not a directory.")
	}
	if filename == "" {
		filename = filex.Name
	}
	if filepath.Clean(filename) != filename || !filenamex.IsAllowed(filename) {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid filename.")
	}
	if newDirectoryName == "" {
		conflict, err := ctx.TenantCtx().TTx.File.Query().Where(
			dbfile.SpaceID(ctx.SpaceCtx().Space.ID),
			dbfile.ParentID(destination.ID),
			dbfile.Name(filename),
			entquery.FileIsInInbox(false),
			dbfile.IDNEQ(filex.ID),
		).Exist(ctx)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		if conflict {
			return nil, e.NewHTTPErrorf(
				http.StatusBadRequest,
				"A file with this name already exists in the destination.",
			)
		}
	}
	moved, err := qq.fileSystem.Move(
		ctx,
		filemodel.NewFile(destination),
		filemodel.NewFile(filex),
		filename,
		newDirectoryName,
	)
	if err != nil {
		return nil, err
	}
	filex, err = moved.Data.Update().SetIsInInbox(false).Save(ctx)
	if err != nil {
		log.Println(err)
	}
	return filex, err
}

// CreateDirectory creates one scoped filing destination.
func (qq *FilingService) CreateDirectory(
	ctx ctxx.Context,
	parentID string,
	name string,
) (*enttenant.File, error) {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Folder mode is not enabled.")
	}
	parent, err := filemodel.NewFileReader().Get(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if !parent.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Parent is not a directory.")
	}
	directory, err := qq.fileSystem.MakeDir(ctx, parent.PublicID.String(), name)
	if err != nil {
		return nil, err
	}
	return directory.Data, nil
}

// ListDirectory lists one page of immediate, filed children.
func (qq *FilingService) ListDirectory(
	ctx ctxx.Context,
	directoryID string,
	offset int,
	limit int,
) (*enttenant.File, []*enttenant.File, bool, error) {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return nil, nil, false, e.NewHTTPErrorf(
			http.StatusBadRequest,
			"Folder mode is not enabled.",
		)
	}
	var directory *enttenant.File
	var err error
	if directoryID == "" {
		directory = ctx.SpaceCtx().SpaceRootDir()
	} else {
		directory, err = filemodel.NewFileReader().Get(ctx, directoryID)
		if err != nil {
			return nil, nil, false, err
		}
	}
	if !directory.IsDirectory {
		return nil, nil, false, e.NewHTTPErrorf(
			http.StatusBadRequest,
			"File is not a directory.",
		)
	}
	children, err := ctx.TenantCtx().TTx.File.Query().Where(
		dbfile.SpaceID(ctx.SpaceCtx().Space.ID),
		dbfile.ParentID(directory.ID),
		entquery.FileIsInInbox(false),
	).Order(
		dbfile.ByIsDirectory(sql.OrderDesc()),
		dbfile.ByName(),
		dbfile.ByID(),
	).Offset(offset).Limit(limit + 1).All(ctx)
	if err != nil {
		log.Println(err)
		return nil, nil, false, err
	}
	hasMore := len(children) > limit
	if hasMore {
		children = children[:limit]
	}
	return directory, children, hasMore, nil
}

func (qq *FilingService) inboxDocument(
	ctx ctxx.Context,
	fileID string,
) (*enttenant.File, error) {
	filex, err := filemodel.NewFileReader().Get(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if filex.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is a directory.")
	}
	if !filex.IsInInbox {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File must be in Inbox.")
	}
	return filex, nil
}
