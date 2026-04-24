package file

import (
	"log"
	"math"
	"slices"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/filesearch"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/db/enttenant/resolvedtagassignment"
	"github.com/simpledms/simpledms/db/enttenant/schema"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	"github.com/simpledms/simpledms/util/sqlutil"
	"github.com/simpledms/simpledms/util/timex"
)

type EntSpaceFileQueryRepository struct {
	spaceID int64
}

var _ FileQueryRepository = (*EntSpaceFileQueryRepository)(nil)

func NewEntSpaceFileQueryRepository(spaceID int64) *EntSpaceFileQueryRepository {
	return &EntSpaceFileQueryRepository{
		spaceID: spaceID,
	}
}

func (qq *EntSpaceFileQueryRepository) BrowseFilesX(
	ctx ctxx.Context,
	filterx *BrowseFileQueryFilterDTO,
) *BrowseFileQueryResultDTO {
	filterx = qq.nilableBrowseFilter(filterx)

	currentDir := qq.browseCurrentDirX(ctx, filterx.CurrentDirPublicID)
	searchQuery := sqlutil.FTSSafeAndQuery(filterx.SearchQuery, 300)

	searchResultQuery := ctx.TenantCtx().TTx.File.Query().
		WithParent().
		WithChildren().
		Where(func(qs *sql.Selector) {
			// subquery to select all files in search scope
			if !filterx.IsRecursive {
				qs.Where(sql.EQ(qs.C(file.FieldParentID), currentDir.ID))
			} else {
				qs.Where(qq.browseDescendantScopePredicate(qs.C(file.FieldID), currentDir.ID))
			}

			if len(filterx.CheckedTagIDs) == 0 {
				return
			}

			resolvedTagAssignmentTable := sql.Table(resolvedtagassignment.Table)
			qs.Where(
				sql.Exists(
					sql.Select(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID)).
						From(resolvedTagAssignmentTable).
						Where(
							sql.And(
								// strange behavior if sql.EQ is used instead of sql.ColumnsEQ:
								// executing the query from debugger manually would work, but not via
								// ent because column name (files.id) is passed in as argument for the
								// prepared statement
								sql.ColumnsEQ(
									resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID),
									qs.C(file.FieldID),
								),
								sql.InInts(
									resolvedTagAssignmentTable.C(resolvedtagassignment.FieldTagID),
									filterx.CheckedTagIDs...,
								),
							),
						).
						GroupBy(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID)).
						Having(
							sql.EQ(
								sql.Count(resolvedTagAssignmentTable.C(resolvedtagassignment.FieldFileID)),
								len(filterx.CheckedTagIDs),
							),
						),
				),
			)
		})

	searchResultQuery = searchResultQuery.Where(
		file.IsInInbox(false),
		file.SpaceID(qq.spaceID),
	)

	if filterx.DocumentTypeID != 0 {
		searchResultQuery = searchResultQuery.Where(file.DocumentTypeID(filterx.DocumentTypeID))
	}

	if searchQuery != "" {
		// TODO give filename a higher priority?
		searchResultQuery = searchResultQuery.Where(func(qs *sql.Selector) {
			fileSearchTable := sql.Table(filesearch.Table)
			qs.Where(
				sql.In(
					qs.C(file.FieldID),
					sql.Select(fileSearchTable.C(filesearch.FieldRowid)).
						From(fileSearchTable).
						Where(
							sql.And(
								sql.EQ(fileSearchTable.C(filesearch.FieldFileSearches), searchQuery),
								sql.LT(fileSearchTable.C(filesearch.FieldRank), 0),
							),
						).
						OrderBy(fileSearchTable.C(filesearch.FieldRank)),
				),
			)
		})
	}

	// TODO use filesearch view instead and order by rank?
	searchResultQuery = searchResultQuery.Order(file.ByIsDirectory(sql.OrderDesc()), file.ByName())

	if filterx.HideDirectories && filterx.HideFiles {
		// do nothing // TODO find a better solution (radio button?)
		searchResultQuery = searchResultQuery.Where(file.And(file.IsDirectory(false), file.IsDirectory(true)))
	} else if filterx.HideDirectories {
		searchResultQuery = searchResultQuery.Where(file.IsDirectory(false))
	} else if filterx.HideFiles {
		searchResultQuery = searchResultQuery.Where(file.IsDirectory(true))
	}

	searchResultQuery = qq.applyBrowsePropertyFilters(ctx, searchResultQuery, filterx.PropertyFilters)

	children := searchResultQuery.Offset(filterx.Offset).Limit(filterx.Limit + 1).AllX(ctx)
	hasMore := len(children) > filterx.Limit
	if hasMore {
		children = children[:filterx.Limit]
	}

	childDTOs := make([]*FileWithChildrenDTO, 0, len(children))
	for _, child := range children {
		childDTOs = append(childDTOs, entFileToWithChildrenDTO(child))
	}

	return &BrowseFileQueryResultDTO{
		CurrentDir: entFileToDTO(currentDir),
		Children:   childDTOs,
		HasMore:    hasMore,
	}
}

