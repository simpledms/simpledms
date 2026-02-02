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
	"entgo.io/ent/dialect/sql/sqljson"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileinfo"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
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
	actions *Actions
	*actionx.Config
	*autil.FormHelper[MoveFileData]
}

func NewMoveFile(infra *common.Infra, actions *Actions, config *actionx.Config) *MoveFile {
	return &MoveFile{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelper[MoveFileData](
			infra,
			config,
			wx.T("Move file"),
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

func (qq *MoveFile) FormHandler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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

	var currentDir *model.File
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
		return qq.infra.Renderer().Render(rw, ctx, &wx.View{
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
			actionx.ResponseWrapper(wrapper),
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
	currentDir *model.File,
	filex *model.File,
	data *MoveFileFormData,
	wrapper actionx.ResponseWrapper,
	hxTargetForm string,
	searchQuery string,
) renderable.Renderable {
	form := &wx.Form{
		Widget: wx.Widget[wx.Form]{
			ID: qq.formID(),
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: hxTargetForm,
			HxSwap:   "outerHTML",
		},
		Children: []wx.IWidget{
			&wx.Container{
				GapY: true,
				Child: []wx.IWidget{
					wx.NewFormFields(ctx, data),
					&wx.Container{
						Child: []wx.IWidget{
							wx.NewLabel(wx.LabelTypeMd, wx.T("Original filename")),
							wx.NewBody(wx.BodyTypeSm, wx.Tu(filex.Data.Name)),
						},
					},
				},
			},
		},
	}
	container := &wx.View{
		Children: []wx.IWidget{
			&wx.Search{
				Widget: wx.Widget[wx.Search]{
					ID: "moveSearch",
				},
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:    qq.FormEndpoint(),
					HxVals:    util.JSON(qq.Data(filex.Data.PublicID.String(), currentDir.Data.PublicID.String())),
					HxTarget:  "#" + qq.filesListID(),
					HxTrigger: fmt.Sprintf("input from:#moveSearch delay:100ms"),
					HxInclude: "#moveSearch, #" + qq.formID(),
				},
				Name:           "SearchQuery",
				Value:          searchQuery,
				SupportingText: wx.Tf("Search in «%s»", currentDir.Data.Name),
				Autofocus:      true,
			},
			qq.formFilesList(ctx, currentDir, filex, hxTargetForm, "", 0), // TODO
			form,
		},
	}

	// fileParentName := filex.QueryParent().OnlyX(ctx).Name

	return autil.WrapWidgetWithID(
		// fmt.Sprintf("Move «%s» from «%s» to «%s»", filex.Name, fileParentName, currentDir.Name),
		wx.Tf("Move file to «%s»", currentDir.Data.Name),
		wx.T("Save"),
		container,
		wrapper,
		wx.DialogLayoutStable,
		qq.popoverID(),
		qq.formID(),
	)
}

func (qq *MoveFile) pageSize() int {
	return 50
}

func (qq *MoveFile) formFilesList(
	ctx ctxx.Context,
	currentDir *model.File,
	filex *model.File,
	hxTargetForm string,
	searchQuery string,
	offset int,
) *wx.ScrollableContent {
	fileListItems := qq.formFilesListItems(ctx, currentDir, filex, hxTargetForm, searchQuery, offset)

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.filesListID(),
		},
		Children: &wx.List{
			Children: fileListItems,
		},
	}
}

func (qq *MoveFile) formFilesListItems(
	ctx ctxx.Context,
	currentDir *model.File,
	filex *model.File,
	hxTargetForm string,
	searchQuery string,
	offset int,
) []wx.IWidget {
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
				fileInfoView := sql.Table(fileinfo.Table)
				qs.Where(
					sql.In(
						qs.C(file.FieldID),
						sql.Select(fileInfoView.C(fileinfo.FieldFileID)).
							From(fileInfoView).
							Where(sql.And(
								sqljson.ValueContains(fileInfoView.C(fileinfo.FieldPath), currentDir.Data.ID),
								sql.NEQ(fileInfoView.C(fileinfo.FieldFileID), currentDir.Data.ID),
							)),
					),
				)
			})
	}

	childDirs := childDirsQuery.Order(file.ByName()).Offset(offset).Limit(qq.pageSize() + 1).AllX(ctx)
	hasMore := len(childDirs) > qq.pageSize()
	if hasMore {
		// conditional necessary to prevent out of bounce access
		childDirs = childDirs[:qq.pageSize()]
	}

	var fileListItems []wx.IWidget

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
			&wx.ListItem{
				Leading:  wx.NewIcon("arrow_upward"),
				Headline: wx.T("Directory up"),
				Type:     wx.ListItemTypeHelper,
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:    qq.FormEndpointWithParams(actionx.ResponseWrapperDialog, hxTargetForm),
					HxVals:    util.JSON(qq.Data(filex.Data.PublicID.String(), parentDir.Data.PublicID.String())),
					HxTarget:  "#" + qq.popoverID() + " .js-dialog-content",
					HxSelect:  ".js-dialog-content",
					HxSwap:    "outerHTML",
					HxInclude: "#" + qq.formID(),
				},
			},
		)
	}

	// load childDirFileInfos for breadcrumbs
	var childDirParentFileInfos map[int64]*enttenant.FileInfo
	if searchQuery != "" {
		var childDirIDs []int64
		for _, childDir := range childDirs {
			childDirIDs = append(childDirIDs, childDir.ParentID)
		}

		slices.Sort(childDirIDs) // necessary for compact to work?
		childDirIDs = slices.Compact(childDirIDs)

		childDirFileInfosSlice := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FileIDIn(childDirIDs...)).AllX(ctx)

		childDirParentFileInfos = make(map[int64]*enttenant.FileInfo)
		for _, childDirFileInfo := range childDirFileInfosSlice {
			childDirParentFileInfos[childDirFileInfo.FileID] = childDirFileInfo
		}
	}

	for _, childDir := range childDirs {
		if childDir.ID == filex.Data.ID {
			// cannot be moved to itself
			continue
		}

		supportingText := ""
		if searchQuery != "" {
			fileInfo := childDirParentFileInfos[childDir.ParentID]
			if fileInfo != nil {
				breadcrumbElems := []string{wx.T("Home").String(ctx)}
				if fileInfo.FullPath != "" {
					breadcrumbElems = append(breadcrumbElems, strings.Split(fileInfo.FullPath, string(os.PathSeparator))...)
				}
				supportingText = strings.Join(breadcrumbElems, " » ")
			}
		}

		fileListItems = append(fileListItems,
			&wx.ListItem{
				// BackgroundColor: "beige",
				Leading:        wx.NewIcon("folder").SmallPadding().HorizontalPadding(),
				Headline:       wx.T(childDir.Name),
				SupportingText: wx.Tu(supportingText),
				HTMXAttrs: wx.HTMXAttrs{
					HxPost:    qq.FormEndpointWithParams(actionx.ResponseWrapperDialog, hxTargetForm),
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
		fileListItems = append(fileListItems, &wx.ListItem{
			Widget: wx.Widget[wx.ListItem]{
				ID: "moveFileLoadMore",
			},
			Headline: wx.T("Loading more..."),
			HTMXAttrs: wx.HTMXAttrs{
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

func (qq *MoveFile) filesListID() string {
	return "filesList"
}
