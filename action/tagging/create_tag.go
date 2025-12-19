package tagging

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/event"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateTagData struct {
	GroupTagID int64           `form_attr_type:"hidden"`
	Name       string          `validate:"required" form_attrs:"autofocus"`
	Type       tagtype.TagType `validate:"required"` // `schema:"default:Base"`
	// IsGroup     bool
	// IsComposed  bool
	// Color    string
	// Icon     string
	// Weight   int
	// AttributeName   string
}

// TODO or AddNewTag NewTag
type CreateTag struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateTagData]
}

func NewCreateTag(
	infra *common.Infra,
	actions *Actions,
) *CreateTag {
	config := actionx.NewConfig(
		actions.Route("create-tag"),
		false,
	)
	return &CreateTag{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelper[CreateTagData](
			infra,
			config,
			wx.T("Create tag"),
			// "TODO", // TODO
		),
	}
}

func (qq *CreateTag) Data(groupTagID int64) *CreateTagData {
	return &CreateTagData{
		GroupTagID: groupTagID,
	}
}

func (qq *CreateTag) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[CreateTagData](rw, req, ctx)
	if err != nil {
		return err
	}

	tagx, err := qq.execute(ctx, data)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.TagCreated.String())
	rw.AddRenderables(wx.NewSnackbarf("Tag «%s» created.", tagx.Name))

	return nil
}

func (qq *CreateTag) execute(ctx ctxx.Context, data *CreateTagData) (*enttenant.Tag, error) {
	// TODO prevent in form
	if data.GroupTagID != 0 && data.Type == tagtype.Group {
		return nil, e.NewHTTPErrorf(http.StatusBadRequest, "Cannot add a tag group as child.")
	}

	tagCreate := ctx.TenantCtx().TTx.Tag.Create().
		SetName(data.Name).
		SetType(data.Type).
		SetSpaceID(ctx.SpaceCtx().Space.ID)
	// SetIsGroup(data.IsGroup).
	// SetIsComposed(data.IsComposed)

	if data.GroupTagID != 0 {
		tagCreate.SetGroupID(data.GroupTagID)
	}

	tag := tagCreate.SaveX(ctx)
	return tag, nil
}