func (qq *EntSpaceFileQueryRepository) InboxFilesX(
	ctx ctxx.Context,
	filterx *InboxFileQueryFilterDTO,
) []*FileWithChildrenDTO {
	// LIMIT must be applied by caller
	filterx = qq.nilableInboxFilter(filterx)

	query := ctx.TenantCtx().TTx.File.Query().
		WithChildren().
		Where(
			file.SpaceID(qq.spaceID),
			file.IsInInbox(true),
		)

	if filterx.SearchQuery != "" {
		// TODO necessary if not full text search? probably not
		query = query.Where(file.NameContains(filterx.SearchQuery))
	}

	// TODO use filesearch view instead and order by rank?
	switch filterx.SortBy {
	case "name":
		query = query.Order(file.ByName())
	case "oldestFirst":
		query = query.Order(file.ByCreatedAt())
	case "newestFirst":
		fallthrough
	default:
		query = query.Order(file.ByCreatedAt(sql.OrderDesc()))
	}

	children := query.AllX(ctx)

	dtos := make([]*FileWithChildrenDTO, 0, len(children))
	for _, child := range children {
		dtos = append(dtos, entFileToWithChildrenDTO(child))
	}

	return dtos
}

func (qq *EntSpaceFileQueryRepository) TrashFilesX(ctx ctxx.Context) []*FileDTO {
	ctxWithDeleted := schema.SkipSoftDelete(ctx)
	files := ctx.TenantCtx().TTx.File.Query().
		Where(
			file.SpaceID(qq.spaceID),
			file.DeletedAtNotNil(),
			file.IsDirectory(false),
		).
		Order(file.ByDeletedAt(sql.OrderDesc())).
		AllX(ctxWithDeleted)

	dtos := make([]*FileDTO, 0, len(files))
	for _, filex := range files {
		dtos = append(dtos, entFileToDTO(filex))
	}

	return dtos
}

func (qq *EntSpaceFileQueryRepository) InboxAssignmentSuggestionDirectoriesX(
	ctx ctxx.Context,
	searchQuery string,
	limit int,
) []*FileWithChildrenDTO {
	if searchQuery == "" {
		return []*FileWithChildrenDTO{}
	}
	if limit <= 0 {
		limit = 10
	}

	destDirs := ctx.TenantCtx().TTx.File.Query().
		WithChildren().
		Where(
			file.SpaceID(qq.spaceID),
			file.IsDirectory(true),
			func(qs *sql.Selector) {
				fileSearchTable := sql.Table(filesearch.Table)

				qs.Where(
					sql.In(
						qs.C(file.FieldID),
						sql.Select(fileSearchTable.C(filesearch.FieldRowid)).
							From(fileSearchTable).
							Where(
								sql.And(
									sql.EQ(fileSearchTable.C(filesearch.FieldFileSearches), searchQuery),
									sql.LT(fileSearchTable.C(filesearch.FieldRank), 0),
								),
							).
							OrderBy(fileSearchTable.C(filesearch.FieldRank)),
					),
				)
			},
		).
		Limit(limit).
		AllX(ctx)

	dtos := make([]*FileWithChildrenDTO, 0, len(destDirs))
	for _, destDir := range destDirs {
		dtos = append(dtos, entFileToWithChildrenDTO(destDir))
	}

	return dtos
}

