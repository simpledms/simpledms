package tenantdatamigration

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/db/enttenant/tenantdatamigration"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/util"
)

const metadataPublicIDBatchSize = 100

const (
	metadataPublicIDTag          = "tag"
	metadataPublicIDProperty     = "property"
	metadataPublicIDDocumentType = "document-type"
)

var metadataPublicIDMigrationKeys = []string{
	"metadata-public-id-tag-v1",
	"metadata-public-id-property-v1",
	"metadata-public-id-document-type-v1",
}

type MetadataPublicIDMigration struct {
	kind string
}

func NewMetadataPublicIDMigrations() []Migration {
	return []Migration{
		&MetadataPublicIDMigration{kind: metadataPublicIDTag},
		&MetadataPublicIDMigration{kind: metadataPublicIDProperty},
		&MetadataPublicIDMigration{kind: metadataPublicIDDocumentType},
	}
}

func (qq *MetadataPublicIDMigration) Key() string {
	switch qq.kind {
	case metadataPublicIDTag:
		return metadataPublicIDMigrationKeys[0]
	case metadataPublicIDProperty:
		return metadataPublicIDMigrationKeys[1]
	case metadataPublicIDDocumentType:
		return metadataPublicIDMigrationKeys[2]
	default:
		panic("unknown metadata public ID migration")
	}
}

func (qq *MetadataPublicIDMigration) RunBatch(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	state *enttenant.TenantDataMigration,
) (*BatchResult, error) {
	ids, table, err := qq.pendingIDs(ctx, tenantDB, state.Cursor)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return NewBatchResult(state.Cursor, true, nil), nil
	}

	publicIDs := make(map[int64]entx.CIText, len(ids))
	for _, id := range ids {
		publicIDs[id] = entx.NewCIText(util.NewPublicID())
	}
	return NewBatchResult(
		ids[len(ids)-1],
		len(ids) < metadataPublicIDBatchSize,
		func(ctx context.Context, tx *enttenant.Tx) error {
			for _, id := range ids {
				var result sql.Result
				query := fmt.Sprintf(
					"UPDATE `%s` SET `public_id` = ? WHERE `id` = ? AND `public_id` IS NULL",
					table,
				)
				if err := tx.Client().Driver().Exec(ctx, query, []any{publicIDs[id], id}, &result); err != nil {
					return err
				}
			}
			return nil
		},
	), nil
}

func (qq *MetadataPublicIDMigration) pendingIDs(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	cursor int64,
) ([]int64, string, error) {
	switch qq.kind {
	case metadataPublicIDTag:
		ids, err := tenantDB.ReadOnlyConn.Tag.Query().Where(
			tag.IDGT(cursor),
			tag.PublicIDIsNil(),
		).Order(tag.ByID()).Limit(metadataPublicIDBatchSize).IDs(ctx)
		return ids, tag.Table, err
	case metadataPublicIDProperty:
		ids, err := tenantDB.ReadOnlyConn.Property.Query().Where(
			property.IDGT(cursor),
			property.PublicIDIsNil(),
		).Order(property.ByID()).Limit(metadataPublicIDBatchSize).IDs(ctx)
		return ids, property.Table, err
	case metadataPublicIDDocumentType:
		ids, err := tenantDB.ReadOnlyConn.DocumentType.Query().Where(
			documenttype.IDGT(cursor),
			documenttype.PublicIDIsNil(),
		).Order(documenttype.ByID()).Limit(metadataPublicIDBatchSize).IDs(ctx)
		return ids, documenttype.Table, err
	default:
		return nil, "", fmt.Errorf("unknown metadata public ID migration %q", qq.kind)
	}
}

func MetadataPublicIDsReady(ctx context.Context, client *enttenant.Client) (bool, error) {
	count, err := client.TenantDataMigration.Query().Where(
		tenantdatamigration.KeyIn(metadataPublicIDMigrationKeys...),
		tenantdatamigration.CompletedAtNotNil(),
	).Count(ctx)
	return count == len(metadataPublicIDMigrationKeys), err
}
