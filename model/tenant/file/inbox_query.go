package file

import (
	"log"
	"net/http"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entquery"
	"github.com/simpledms/simpledms/db/enttenant"
	dbfile "github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/model/main/common/filesource"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/sqlutil"
)

type InboxQuery struct{}

func NewInboxQuery() *InboxQuery {
	return &InboxQuery{}
}

// Query preserves Inbox search semantics; callers must apply their page bounds.
func (qq *InboxQuery) Query(
	ctx ctxx.Context, text, sort string, sources []string,
) (*enttenant.FileQuery, error) {
	query := ctx.TenantCtx().TTx.File.Query().Where(dbfile.SpaceID(ctx.SpaceCtx().Space.ID))
	search := sqlutil.FTSSafeAndQuery(text, 300)
	if search != "" {
		query.Where(dbfile.IsInInbox(true), dbfile.IsDirectory(false))
		query.Where(func(selector *sql.Selector) {
			entquery.ApplyFileSearchCandidateFilterWithDirectory(
				selector, search, ctx.SpaceCtx().Space.ID, true, false,
			)
		})
	} else {
		query.Where(entquery.FileIsInInbox(true), entquery.FileIsDirectory(false))
		if sort == "rank" {
			sort = ""
		}
	}
	var values []filesource.FileSource
	for _, value := range sources {
		source, err := filesource.FileSourceString(value)
		if err != nil {
			log.Println("invalid Inbox source filter")
			return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid source filter.")
		}
		values = append(values, source)
	}
	if len(values) > 0 {
		query.Where(dbfile.SourceIn(values...))
	}
	switch sort {
	case "rank":
		query.Order(entquery.OrderFileSearchRankWithDirectory(
			search, ctx.SpaceCtx().Space.ID, true, false,
		), dbfile.ByCreatedAt(sql.OrderDesc()))
	case "name":
		query.Order(dbfile.ByName())
	case "oldestFirst":
		query.Order(dbfile.ByCreatedAt())
	case "", "newestFirst":
		query.Order(dbfile.ByCreatedAt(sql.OrderDesc()))
	default:
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid sort order.")
	}
	return query.Order(dbfile.ByID()), nil
}
