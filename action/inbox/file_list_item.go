package inbox

// package action

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileListItemData struct {
	FileID int64
}

type FileListItem struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFileListItem(infra *common.Infra, actions *Actions) *FileListItem {
	return &FileListItem{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("file-list-item"),
			true,
		),
	}
}

func (qq *FileListItem) Data(fileID int64) *FileListItemData {
	return &FileListItemData{
		FileID: fileID,
	}
}

func (qq *FileListItem) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileListItemData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := ctx.TenantCtx().TTx.File.Query().WithChildren().Where(file.ID(data.FileID)).OnlyX(ctx)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		// TODO is hrefFn correct?
		qq.Widget(ctx, route.Inbox, filex, false), // TODO isSelected via state or data?
	)
}

type HrefFn func(tenantID, spaceID, fileID string) string

func (qq *FileListItem) Widget(
	ctx ctxx.Context,
	hrefFn HrefFn,
	// listState *ListFilesState,
	fileWithChildren *enttenant.File,
	isSelected bool,
) *wx.ListItem {
	/*trailing := &IconButton{
		Icon:     "more_vert",
		Children: NewFileContextMenu(qq.actions).Widget(fileWithChildren),
	}*/

	htmxAttrs := wx.HTMXAttrs{
		HxTarget: "#details",
		HxSwap:   "outerHTML",
		HxGet:    hrefFn(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, fileWithChildren.PublicID.String()),
	}

	return &wx.ListItem{
		RadioGroupName: "fileListRadioGroup",
		Leading:        wx.NewIcon("description").SmallPadding(),
		Headline:       wx.T(fileWithChildren.Name),
		/*SupportingText: wx.Tf(
			"%s, %s",
			qq.infra.FileRepo.GetXX(fileWithChildren).CurrentVersion(ctx).SizeString(),
			fileWithChildren.ModifiedAt.Format("02. January 06"),
		),*/
		HTMXAttrs: htmxAttrs,
		// Trailing:   trailing,
		IsSelected:  isSelected,
		ContextMenu: NewFileContextMenu(qq.actions).Widget(ctx, fileWithChildren),
	}
}
