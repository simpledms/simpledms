package documenttype

import (
	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type RenameCmdData struct {
	ID      int64  `form_attr_type:"hidden"`
	NewName string `validate:"required"`
}

type RenameCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[RenameCmdData]
}

func NewRenameCmd(infra *common.Infra, actions *Actions) *RenameCmd {
	config := actionx.NewConfig(
		actions.Route("rename-cmd"),
		false,
	)
	return &RenameCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[RenameCmdData](infra, config, widget.T("RenameCmd document type")),
	}
}

func (qq *RenameCmd) Data(id int64, newName string) *RenameCmdData {
	return &RenameCmdData{
		ID:      id,
		NewName: newName,
	}
}

func (qq *RenameCmd) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[RenameCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	documentTypex, err := documenttypemodel.QueryByID(ctx, ctx.SpaceCtx().Space.ID, data.ID)
	if err != nil {
		return err
	}

	err = documentTypex.Rename(ctx, data.NewName)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.DocumentTypeUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("Document type renamed to «%s».", data.NewName))

	return nil
}
