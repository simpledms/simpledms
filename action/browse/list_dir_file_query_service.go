package browse

import (
	"slices"

	"entgo.io/ent/dialect/sql"

	"github.com/marcobeierer/go-core/db/entx"

	"github.com/marcobeierer/go-core/util/sqlutil"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/filesearch"
	"github.com/simpledms/simpledms/db/enttenant/resolvedtagassignment"
)

type listDirApplyPropertyFilterFunc func(
	ctx ctxx.Context,
	query *enttenant.FileQuery,
	state *ListDirPartialState,
) *enttenant.FileQuery

type ListDirFileQueryResult struct {
	CurrentDir           *enttenant.File
	Children             []*enttenant.File
	HasMore              bool
	ChildParentFullPaths map[int64]string
}

type ListDirFileQueryService struct {
	infra *common.Infra
}

func NewListDirFileQueryService(infra *common.Infra) *ListDirFileQueryService {
	return &ListDirFileQueryService{
		infra: infra,
	}
}

func (qq *ListDirFileQueryService) Query(
	ctx ctxx.Context,
	state *ListDirPartialState,
	data *ListDirPartialData,
	offset int,
	pageSize int,
	applyPropertyFilter listDirApplyPropertyFilterFunc,
) *ListDirFileQueryResult {
	// necessary to prevent putting escaped search query into search query field on re-rendering
	state.searchQueryRaw = state.SearchQuery
	// FIXME should be automatically applied on parsing state
	state.SearchQuery = sqlutil.FTSSafeAndQuery(state.SearchQuery, 300)

	// TODO find a better solution; not really robust
	if ctx.SpaceCtx().Space.IsFolderMode && state.DocumentTypeID == 0 && len(state.CheckedTagIDs) == 0 && state.SearchQuery == "" {
		state.IsRecursive = false
		state.HideDirectories = false
	} else {
		state.IsRecursive = true
	}

	var currentDir *enttenant.File
	if data.CurrentDirID == "" {
		currentDir = ctx.SpaceCtx().SpaceRootDir()
		data.CurrentDirID = currentDir.PublicID.String()
	}
	if currentDir == nil {
		currentDir = ctx.SpaceCtx().Space.QueryFiles().Where(file.PublicID(entx.NewCIText(data.CurrentDirID))).OnlyX(ctx)
	}

	// TODO sort by relevance
	searchResultQuery := ctx.AppCtx().TTx.File.Query().
		WithParent().
		WithChildren(). // necessary to count children
		Where(func(qs *sql.Selector) {
			// subquery to select all files in search scope
			if !state.IsRecursive {
				qs.Where(sql.EQ(qs.C(file.FieldParentID), currentDir.ID))
			} else {
				qs.Where(qq.descendantScopePredicate(qs.C(file.FieldID), currentDir.ID, ctx.SpaceCtx().Space.ID))
			}

			if len(state.ListFilterTagsPartialState.CheckedTagIDs) > 0 {
				resolvedTagAssignmentTable := sql.Table(resolvedtagassignment.Table)
				qs.Where(
					sql.Exists(
						sql.Select(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID)).
							From(resolvedTagAssignmentTable).
							Where(
								sql.And(
									// stange behavior if sql.EQ is used instead of sql.ColumnsEQ:
									// executing the query from debugger manually would work, but not via
									// ent because column name (files.id) is passed in as argument for the
									// prepared statement
									sql.ColumnsEQ(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID), qs.C(file.FieldID)),
									sql.InInts(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldTagID), state.ListFilterTagsPartialState.CheckedTagIDs...),
								),
							).
							GroupBy(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID)).
							Having(sql.EQ(sql.Count(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID)), len(state.ListFilterTagsPartialState.CheckedTagIDs))),
					),
				)
			}
		})

	searchResultQuery = searchResultQuery.Where(file.IsInInbox(false))

	// searchResultQuery = searchResultQuery.Where(file.HasSpaceAssignmentWith(spacefileassignment.SpaceID(ctx.SpaceCtx().Space.ID)))
	searchResultQuery = searchResultQuery.Where(file.SpaceID(ctx.SpaceCtx().Space.ID))

	if state.DocumentTypeID != 0 {
		searchResultQuery = searchResultQuery.Where(file.DocumentTypeID(state.DocumentTypeID))
	}

	if state.SearchQuery != "" {
		// TODO give filename a higher priority?
		searchResultQuery.Where(
			func(qs *sql.Selector) {
				fileSearchTable := sql.Table(filesearch.Table)

				qs.Where(
					sql.In(qs.C(file.FieldID),
						sql.Select(fileSearchTable.C(filesearch.FieldRowid)).From(fileSearchTable).
							Where(
								sql.And(
									sql.EQ(fileSearchTable.C(filesearch.FieldFileSearches), state.SearchQuery),
									sql.LT(fileSearchTable.C(filesearch.FieldRank), 0),
								),
							).
							OrderBy(fileSearchTable.C(filesearch.FieldRank)),
					),
				)
			},
		)
	}

	// TODO use filesearch view instead and order by rank?
	searchResultQuery = searchResultQuery.Order(file.ByIsDirectory(sql.OrderDesc()), file.ByName())

	if state.HideDirectories && state.HideFiles {
		// do nothing // TODO find a better solution (radio button?)
		searchResultQuery = searchResultQuery.Where(file.And(file.IsDirectory(false), file.IsDirectory(true)))
	} else if state.HideDirectories {
		searchResultQuery = searchResultQuery.Where(file.IsDirectory(false))
	} else if state.HideFiles {
		searchResultQuery = searchResultQuery.Where(file.IsDirectory(true))
	}

	searchResultQuery = applyPropertyFilter(ctx, searchResultQuery, state)

	children := searchResultQuery.Offset(offset).Limit(pageSize + 1).AllX(ctx)
	hasMore := len(children) > pageSize
	if hasMore {
		// conditional necessary to prevent out of bounce access
		children = children[:pageSize]
	}

	// get parent directory full paths for breadcrumbs...
	var childParentFullPaths map[int64]string
	if state.IsRecursive {
		var childParentIDs []int64
		for _, child := range children {
			childParentIDs = append(childParentIDs, child.ParentID)
		}
		slices.Sort(childParentIDs) // necessary for compact to work?
		childParentIDs = slices.Compact(childParentIDs)
		childParentFullPaths = qq.infra.FileSystem().FileTree().FullPathsByFileIDX(ctx, childParentIDs)
	}

	return &ListDirFileQueryResult{
		CurrentDir:           currentDir,
		Children:             children,
		HasMore:              hasMore,
		ChildParentFullPaths: childParentFullPaths,
	}
}

func (qq *ListDirFileQueryService) descendantScopePredicate(
	fileColumn string,
	rootID int64,
	spaceID int64,
) *sql.Predicate {
	return sql.In(fileColumn, qq.infra.FileSystem().FileTree().DescendantIDsSubQuery(rootID, spaceID))
}
