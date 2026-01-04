package filesystem

import (
	"context"
	"io"
	"time"

	"filippo.io/age"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model"
)

type FileSystem interface {
	OpenFile(ctx ctxx.Context, file *model.StoredFile) (io.ReadCloser, error)
	UnsafeOpenFile(ctx context.Context, x25519Identity *age.X25519Identity, file *model.StoredFile) (io.ReadCloser, error)
	SaveFile(
		ctx ctxx.Context,
		// fileToSave multipart.File,
		fileToSave io.Reader,
		filename string,
		isInInbox bool,
		parentDirFileID int64,
	) (*enttenant.File, error)
	SaveTemporaryFileToAccount(
		ctx ctxx.Context,
		fileToSave io.Reader,
		originalFilename string,
		uploadToken string,
		fileIndex int,
		expiresAt time.Time,
	) (*entmain.TemporaryFile, error)
	PreparePersistingTemporaryAccountFile(
		ctx ctxx.Context,
		tmpFile *entmain.TemporaryFile,
		parentDirFileID int64,
		isInInbox bool,
	) (*enttenant.File, error)
	PersistTemporaryTenantFile(
		ctx context.Context,
		tenantX25519Identity *age.X25519Identity,
		filex *enttenant.StoredFile,
	) error
	UpdateMimeType(ctx ctxx.Context, force bool, filex *model.StoredFile) (string, error)
}
