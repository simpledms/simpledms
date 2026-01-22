package tagging

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type EditTagCmdData struct {
	// necessary for response (isChecked)
	TagID int64  `validate:"required" form_attr_type:"hidden"`
	Name  string `validate:"required" form_attrs:"autofocus"`
}

type EditTagCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditTagCmdData]
}

func NewEditTagCmd(
	infra *common.Infra,
	actions *Actions,
) *EditTagCmd {
	config := actionx.NewConfig(actions.Route("edit-tag"), false)
	return &EditTagCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelper[EditTagCmdData](
			infra,
			config,
			wx.T("Edit tag"),
			// "",
		),
	}
}

func (qq *EditTagCmd) Data(tagID int64, name string) *EditTagCmdData {
	return &EditTagCmdData{
		TagID: tagID,
		Name:  name,
	}
}

func (qq *EditTagCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := qq.MapFormData(rw, req, ctx)
	if err != nil {
		return err
	}

	tagx := ctx.TenantCtx().TTx.
		Tag.
		UpdateOneID(data.TagID).
		SetName(data.Name).
		SaveX(ctx)

	rw.Header().Set("HX-Trigger", event.TagUpdated.String())
	rw.Header().Set("HX-Reswap", "none")
	rw.AddRenderables(wx.NewSnackbarf("«%s» updated.", tagx.Name))

	return nil
}
