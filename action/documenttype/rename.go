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

type RenameData struct {
	ID      int64  `form_attr_type:"hidden"`
	NewName string `validate:"required"`
}

type Rename struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[RenameData]
}

func NewRename(infra *common.Infra, actions *Actions) *Rename {
	config := actionx.NewConfig(
		actions.Route("rename"),
		false,
	)
	return &Rename{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[RenameData](infra, config, wx.T("Rename document type")),
	}
}

func (qq *Rename) Data(id int64, newName string) *RenameData {
	return &RenameData{
		ID:      id,
		NewName: newName,
	}
}

func (qq *Rename) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[RenameData](rw, req, ctx)
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
