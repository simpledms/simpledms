package inbox

import (
	"os"
	"strings"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
)

type AssignmentDirectoryListItemPartialData struct {
	DestDirID int64
	FileID    int64
}

type AssignmentDirectoryListItemPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAssignmentDirectoryListItemPartial(infra *common.Infra, actions *Actions) *AssignmentDirectoryListItemPartial {
	return &AssignmentDirectoryListItemPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("assignment-directory-list-item-partial"),
			true,
		),
	}
}

func (qq *AssignmentDirectoryListItemPartial) Data(destDirID, fileID int64) *AssignmentDirectoryListItemPartialData {
	return &AssignmentDirectoryListItemPartialData{
		DestDirID: destDirID,
		FileID:    fileID,
	}
}

func (qq *AssignmentDirectoryListItemPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AssignmentDirectoryListItemPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	destDir := ctx.AppCtx().TTx.File.GetX(ctx, data.DestDirID)
	filex := ctx.AppCtx().TTx.File.GetX(ctx, data.FileID)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, destDir, filex, ""),
	)
}

func (qq *AssignmentDirectoryListItemPartial) Widget(
	ctx ctxx.Context,
	destDir *enttenant.File,
	fileToAssign *enttenant.File,
	destParentFullPath string,
) *widget.ListItem {
	if destParentFullPath == "" {
		// if ID is used instead of ParentID, lastElem must be removed in next step (filepath.Dir)
		destParentFullPath = qq.infra.FileSystem().FileTree().FullPathByFileIDX(ctx, destDir.ParentID)
	}
	breadcrumbElems := []string{widget.T("Home").String(ctx)}
	if destParentFullPath != "" {
		breadcrumbElems = append(breadcrumbElems, strings.Split(destParentFullPath, string(os.PathSeparator))...)
	}
	supportingText := strings.Join(breadcrumbElems, " » ")

	return &widget.ListItem{
		Headline:       widget.Tf(destDir.Name),
		SupportingText: widget.Tu(supportingText),
		HTMXAttrs: qq.actions.AssignFileCmd.ModalLinkAttrs(
			qq.actions.AssignFileCmd.Data(destDir.PublicID.String(), fileToAssign.PublicID.String(), fileToAssign.Name),
			"#innerContent",
		).SetHxHeaders(autil.QueryHeader(
			qq.actions.InboxPage.Endpoint(),
			qq.actions.InboxPage.Data(),
		)),
	}
}
