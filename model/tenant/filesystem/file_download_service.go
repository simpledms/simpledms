package filesystem

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/fileversion"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/mimetypex"
)

// MaxDownloadChunkBytes bounds the decoded content returned by a native download call.
const MaxDownloadChunkBytes = 1024 * 1024

// FileDownloadService reads a scoped, immutable version through the canonical storage pipeline.
type FileDownloadService struct {
	fileSystem *S3FileSystem
}

// NewFileDownloadService uses the same storage reader as browser downloads.
func NewFileDownloadService(fileSystem *S3FileSystem) *FileDownloadService {
	return &FileDownloadService{
		fileSystem: fileSystem,
	}
}

// Read returns one bounded plaintext range of a scoped file version.
func (qq *FileDownloadService) Read(
	ctx ctxx.Context,
	publicID string,
	versionNumber int,
	offset int64,
	length int,
) (*FileDownloadResult, error) {
	if publicID == "" || len(publicID) > 100 || versionNumber < 0 ||
		offset < 0 || offset > 1<<53-1 || length < 1 || length > MaxDownloadChunkBytes {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid download range.")
	}
	if offset > 0 && versionNumber == 0 {
		return nil, e.NewHTTPErrorf(
			http.StatusBadRequest, "A version number is required for download continuation.",
		)
	}
	filex, err := filemodel.NewFileReader().Get(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if filex.IsDirectory {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "File is a folder.")
	}
	query := filex.QueryFileVersions().WithStoredFile()
	if versionNumber > 0 {
		query.Where(fileversion.VersionNumber(versionNumber))
	}
	version, err := query.Order(fileversion.ByVersionNumber(sql.OrderDesc())).First(ctx)
	if err != nil {
		log.Println(err)
		if enttenant.IsNotFound(err) {
			return nil, e.NewHTTPErrorf(http.StatusNotFound, "File version not found.")
		}
		return nil, err
	}
	stored := version.Edges.StoredFile
	if stored == nil {
		return nil, e.NewHTTPErrorf(http.StatusNotFound, "File version not found.")
	}
	if stored.Size < 0 || stored.Size > 1<<53-1 || offset > stored.Size {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid download range.")
	}
	reader, err := qq.fileSystem.OpenFile(ctx, storedfilemodel.NewStoredFile(stored))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			log.Println(err)
		}
	}()
	// Encrypted, gzip-compressed storage cannot seek in plaintext. Reopen the pinned version
	// and discard its prefix; the response stays bounded even though later ranges cost more I/O.
	if _, err := io.CopyN(io.Discard, reader, offset); err != nil {
		log.Println(err)
		return nil, fmt.Errorf("read download prefix: %w", err)
	}
	count := min(int64(length), stored.Size-offset)
	content := make([]byte, int(count))
	if _, err := io.ReadFull(reader, content); err != nil {
		log.Println(err)
		return nil, fmt.Errorf("read download content: %w", err)
	}
	hasMore := offset+count < stored.Size
	if !hasMore {
		var extra [1]byte
		n, err := reader.Read(extra[:])
		if n != 0 || err != io.EOF {
			log.Printf("download did not end at the recorded size: %v", err)
			return nil, fmt.Errorf("download did not end at the recorded size")
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &FileDownloadResult{
		FilePublicID:  filex.PublicID.String(),
		Filename:      filex.Name,
		VersionNumber: version.VersionNumber,
		MIMEType:      mimetypex.Resolve(stored.MimeType, stored.Filename),
		Size:          stored.Size,
		ContentSHA256: stored.ContentSha256,
		Offset:        offset,
		Content:       content,
		HasMore:       hasMore,
	}, nil
}
