package common

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/action/util"

	common2 "github.com/simpledms/simpledms/core/action/common"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/renderable"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	actionx2 "github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
)

// TODO rename to MoveFileCommand or MoveFileCmd?
type MoveFileData struct {
	FileID       string `form_attr_type:"hidden"`
	CurrentDirID string `form_attr_type:"hidden"`
}

type MoveFileFormData struct {
	MoveFileData `structs:",flatten"`
	NewDirName   string `form_leading_icon:"create_new_folder"`        // TODO jumps around on selection with autofocus... `form_attrs:"autofocus"`
	Filename     string `validate:"required" form_leading_icon:"edit"` // TODO or `save` or `edit`?
}

// necessary to render page
type MoveFileState struct {
	// MoveFileData
	SearchQuery string
}

type MoveFile struct {
	infra   *common.Infra
	actions *common2.Actions
	*actionx2.Config
	*autil.FormHelper[MoveFileData]
}

func NewMoveFile(infra *common.Infra, actions *common2.Actions, config *actionx2.Config) *MoveFile {
	return &MoveFile{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelper[MoveFileData](
			infra,
			config,
			widget.T("Move file"),
		),
	}
}

func (qq *MoveFile) Data(fileID, currentDirID string) *MoveFileData {
	return &MoveFileData{
		FileID:       fileID,
		CurrentDirID: currentDirID,
		// Filename:     filename,
	}
}

func (qq *MoveFile) FormHandler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	if !ctx.SpaceCtx().Space.IsFolderMode {
		return e.NewHTTPErrorf(http.StatusMethodNotAllowed, "Only allowed in folder mode.")
	}

	data, err := autil.FormDataX[MoveFileFormData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	state, err := autil.FormData[MoveFileState](rw, req, ctx)
	if err != nil {
		return err
	}

	hxTarget := req.URL.Query().Get("hx-target")
	wrapper := req.URL.Query().Get("wrapper")

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	var currentDir *filemodel.File
	if data.CurrentDirID == "" {
		// navigate from current directory
		currentDir, err = filex.Parent(ctx)
		if err != nil {
			log.Println(err)
			return err
		}
	} else {
		currentDir = qq.infra.FileRepo.GetX(ctx, data.CurrentDirID)
	}

	if data.Filename == "" {
		data.Filename = filex.Data.Name
	}

	// state := autil.StateX[MoveDirSate](rw, req)

	switch req.Header.Get("Hx-Target") {
	case qq.filesListID(): // used in search
		return qq.infra.Renderer().Render(rw, ctx, qq.formFilesList(
			ctx,
			currentDir,
			filex,
			hxTarget,
			state.SearchQuery,
			0,
		))
	case "moveFileLoadMore":
		offset := 0
		offsetStr := req.URL.Query().Get("offset")
		if offsetStr != "" {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil {
				log.Println(err)
				return err
			}
		}
		return qq.infra.Renderer().Render(rw, ctx, &widget.View{
			Children: qq.formFilesListItems(
				ctx,
				currentDir,
				filex,
				hxTarget,
				state.SearchQuery,
				offset,
			)},
		)
	}

	return qq.infra.Renderer().Render(rw, ctx,
		qq.Form(
			ctx,
			currentDir,
			filex,
			data,
			actionx2.ResponseWrapper(wrapper),
			hxTarget,
			state.SearchQuery,
		),
	)
}

func (qq *MoveFile) popoverID() string {
	// random doesn't work on back (dir up) if unique;
	return "moveFilePopover"
}
func (qq *MoveFile) formID() string {
	return "moveFileForm"
}

