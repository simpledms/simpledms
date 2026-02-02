package browse

import (
	"fmt"
	"os"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/fileinfo"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileListItemPartialData struct {
	CurrentDirID string
	FileID       string
}

type FileListItemPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileListItemPartial(infra *common.Infra, actions *Actions) *FileListItemPartial {
	return &FileListItemPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("file-list-item-partial"),
			true,
		),
	}
}

func (qq *FileListItemPartial) Data(currentDirID, fileID string) *FileListItemPartialData {
	return &FileListItemPartialData{
		CurrentDirID: currentDirID,
		FileID:       fileID,
	}
}

func (qq *FileListItemPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileListItemPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := ctx.TenantCtx().TTx.File.Query().WithChildren().Where(file.PublicID(entx.NewCIText(data.FileID))).OnlyX(ctx)

	qq.infra.Renderer().RenderX(
		rw,
		ctx,
		qq.Widget(ctx, data.CurrentDirID, filex, "", false, false), // TODO is selected? and hide context menu
	)
	return nil
}

// type HrefFn func(dirID, fileID int64, tab string) string

// TODO type data?
//
// Deprecated: use DirectoryListItem or FileListItemPartial directly
func (qq *FileListItemPartial) Widget(
	ctx ctxx.Context,
	currentDirID string,
	filex *enttenant.File,
	parentFullPath string, // only necessary with breadcrumbs
	isSelected bool,
	// hideContextMenu bool,
	showBreadcrumbs bool,
) *wx.ListItem {
	if filex.IsDirectory {
		return qq.DirectoryListItem(ctx, currentDirID, filex, parentFullPath, showBreadcrumbs)
	}
	return qq.fileListItem(ctx, currentDirID, filex, parentFullPath, isSelected, showBreadcrumbs)
}

// TODO make private again
func (qq *FileListItemPartial) DirectoryListItem(
	ctx ctxx.Context,
	currentDirID string,
	fileWithChildren *enttenant.File,
	parentFullPath string, // only necessary with breadcrumbs
	showBreadcrumbs bool,
) *wx.ListItem {
	supportingText := ""
	if showBreadcrumbs {
		if parentFullPath == "" {
			// if ID is used instead of ParentID, lastElem must be removed in next step (filepath.Dir)
			parentFullPath = ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FileID(fileWithChildren.ParentID)).OnlyX(ctx).FullPath
		}

		currentDirPath := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.PublicFileID(currentDirID)).OnlyX(ctx).FullPath
		if parentFullPath == currentDirPath {
			supportingText = qq.supportingTextDirectory(fileWithChildren, supportingText)
		} else {
			parentFullPath = strings.TrimPrefix(parentFullPath, currentDirPath+string(os.PathSeparator))

			var breadcrumbElems []string
			if parentFullPath != "" {
				breadcrumbElems = append(breadcrumbElems, strings.Split(parentFullPath, string(os.PathSeparator))...)
			}
			supportingText = strings.Join(breadcrumbElems, " » ")
		}
	} else {
		supportingText = qq.supportingTextDirectory(fileWithChildren, supportingText)
	}

	icon := wx.NewIcon("folder")
	headline := wx.T(fileWithChildren.Name)

	// check if root dir
	if fileWithChildren.ParentID == 0 {
		icon = wx.NewIcon("home")
	}

	return &wx.ListItem{
		RadioGroupName: "fileListRadioGroup",
		// BackgroundColor: "beige",
		Leading:        icon.SmallPadding(),
		Headline:       headline,
		SupportingText: wx.Tu(supportingText),
		ContextMenu:    NewFileContextMenuPartial(qq.actions).Widget(ctx, fileWithChildren),
		HTMXAttrs: wx.HTMXAttrs{
			HxGet:     route.Browse(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, fileWithChildren.PublicID.String()),
			HxHeaders: autil.ResetStateHeader(), // necessary to close side sheet
			HxSwap: fmt.Sprintf(
				// duplicate in ListDirPartial
				// bottom instead of top prevents small jump on nav
				// TODO long query is not ideal because it is error prone, but spaces are not allowed in htmx...
				"innerHTML show:#%s>div>.js-list>.js-list-item:first-child:bottom",
				qq.actions.ListDirPartial.FileListID(),
			),
		},
		// TODO use file_paths view instead?
		// Href: hrefFn(fileWithChildren.ID), // commented on 2024.10.27
	}
}

