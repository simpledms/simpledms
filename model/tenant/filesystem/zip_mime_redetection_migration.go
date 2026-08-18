package filesystem

import (
	"context"
	"io"

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

	mimeTypes := make([]string, len(filex))
	for i, storedFilex := range filex {
		openedFile, err := qq.openFile(ctx, qq.identity, storedfilemodel.NewStoredFile(storedFilex))
		if err != nil {
			return nil, err
		}
		mimeType, detectErr := detectMIME(openedFile)
		closeErr := openedFile.Close()
		if detectErr != nil {
			return nil, detectErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		mimeTypes[i] = mimeType
	}

	return tenantdatamigration.NewBatchResult(
		filex[len(filex)-1].ID,
		true,
		func(ctx context.Context, tx *enttenant.Tx) error {
			for i, storedFilex := range filex {
				if err := tx.StoredFile.UpdateOneID(storedFilex.ID).SetMimeType(mimeTypes[i]).Exec(ctx); err != nil {
					return err
				}
			}
			return nil
		},
	), nil
}