// returns form, filex and currentDir
// TODO use FormHelper instead?
func (qq *MoveFile) Form(
	ctx ctxx.Context,
	currentDir *filemodel.File,
	filex *filemodel.File,
	data *MoveFileFormData,
	wrapper actionx2.ResponseWrapper,
	hxTargetForm string,
	searchQuery string,
) renderable.Renderable {
	form := &widget.Form{
		Widget: widget.Widget[widget.Form]{
			ID: qq.formID(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: hxTargetForm,
			HxSwap:   "outerHTML",
		},
		Children: []widget.IWidget{
			&widget.Container{
				GapY: true,
				Child: []widget.IWidget{
					widget.NewFormFields(ctx, data),
					&widget.Container{
						Child: []widget.IWidget{
							widget.NewLabel(widget.LabelTypeMd, widget.T("Original filename")),
							widget.NewBody(widget.BodyTypeSm, widget.Tu(filex.Data.Name)),
						},
					},
				},
			},
		},
	}
	container := &widget.View{
		Children: []widget.IWidget{
			&widget.Search{
				Widget: widget.Widget[widget.Search]{
					ID: "moveSearch",
				},
				HTMXAttrs: widget.HTMXAttrs{
					HxPost:    qq.FormEndpoint(),
					HxVals:    util.JSON(qq.Data(filex.Data.PublicID.String(), currentDir.Data.PublicID.String())),
					HxTarget:  "#" + qq.filesListID(),
					HxTrigger: fmt.Sprintf("input from:#moveSearch delay:100ms"),
					HxInclude: "#moveSearch, #" + qq.formID(),
				},
				Name:           "SearchQuery",
				Value:          searchQuery,
				SupportingText: widget.Tf("Search in «%s»", currentDir.Data.Name),
				Autofocus:      true,
			},
			qq.formFilesList(ctx, currentDir, filex, hxTargetForm, "", 0), // TODO
			form,
		},
	}

	// fileParentName := filex.QueryParent().OnlyX(ctx).Name

	return autil.WrapWidgetWithID(
		// fmt.Sprintf("Move «%s» from «%s» to «%s»", filex.Name, fileParentName, currentDir.Name),
		widget.Tf("Move file to «%s»", currentDir.Data.Name),
		widget.T("Save"),
		container,
		wrapper,
		widget.DialogLayoutStable,
		qq.popoverID(),
		qq.formID(),
	)
}

func (qq *MoveFile) pageSize() int {
	return 50
}

func (qq *MoveFile) formFilesList(
	ctx ctxx.Context,
	currentDir *filemodel.File,
	filex *filemodel.File,
	hxTargetForm string,
	searchQuery string,
	offset int,
) *widget.ScrollableContent {
	fileListItems := qq.formFilesListItems(ctx, currentDir, filex, hxTargetForm, searchQuery, offset)

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.filesListID(),
		},
		Children: &widget.List{
			Children: fileListItems,
		},
	}
}

