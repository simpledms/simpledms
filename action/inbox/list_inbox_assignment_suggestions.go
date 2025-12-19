package inbox

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"entgo.io/ent/dialect/sql"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/enttenant/file"
	"github.com/simpledms/simpledms/enttenant/fileinfo"
	"github.com/simpledms/simpledms/enttenant/filesearch"
	wx "github.com/simpledms/simpledms/ui/widget"
)

type ListInboxAssignmentSuggestionsData struct {
	FileID int64
}

// TODO or InboxAssignmentSuggestionsList
type ListInboxAssignmentSuggestions struct {
	infra   *common.Infra
	actions *Actions
	// *actionx.Config
}

func NewListInboxAssignmentSuggestions(infra *common.Infra, actions *Actions) *ListInboxAssignmentSuggestions {
	return &ListInboxAssignmentSuggestions{
		infra:   infra,
		actions: actions,
		/*Config: actionx.NewConfig(
			"/list-inbox-assignment-suggestions",
			true,
		),*/
	}
}

func (qq *ListInboxAssignmentSuggestions) Data(fileID int64) *ListInboxAssignmentSuggestionsData {
	return &ListInboxAssignmentSuggestionsData{
		FileID: fileID,
	}
}

/*
func (qq *ListInboxAssignmentSuggestions) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx *ctxx.Context) error {
	data, err := autil.FormData[ListInboxAssignmentSuggestionsData](rw, req, ctx)
	if err != nil {
		return err
	}
	return qq.infra.Renderer().Render(
		rw,
		qq.Widget(ctx),
	)

}
*/

// https://www.lawlessfrench.com/pronunciation/accents/
// https://en.wikipedia.org/wiki/Diacritic
// https://www.ut.edu/academics/college-of-arts-and-letters/department-of-languages-and-linguistics/typing-accented-characters
// last was used for list below
var regexpLowerAlphaNum = regexp.MustCompile("[^a-z0-9àèìòùáéíóúýâêîôûãñõäëïöüÿåæœçðø¿¡ß]+")

func (qq *ListInboxAssignmentSuggestions) Widget(ctx ctxx.Context, fileID int64) *wx.List {
	// TODO
	fileToAssign := ctx.TenantCtx().TTx.File.GetX(ctx, fileID)

	filename := filepath.Clean(fileToAssign.Name)
	// remove file extension
	extIndex := strings.LastIndex(filename, ".")
	if extIndex != -1 {
		filename = filename[:extIndex]
	}
	// to prevent collisions with search operators, OR AND must be uppercase in sqlite fts queries
	// to be interpreted as search operator; search itself is case insensitive;
	// IMPORTANT
	// must be before regexp replace so that regexp doesn't have to list upper case letters
	filename = strings.ToLower(filename)
	// replace all non alpha chars with space
	filename = regexpLowerAlphaNum.ReplaceAllString(filename, " ")

	// remove redundant whitespace (double space, etc.)
	filenameArr := strings.Fields(filename)

	// remove duplicate entires, only works if sorted, order seems not to affect fts rank
	slices.Sort(filenameArr)
	filenameArr = slices.Clip(slices.Compact(filenameArr)) // not sure if slices.Clip is necessary

	// TODO get rid of all number elements? probably just date (test first if it affects search result if we keep it)

	// TODO add * as suffix to each elem?
	searchQuery := strings.Join(filenameArr, " OR ")
	// searchQuery := "module* OR go* OR simpledms*"

	// TODO also query content via OCR

	// select *, rank from file_search where filename match 'module* OR go* OR simpledms*' and is_directory and rank < -5 order by rank limit 10;
	// is not case-sensitive
	/*
		problem with this approach is that I could not figure out quickly how to access `rank`
		and matching against database also seemed not to work; also seemed noticabily slow...
		maybe because file_searches.file_id is not indexed...
		files := ctx.TenantCtx().TTx.File.Query().
			Where(
				file.IsDirectory(true),
				func(qs *sql.Selector) {
					fileSearchTable := sql.Table(filesearch.Table)

					qs.LeftJoin(fileSearchTable).On(fileSearchTable.C(filesearch.FieldFileID), qs.C(file.FieldID)).
						Where(
							sql.And(
								sql.EQ(fileSearchTable.C(filesearch.FieldFileSearches), searchQuery),
								sql.LT(fileSearchTable.C(filesearch.FieldRank), -5),
							),
						).
						OrderBy(fileSearchTable.C(filesearch.FieldRank))
				},
			).
			Limit(10).
			AllX(ctx)
	*/

	/*
		fileIDs := []int64{}

		res := ctx.TenantCtx().TTx.FileSearch.Query().
			Select(filesearch.FieldFileID).
			Where(
				filesearch.IsDirectory(true),
				filesearch.FileSearches(searchQuery), // TODO is this safe against sqli?
				filesearch.RankLT(-5),
			).
			Limit(10).
			Order(filesearch.ByRank()).
			AllX(ctx)

	*/

	destDirs := ctx.TenantCtx().TTx.File.Query().
		WithChildren().
		Where(
			file.IsDirectory(true),
			func(qs *sql.Selector) {
				fileSearchTable := sql.Table(filesearch.Table)

				qs.Where(
					sql.In(qs.C(file.FieldID),
						sql.Select(fileSearchTable.C(filesearch.FieldRowid)).From(fileSearchTable).
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
		Limit(10).
		AllX(ctx)

	var destDirParentIDs []int64
	for _, child := range destDirs {
		destDirParentIDs = append(destDirParentIDs, child.ParentID)
	}
	slices.Sort(destDirParentIDs) // necessary for compact to work?
	destDirParentIDs = slices.Compact(destDirParentIDs)
	destDirParentFileInfosSlice := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FileIDIn(destDirParentIDs...)).AllX(ctx)
	destDirParentFileInfos := make(map[int64]*enttenant.FileInfo)
	for _, destDirParentFileInfo := range destDirParentFileInfosSlice {
		destDirParentFileInfos[destDirParentFileInfo.FileID] = destDirParentFileInfo
	}

	var items []*wx.ListItem

	for _, destDir := range destDirs {
		// TODO context menu: Open / Browse (for pasting)
		// TODO click open rename
		// TODO use a custom one and share code with file list?
		// TODO show breadcrumbs (first and last folder only?)
		items = append(items, qq.actions.AssignmentDirectoryListItem.Widget(
			ctx,
			destDir,
			fileToAssign,
			destDirParentFileInfos[destDir.ParentID].FullPath,
		))
	}

	return &wx.List{
		Children: items,
	}

	/*
		return &Column{
			Children: []IWidget{
				&ListView{
					Children: items,
				},
			},
		}

	*/
}
