package browse

import (
	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FileVersionFromInboxListPartial struct {
	infra   *common.Infra
	actions *Actions
	helper  *acommon.MergeFileVersionHelper
	*actionx.Config
}

func NewFileVersionFromInboxListPartial(infra *common.Infra, actions *Actions) *FileVersionFromInboxListPartial {
	config := actionx.NewConfig(actions.Route("file-version-from-inbox-list-partial"), true)
	return &FileVersionFromInboxListPartial{
		infra:   infra,
		actions: actions,
		helper:  acommon.NewMergeFileVersionHelper(),
		Config:  config,
	}
}

func (qq *FileVersionFromInboxListPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FileVersionFromInboxDialogData](rw, req, ctx)
	if err != nil {
		return err
	}

	if data.TargetFileID == "" {
		return nil
	}

	targetFile := qq.infra.FileRepo.GetX(ctx, data.TargetFileID)
	files := qq.helper.SuggestInboxSources(ctx, targetFile.Data, data.SearchQuery, 0)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.actions.FileVersionFromInboxDialog.listWrapper(ctx, data, files),
	)
}