func (qq *MoveFile) formFilesListItems(
	ctx ctxx.Context,
	currentDir *filemodel.File,
	filex *filemodel.File,
	hxTargetForm string,
	searchQuery string,
	offset int,
) []widget.IWidget {
	// TODO process searchQuery and add breadcrumbs if search is used

	var childDirsQuery *enttenant.FileQuery

	if searchQuery == "" {
		childDirsQuery = currentDir.Data.
			QueryChildren().
			Where(file.IsDirectory(true))
	} else {
		childDirsQuery = ctx.TenantCtx().TTx.File.Query().
			Where(
				file.NameContains(searchQuery),
				file.IsDirectory(true),
			).
			Where(func(qs *sql.Selector) {
				// add dirs recursively from current dir if in search mode
				qs.Where(qq.descendantScopePredicate(qs.C(file.FieldID), currentDir.Data.ID, ctx.SpaceCtx().Space.ID))
			})
	}

	childDirs := childDirsQuery.Order(file.ByName()).Offset(offset).Limit(qq.pageSize() + 1).AllX(ctx)
	hasMore := len(childDirs) > qq.pageSize()
	if hasMore {
		// conditional necessary to prevent out of bounce access
		childDirs = childDirs[:qq.pageSize()]
	}

	var fileListItems []widget.IWidget

	// TODO selectDir command with custom action...

	// not safe to do in condition above because data.CurrentDirID could be id of root
	currentDirIsRoot := currentDir.Data.ParentID == 0

	if !currentDirIsRoot {
		parentDir, err := currentDir.Parent(ctx)
		if err != nil {
			log.Println(err)
			panic(err) // FIXME panic or okay?
		}
		fileListItems = append(fileListItems,
			&widget.ListItem{
				Leading:  widget.NewIcon("arrow_upward"),
				Headline: widget.T("Directory up"),
				Type:     widget.ListItemTypeHelper,
				HTMXAttrs: widget.HTMXAttrs{
					HxPost:    qq.FormEndpointWithParams(actionx2.ResponseWrapperDialog, hxTargetForm),
					HxVals:    util.JSON(qq.Data(filex.Data.PublicID.String(), parentDir.Data.PublicID.String())),
					HxTarget:  "#" + qq.popoverID() + " .js-dialog-content",
					HxSelect:  ".js-dialog-content",
					HxSwap:    "outerHTML",
					HxInclude: "#" + qq.formID(),
				},
			},
		)
	}

	// load parent directory full paths for breadcrumbs
	var childDirParentFullPaths map[int64]string
	if searchQuery != "" {
		var childDirIDs []int64
		for _, childDir := range childDirs {
			childDirIDs = append(childDirIDs, childDir.ParentID)
		}

		slices.Sort(childDirIDs) // necessary for compact to work?
		childDirIDs = slices.Compact(childDirIDs)
		childDirParentFullPaths = qq.infra.FileSystem().FileTree().FullPathsByFileIDX(ctx, childDirIDs)
	}

	for _, childDir := range childDirs {
		if childDir.ID == filex.Data.ID {
			// cannot be moved to itself
			continue
		}

		supportingText := ""
		if searchQuery != "" {
			fullPath, found := childDirParentFullPaths[childDir.ParentID]
			if found {
				breadcrumbElems := []string{widget.T("Home").String(ctx)}
				if fullPath != "" {
					breadcrumbElems = append(breadcrumbElems, strings.Split(fullPath, string(os.PathSeparator))...)
				}
				supportingText = strings.Join(breadcrumbElems, " » ")
			}
		}

		fileListItems = append(fileListItems,
			&widget.ListItem{
				// BackgroundColor: "beige",
				Leading:        widget.NewIcon("folder").SmallPadding().HorizontalPadding(),
				Headline:       widget.T(childDir.Name),
				SupportingText: widget.Tu(supportingText),
				HTMXAttrs: widget.HTMXAttrs{
					HxPost:    qq.FormEndpointWithParams(actionx2.ResponseWrapperDialog, hxTargetForm),
					HxVals:    util.JSON(qq.Data(filex.Data.PublicID.String(), childDir.PublicID.String())),
					HxTarget:  "#" + qq.popoverID() + " .js-dialog-content",
					HxSelect:  ".js-dialog-content",
					HxSwap:    "outerHTML",
					HxInclude: "#" + qq.formID(),
				},
			},
		)
	}

	if hasMore {
		fileListItems = append(fileListItems, &widget.ListItem{
			Widget: widget.Widget[widget.ListItem]{
				ID: "moveFileLoadMore",
			},
			Headline: widget.T("Loading more..."),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:    qq.FormEndpoint() + "?offset=" + strconv.Itoa(offset+qq.pageSize()), // FIXME
				HxVals:    util.JSON(qq.Data(filex.Data.PublicID.String(), currentDir.Data.PublicID.String())),
				HxTrigger: "intersect once",
				HxTarget:  "#moveFileLoadMore",
				HxSwap:    "outerHTML",
			},
		})
	}

	return fileListItems
}

func (qq *MoveFile) descendantScopePredicate(fileColumn string, rootID, spaceID int64) *sql.Predicate {
	return sql.In(fileColumn, qq.infra.FileSystem().FileTree().DescendantIDsSubQuery(rootID, spaceID))
}

func (qq *MoveFile) filesListID() string {
	return "filesList"
}