func (qq *FileListItemPartial) supportingTextDirectory(fileWithChildren *enttenant.File, supportingText string) string {
	// TODO is this faster than queries above? probably
	var dirCount, fileCount int64
	for _, childOfChild := range fileWithChildren.Edges.Children {
		if childOfChild.IsDirectory {
			dirCount++
		} else {
			fileCount++
		}
	}

	var supportingTextArr []string
	if dirCount > 1 {
		supportingTextArr = append(supportingTextArr, fmt.Sprintf("%d directories", dirCount))
	} else if dirCount == 1 {
		supportingTextArr = append(supportingTextArr, fmt.Sprintf("%d directory", dirCount))
	}
	if fileCount > 1 {
		supportingTextArr = append(supportingTextArr, fmt.Sprintf("%d files", fileCount))
	} else if fileCount == 1 {
		supportingTextArr = append(supportingTextArr, fmt.Sprintf("%d file", fileCount))
	}
	if len(supportingTextArr) == 0 {
		supportingTextArr = append(supportingTextArr, "Empty directory")
	}

	supportingText = fmt.Sprintf(strings.Join(supportingTextArr, ", ")) // TODO add size?
	return supportingText
}

func (qq *FileListItemPartial) fileListItem(
	ctx ctxx.Context,
	currentDirID string,
	fileWithChildren *enttenant.File,
	parentFullPath string, // only necessary with breadcrumbs
	isSelected bool,
	// hideContextMenu bool,
	showBreadcrumbs bool,
) *wx.ListItem {
	htmxAttrs := wx.HTMXAttrs{
		HxTarget: "#details",
		HxSwap:   "outerHTML",
		// dirID and not fileWithChildren.ID so that it works nicely with `recursive` filter
		HxGet:     route.BrowseFile(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, currentDirID, fileWithChildren.PublicID.String()),
		HxHeaders: autil.PreserveStateHeader(),
	}

	filexx := qq.infra.FileRepo.GetXX(fileWithChildren)

	supportingText := ""
	hasBreadcrumbs := false
	if showBreadcrumbs {
		if parentFullPath == "" {
			// if ID is used instead of ParentID, lastElem must be removed in next step (filepath.Dir)
			parentFullPath = ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FileID(fileWithChildren.ParentID)).OnlyX(ctx).FullPath
		}

		currentDirPath := ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.PublicFileID(currentDirID)).OnlyX(ctx).FullPath
		if parentFullPath == currentDirPath {
			supportingText = qq.supportingTextFile(ctx, filexx, supportingText)
		} else {
			parentFullPath = strings.TrimPrefix(parentFullPath, currentDirPath+string(os.PathSeparator))

			var breadcrumbElems []string
			if parentFullPath != "" {
				breadcrumbElems = append(breadcrumbElems, strings.Split(parentFullPath, string(os.PathSeparator))...)
			}
			supportingText = strings.Join(breadcrumbElems, " » ")
			hasBreadcrumbs = true
		}
	} else {
		supportingText = qq.supportingTextFile(ctx, filexx, supportingText)
	}

	withDocumentType := hasBreadcrumbs
	headline := wx.Tu(filexx.FilenameInApp(ctx, withDocumentType))

	return &wx.ListItem{
		RadioGroupName: "fileListRadioGroup",
		// BackgroundColor: "aliceblue",
		Leading:        wx.NewIcon("description").SmallPadding(),
		ContextMenu:    NewFileContextMenuPartial(qq.actions).Widget(ctx, fileWithChildren),
		Headline:       headline,
		SupportingText: wx.Tu(supportingText),
		HTMXAttrs:      htmxAttrs,
		IsSelected:     isSelected,
	}
}

func (qq *FileListItemPartial) supportingTextFile(ctx ctxx.Context, filexx *model.File, supportingText string) string {
	if filexx.Data.DocumentTypeID != 0 {
		documentTypex, err := filexx.Data.Edges.DocumentTypeOrErr()
		if err != nil {
			documentTypex = filexx.Data.QueryDocumentType().OnlyX(ctx)
		}
		supportingText = documentTypex.Name
	}
	return supportingText
}
