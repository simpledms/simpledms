package tagging

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	taggingmodel "github.com/simpledms/simpledms/model/tenant/tagging"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateAndAssignTagCmdData struct {
	FileID           string `form_attr_type:"hidden"`
	CreateTagCmdData `structs:",flatten"`
}

type CreateAndAssignTagCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateAndAssignTagCmdData]
}

func NewCreateAndAssignTagCmd(
	infra *common.Infra,
	actions *Actions,
) *CreateAndAssignTagCmd {
	config := actionx.NewConfig(
		actions.Route("create-and-assign-tag-cmd"),
		false,
	).EnableCommittedResponse()
	return &CreateAndAssignTagCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelperX[CreateAndAssignTagCmdData](
			infra,
			config,
			widget.T("Create and assign tag"),
			widget.T("Create"),
		),
	}
}

func (qq *CreateAndAssignTagCmd) Data(fileID string, parentTagID int64) *CreateAndAssignTagCmdData {
	return &CreateAndAssignTagCmdData{
		FileID: fileID,
		CreateTagCmdData: CreateTagCmdData{
			GroupTagID: parentTagID,
		},
	}
}

func (qq *CreateAndAssignTagCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := qq.MapFormData(rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	var tagx *enttenant.Tag
	if data.Type == tagtype.Group {
		tagx, err = qq.actions.CreateTagCmd.execute(ctx, &data.CreateTagCmdData)
	} else {
		tagx, err = taggingmodel.NewTagService().CreateAndAssignToFile(
			ctx, filex.Data.ID, ctx.SpaceCtx().Space.ID, data.GroupTagID, data.Name, data.Type,
		)
	}
	if err != nil {
		log.Println(err)
		return err
	}
	rw.Header().Set("HX-Trigger", event.TagCreated.String())
	rw.AddRenderables(widget.NewSnackbarf("«%s» created and assigned.", tagx.Name))
	return nil
}
