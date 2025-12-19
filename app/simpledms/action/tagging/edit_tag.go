package tagging

import (
	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type EditTagData struct {
	// necessary for response (isChecked)
	TagID int64  `validate:"required" form_attr_type:"hidden"`
	Name  string `validate:"required" form_attrs:"autofocus"`
}

type EditTag struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[EditTagData]
}

func NewEditTag(
	infra *common.Infra,
	actions *Actions,
) *EditTag {
	config := actionx.NewConfig(actions.Route("edit-tag"), false)
	return &EditTag{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelper[EditTagData](
			infra,
			config,
			wx.T("Edit tag"),
			// "",
		),
	}
}

func (qq *EditTag) Data(tagID int64, name string) *EditTagData {
	return &EditTagData{
		TagID: tagID,
		Name:  name,
	}
}

func (qq *EditTag) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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