func (qq *EntSpaceFileQueryRepository) browseCurrentDirX(ctx ctxx.Context, currentDirPublicID string) *enttenant.File {
	if currentDirPublicID == "" {
		return ctx.SpaceCtx().SpaceRootDir()
	}

	return ctx.TenantCtx().TTx.File.Query().
		Where(
			file.PublicIDEQ(entx.NewCIText(currentDirPublicID)),
			file.SpaceID(qq.spaceID),
		).
		OnlyX(ctx)
}

func (qq *EntSpaceFileQueryRepository) nilableBrowseFilter(
	filterx *BrowseFileQueryFilterDTO,
) *BrowseFileQueryFilterDTO {
	if filterx == nil {
		return &BrowseFileQueryFilterDTO{
			Limit: 50,
		}
	}

	if filterx.Limit <= 0 {
		filterx.Limit = 50
	}

	return filterx
}

func (qq *EntSpaceFileQueryRepository) nilableInboxFilter(
	filterx *InboxFileQueryFilterDTO,
) *InboxFileQueryFilterDTO {
	if filterx == nil {
		return &InboxFileQueryFilterDTO{}
	}

	return filterx
}

func (qq *EntSpaceFileQueryRepository) browseDescendantScopePredicate(
	fileColumn string,
	rootID int64,
) *sql.Predicate {
	filesTable := sql.Table(file.Table)
	recursiveFilesTable := sql.Table(file.Table).As("f")
	recursiveDescendantsTable := sql.Table("descendants").As("d")
	descendantsTable := sql.Table("descendants")

	anchor := sql.Select(filesTable.C(file.FieldID)).
		From(filesTable).
		Where(
			sql.And(
				sql.EQ(filesTable.C(file.FieldID), rootID),
				sql.EQ(filesTable.C(file.FieldSpaceID), qq.spaceID),
				sql.IsNull(filesTable.C(file.FieldDeletedAt)),
			),
		)

	recursive := sql.Select(recursiveFilesTable.C(file.FieldID)).
		From(recursiveFilesTable).
		Join(recursiveDescendantsTable).
		On(recursiveFilesTable.C(file.FieldParentID), recursiveDescendantsTable.C("id")).
		Where(
			sql.And(
				sql.EQ(recursiveFilesTable.C(file.FieldSpaceID), qq.spaceID),
				sql.IsNull(recursiveFilesTable.C(file.FieldDeletedAt)),
			),
		)

	withDescendants := sql.WithRecursive("descendants", "id").As(anchor.UnionAll(recursive))

	return sql.In(
		fileColumn,
		sql.Select(descendantsTable.C("id")).
			From(descendantsTable).
			Where(sql.NEQ(descendantsTable.C("id"), rootID)).
			Prefix(withDescendants),
	)
}

