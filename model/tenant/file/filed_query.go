package file

import (
	"log"
	"net/http"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entquery"
	"github.com/simpledms/simpledms/db/enttenant"
	dbfile "github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/resolvedtagassignment"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/sqlutil"
)

const (
	filedSortNewestFirst = "newestFirst"
	filedSortOldestFirst = "oldestFirst"
	filedSortName        = "name"
	filedSortRank        = "rank"
)

// FiledQuery applies the shared filed-document filtering and ordering rules.
type FiledQuery struct{}

// NewFiledQuery creates a filed-document query service.
func NewFiledQuery() *FiledQuery {
	return &FiledQuery{}
}

// Query returns filed documents across the authorized Space. Callers own page bounds.
func (qq *FiledQuery) Query(
	ctx ctxx.Context,
	text string,
	sort string,
	tagIDs []int64,
	documentTypeID int64,
) (*enttenant.FileQuery, error) {
	query := ctx.TenantCtx().TTx.File.Query().Where(entquery.FileIsDirectory(false))
	return qq.Apply(ctx, query, sqlutil.FTSSafeAndQuery(text, 300), sort, tagIDs, documentTypeID)
}

// Apply adds the shared filed-document search, classification, and ordering semantics.
func (qq *FiledQuery) Apply(
	ctx ctxx.Context,
	query *enttenant.FileQuery,
	search string,
	sort string,
	tagIDs []int64,
	documentTypeID int64,
) (*enttenant.FileQuery, error) {
	query.Where(entquery.FileIsInInbox(false), dbfile.SpaceID(ctx.SpaceCtx().Space.ID))
	if len(tagIDs) > 0 {
		resolvedTagIDs := make([]int, 0, len(tagIDs))
		for _, tagID := range tagIDs {
			resolvedTagIDs = append(resolvedTagIDs, int(tagID))
		}
		query.Where(func(selector *sql.Selector) {
			resolved := sql.Table(resolvedtagassignment.Table)
			selector.Where(sql.Exists(
				sql.Select(resolved.C(resolvedtagassignment.FieldFileID)).
					From(resolved).
					Where(sql.And(
						sql.ColumnsEQ(
							resolved.C(resolvedtagassignment.FieldFileID),
							selector.C(dbfile.FieldID),
						),
						sql.InInts(
							resolved.C(resolvedtagassignment.FieldTagID),
							resolvedTagIDs...,
						),
					)).
					GroupBy(resolved.C(resolvedtagassignment.FieldFileID)).
					Having(sql.EQ(
						sql.Count(resolved.C(resolvedtagassignment.FieldFileID)),
						len(tagIDs),
					)),
			))
		})
	}
	if documentTypeID != 0 {
		query.Where(dbfile.DocumentTypeID(documentTypeID))
	}
	if search != "" {
		query.Where(func(selector *sql.Selector) {
			entquery.ApplyFileSearchCandidateFilter(
				selector,
				search,
				ctx.SpaceCtx().Space.ID,
				false,
			)
		})
	} else if sort == filedSortRank {
		sort = ""
	}
	switch sort {
	case filedSortRank:
		query.Order(
			entquery.OrderFileSearchRank(search, ctx.SpaceCtx().Space.ID, false),
			dbfile.ByIsDirectory(sql.OrderDesc()),
			dbfile.ByName(),
		)
	case filedSortNewestFirst:
		query.Order(
			dbfile.ByIsDirectory(sql.OrderDesc()),
			dbfile.ByCreatedAt(sql.OrderDesc()),
			dbfile.ByName(),
		)
	case filedSortOldestFirst:
		query.Order(
			dbfile.ByIsDirectory(sql.OrderDesc()),
			dbfile.ByCreatedAt(),
			dbfile.ByName(),
		)
	case "", filedSortName:
		query.Order(dbfile.ByIsDirectory(sql.OrderDesc()), dbfile.ByName())
	default:
		log.Println("invalid filed-file sort order")
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid sort order.")
	}
	return query.Order(dbfile.ByID()), nil
}
