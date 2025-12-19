package inbox

import (
	"os"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/enttenant/fileinfo"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AssignmentDirectoryListItemData struct {
	DestDirID int64
	FileID    int64
}

type AssignmentDirectoryListItem struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAssignmentDirectoryListItem(infra *common.Infra, actions *Actions) *AssignmentDirectoryListItem {
	return &AssignmentDirectoryListItem{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("assignment-directory-list-item"),
			true,
		),
	}
}

func (qq *AssignmentDirectoryListItem) Data(destDirID, fileID int64) *AssignmentDirectoryListItemData {
	return &AssignmentDirectoryListItemData{
		DestDirID: destDirID,
		FileID:    fileID,
	}
}

func (qq *AssignmentDirectoryListItem) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignmentDirectoryListItemData](rw, req, ctx)
	if err != nil {
		return err
	}

	destDir := ctx.TenantCtx().TTx.File.GetX(ctx, data.DestDirID)
	filex := ctx.TenantCtx().TTx.File.GetX(ctx, data.FileID)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, destDir, filex, ""),
	)
}

func (qq *AssignmentDirectoryListItem) Widget(
	ctx ctxx.Context,
	destDir *enttenant.File,
	fileToAssign *enttenant.File,
	destParentFullPath string,
) *wx.ListItem {
	if destParentFullPath == "" {
		// if ID is used instead of ParentID, lastElem must be removed in next step (filepath.Dir)
		destParentFullPath = ctx.TenantCtx().TTx.FileInfo.Query().Where(fileinfo.FileID(destDir.ParentID)).OnlyX(ctx).FullPath
	}
	breadcrumbElems := []string{wx.T("Home").String(ctx)}
	if destParentFullPath != "" {
		breadcrumbElems = append(breadcrumbElems, strings.Split(destParentFullPath, string(os.PathSeparator))...)
	}
	supportingText := strings.Join(breadcrumbElems, " » ")

	return &wx.ListItem{
		Headline:       wx.Tf(destDir.Name),
		SupportingText: wx.Tu(supportingText),
		HTMXAttrs: qq.actions.AssignFile.ModalLinkAttrs(
			qq.actions.AssignFile.Data(destDir.PublicID.String(), fileToAssign.PublicID.String(), fileToAssign.Name),
			"#innerContent",
		).SetHxHeaders(autil.QueryHeader(
			qq.actions.Page.Endpoint(),
			qq.actions.Page.Data(),
		)),
	}
}