func (qq *EntSpaceFileQueryRepository) applyBrowsePropertyFilters(
	ctx ctxx.Context,
	query *enttenant.FileQuery,
	propertyFilters []BrowsePropertyFilterDTO,
) *enttenant.FileQuery {
	propertyIDs := make([]int64, 0, len(propertyFilters))
	for _, propertyFilter := range propertyFilters {
		propertyIDs = append(propertyIDs, propertyFilter.PropertyID)
	}

	// must be sorted
	slices.Sort(propertyIDs)
	propertyIDs = slices.Compact(propertyIDs)
	if len(propertyIDs) == 0 {
		return query
	}

	propertiesx := ctx.SpaceCtx().Space.QueryProperties().Where(property.IDIn(propertyIDs...)).AllX(ctx)
	propertiesByID := make(map[int64]*enttenant.Property, len(propertiesx))
	for _, propertyx := range propertiesx {
		propertiesByID[propertyx.ID] = propertyx
	}

	for _, propertyFilter := range propertyFilters {
		propertyx, found := propertiesByID[propertyFilter.PropertyID]
		if !found {
			continue
		}

		switch propertyx.Type {
		case fieldtype.Text:
			switch propertyFilter.Operator {
			case "contains":
				query = query.Where(file.HasPropertyAssignmentWith(
					// Fold makes case insensitive
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.TextValueContainsFold(propertyFilter.Value),
				))
			case "equals":
				query = query.Where(file.HasPropertyAssignmentWith(
					// Fold makes case insensitive
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.TextValueEqualFold(propertyFilter.Value),
				))
			case "starts_with":
				query = query.Where(file.HasPropertyAssignmentWith(
					// is case insensitive
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.TextValueHasPrefix(propertyFilter.Value),
				))
			}
		case fieldtype.Number:
			value, err := strconv.Atoi(propertyFilter.Value)
			if err != nil {
				log.Println(err)
				continue
			}

			switch propertyFilter.Operator {
			case "equals":
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValue(value),
				))
			case "greater_than":
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValueGT(value),
				))
			case "less_than":
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValueLT(value),
				))
			}
		case fieldtype.Date:
			switch propertyFilter.Operator {
			case "equals":
				value, err := timex.ParseDate(propertyFilter.Value)
				if err != nil {
					log.Println(err)
					continue
				}
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.DateValue(value),
				))
			case "greater_than":
				value, err := timex.ParseDate(propertyFilter.Value)
				if err != nil {
					log.Println(err)
					continue
				}
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.DateValueGT(value),
				))
			case "less_than":
				value, err := timex.ParseDate(propertyFilter.Value)
				if err != nil {
					log.Println(err)
					continue
				}
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.DateValueLT(value),
				))
			case "between":
				startDate := ""
				endDate := ""
				if propertyFilter.Value != "" {
					parts := strings.SplitN(propertyFilter.Value, ",", 2)
					startDate = parts[0]
					if len(parts) > 1 {
						endDate = parts[1]
					}
				}

				if startDate != "" {
					value, err := timex.ParseDate(startDate)
					if err != nil {
						log.Println(err)
						continue
					}
					query = query.Where(file.HasPropertyAssignmentWith(
						filepropertyassignment.PropertyID(propertyFilter.PropertyID),
						filepropertyassignment.DateValueGTE(value),
					))
				}

				if endDate != "" {
					value, err := timex.ParseDate(endDate)
					if err != nil {
						log.Println(err)
						continue
					}
					query = query.Where(file.HasPropertyAssignmentWith(
						filepropertyassignment.PropertyID(propertyFilter.PropertyID),
						filepropertyassignment.DateValueLTE(value),
					))
				}
			}
		case fieldtype.Money:
			valueFloat, err := strconv.ParseFloat(propertyFilter.Value, 64)
			if err != nil {
				log.Println(err)
				continue
			}
			// convert to minor unit // TODO is this good enough?
			value := int(math.Round(valueFloat * 100))

			switch propertyFilter.Operator {
			case "equals":
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValue(value),
				))
			case "greater_than":
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValueGT(value),
				))
			case "less_than":
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.NumberValueLT(value),
				))
			}
		case fieldtype.Checkbox:
			value, err := strconv.ParseBool(propertyFilter.Value)
			if err != nil {
				log.Println(err)
				continue
			}
			if value {
				query = query.Where(file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(propertyFilter.PropertyID),
					filepropertyassignment.BoolValue(true),
				))
			} else {
				// file.Or ensures that also listed if no property assignment
				// FIXME doesn't return all results without assignment!
				query = query.Where(
					file.Or(
						file.HasPropertyAssignmentWith(
							filepropertyassignment.PropertyID(propertyFilter.PropertyID),
							filepropertyassignment.Or(
								filepropertyassignment.BoolValue(false),
								filepropertyassignment.BoolValueIsNil(),
							),
						),
						file.Not(file.HasPropertyAssignment()),
					),
				)
			}
		}
	}

	return query
}
