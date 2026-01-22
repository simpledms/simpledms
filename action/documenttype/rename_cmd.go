package documenttype

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
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
		FormHelper: autil.NewFormHelper[RenameCmdData](infra, config, wx.T("RenameCmd document type")),
	}
}

func (qq *RenameCmd) Data(id int64, newName string) *RenameCmdData {
	return &RenameCmdData{
		ID:      id,
		NewName: newName,
	}
}

func (qq *RenameCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[RenameCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	err = ctx.TenantCtx().TTx.DocumentType.
		UpdateOneID(data.ID).
		Where(documenttype.SpaceID(ctx.SpaceCtx().Space.ID)).
		SetName(data.NewName).
		Exec(ctx)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.DocumentTypeUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("Document type renamed to «%s».", data.NewName))

	return nil
}
