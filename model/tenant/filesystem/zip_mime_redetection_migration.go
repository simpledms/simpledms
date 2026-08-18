package filesystem

import (
	"context"
	"io"
	"log"
	"strings"

	"entgo.io/ent/dialect/sql"
	"filippo.io/age"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/storedfile"
	"github.com/simpledms/simpledms/db/sqlx"
	storedfilemodel "github.com/simpledms/simpledms/model/tenant/storedfile"
	"github.com/simpledms/simpledms/model/tenant/tenantdatamigration"
)

const ZIPMIMERedetectionMigrationKey = "zip-mime-redetection-v2"

type ZIPMIMERedetectionMigration struct {
	openFile func(context.Context, *age.X25519Identity, *storedfilemodel.StoredFile) (io.ReadCloser, error)
	identity *age.X25519Identity
}

func NewZIPMIMERedetectionMigration(
	fileSystem *S3FileSystem,
	identity *age.X25519Identity,
) *ZIPMIMERedetectionMigration {
	return newZIPMIMERedetectionMigration(fileSystem.UnsafeOpenFile, identity)
}

func newZIPMIMERedetectionMigration(
	openFile func(context.Context, *age.X25519Identity, *storedfilemodel.StoredFile) (io.ReadCloser, error),
	identity *age.X25519Identity,
) *ZIPMIMERedetectionMigration {
	return &ZIPMIMERedetectionMigration{openFile: openFile, identity: identity}
}

func (qq *ZIPMIMERedetectionMigration) Key() string {
	return ZIPMIMERedetectionMigrationKey
}

func (qq *ZIPMIMERedetectionMigration) RunBatch(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	state *enttenant.TenantDataMigration,
) (*tenantdatamigration.BatchResult, error) {
	filex, err := tenantDB.ReadOnlyConn.StoredFile.Query().
		Where(
			storedfile.IDGT(state.Cursor),
			storedfile.MimeTypeEQ("application/zip"),
			storedfile.CreatedAtLT(state.FirstStartedAt),
			storedfile.CopiedToFinalDestinationAtNotNil(),
		).
		Order(storedfile.ByID(sql.OrderAsc())).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(filex) == 0 {
		return tenantdatamigration.NewBatchResult(state.Cursor, true, nil), nil
	}

	updates := make([]mimeTypeUpdate, 0, len(filex))
	for _, storedFilex := range filex {
		openedFile, err := qq.openFile(ctx, qq.identity, storedfilemodel.NewStoredFile(storedFilex))
		if err != nil {
			if isMissingS3ObjectError(err) {
				log.Printf("skipping missing stored file %d", storedFilex.ID)
				continue
			}
			return nil, err
		}
		mimeType, detectErr := detectMIME(openedFile)
		closeErr := openedFile.Close()
		if detectErr != nil {
			if isMissingS3ObjectError(detectErr) {
				log.Printf("skipping missing stored file %d", storedFilex.ID)
				continue
			}
			return nil, detectErr
		}
		if closeErr != nil {
			if isMissingS3ObjectError(closeErr) {
				log.Printf("skipping missing stored file %d", storedFilex.ID)
				continue
			}
			return nil, closeErr
		}
		updates = append(updates, mimeTypeUpdate{storedFileID: storedFilex.ID, mimeType: mimeType})
	}

	return tenantdatamigration.NewBatchResult(
		filex[len(filex)-1].ID,
		true,
		func(ctx context.Context, tx *enttenant.Tx) error {
			for _, update := range updates {
				if err := tx.StoredFile.UpdateOneID(update.storedFileID).SetMimeType(update.mimeType).Exec(ctx); err != nil {
					return err
				}
			}
			return nil
		},
	), nil
}

type mimeTypeUpdate struct {
	storedFileID int64
	mimeType     string
}

func isMissingS3ObjectError(err error) bool {
	// age exposes MinIO's NoSuchKey response only as text while reading the object stream.
	return strings.Contains(err.Error(), "The specified key does not exist.")
}
