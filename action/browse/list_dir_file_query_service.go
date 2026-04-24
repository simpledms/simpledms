package browse

import (
	"slices"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/util/sqlutil"
)

type ListDirFileQueryResult struct {
	CurrentDir           *filemodel.FileDTO
	Children             []*filemodel.FileWithChildrenDTO
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

	propertyFilters := make([]filemodel.BrowsePropertyFilterDTO, 0, len(state.PropertyValues))
	for _, propertyFilter := range state.PropertyValues {
		propertyFilters = append(propertyFilters, filemodel.BrowsePropertyFilterDTO{
			PropertyID: propertyFilter.PropertyID,
			Operator:   propertyFilter.Operator,
			Value:      propertyFilter.Value,
		})
	}

	repos := qq.infra.SpaceFileRepoFactory().ForSpaceX(ctx)
	queryResult := repos.Query.BrowseFilesX(ctx, &filemodel.BrowseFileQueryFilterDTO{
		CurrentDirPublicID: data.CurrentDirID,
		SearchQuery:        state.SearchQuery,
		DocumentTypeID:     state.DocumentTypeID,
		CheckedTagIDs:      state.ListFilterTagsPartialState.CheckedTagIDs,
		HideDirectories:    state.HideDirectories,
		HideFiles:          state.HideFiles,
		IsRecursive:        state.IsRecursive,
		Offset:             offset,
		Limit:              pageSize,
		PropertyFilters:    propertyFilters,
	})

	currentDir := queryResult.CurrentDir
	if data.CurrentDirID == "" && currentDir != nil {
		data.CurrentDirID = currentDir.PublicID
	}

	children := queryResult.Children
	hasMore := queryResult.HasMore

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
