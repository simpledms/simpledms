package browse

import (
	"log"

	acommon "github.com/simpledms/simpledms/action/common"
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type SelectDirMakeDirCmdData struct {
	*acommon.MoveFileData `structs:",flatten"` // TODO use SelectDirPartialData once implemented
	NewDirName            string               `form_attrs:"autofocus"`
}

// Name only makes sense once SelectDirPartial is factored out from MoveFileCmd and can
// be used independently
type SelectDirMakeDirCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[SelectDirMakeDirCmdData]
}

func NewSelectDirMakeDirCmd(
	infra *common.Infra,
	actions *Actions,
) *SelectDirMakeDirCmd {
	config := actionx.NewConfig(
		actions.Route("select-dir/make-dir-cmd"), // TODO suffix should be handled by actions (embedding)
		false,
	)
	return &SelectDirMakeDirCmd{
		infra,
		actions,
		config,
		autil.NewFormHelper[SelectDirMakeDirCmdData](
			infra,
			config,
			wx.T("Create directory"),
			// "#fileList",
		),
	}
}

func (qq *SelectDirMakeDirCmd) Data(moveFileData *acommon.MoveFileData, dirName string) *SelectDirMakeDirCmdData {
	return &SelectDirMakeDirCmdData{
		MoveFileData: moveFileData,
		NewDirName:   dirName,
	}
}

func (qq *SelectDirMakeDirCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[SelectDirMakeDirCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex, err := qq.infra.FileSystem().MakeDir(ctx, data.CurrentDirID, data.NewDirName)
	if err != nil {
		log.Println(err)
		return err
	}

	// switch dir
	data.MoveFileData.CurrentDirID = filex.Data.PublicID.String()

	// hxTarget := req.URL.Query().Get("hx-target")

	// TODO how to handle type of view? (table, list, cards)
	// TODO return list partial / may depend on context...
	return qq.infra.Renderer().Render(rw, ctx,
		// qq.actions.MoveFileCmd.Form(ctx, data.MoveFileCmdData, actionx.ResponseWrapperDialog, "#fileList"),
		wx.NewSnackbarf("«%s» created.", filex.Data.Name),
	)
}
