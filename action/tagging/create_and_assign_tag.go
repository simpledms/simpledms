package tagging

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/event"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateAndAssignTagData struct {
	FileID        string `form_attr_type:"hidden"`
	CreateTagData `structs:",flatten"`
}

type CreateAndAssignTag struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateAndAssignTagData]
}

func NewCreateAndAssignTag(
	infra *common.Infra,
	actions *Actions,
) *CreateAndAssignTag {
	config := actionx.NewConfig(
		actions.Route("create-and-assign-tag"),
		false,
	)
	return &CreateAndAssignTag{
		infra:   infra,
		actions: actions,
		Config:  config,
		FormHelper: autil.NewFormHelper[CreateAndAssignTagData](
			infra,
			config,
			wx.T("Create and assign tag"),
			// "#tagAssignmentList",
		),
	}
}

func (qq *CreateAndAssignTag) Data(fileID string, parentTagID int64) *CreateAndAssignTagData {
	return &CreateAndAssignTagData{
		FileID: fileID,
		CreateTagData: CreateTagData{
			GroupTagID: parentTagID,
		},
	}
}

func (qq *CreateAndAssignTag) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := qq.FormHelper.MapFormData(rw, req, ctx)
	if err != nil {
		return err
	}

	tagx, err := qq.actions.CreateTag.execute(ctx, &data.CreateTagData)
	if err != nil {
		log.Println(err)
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	// TODO move to model
	if tagx.Type != tagtype.Group {
		ctx.TenantCtx().TTx.TagAssignment.Create().
			SetTag(tagx).
			SetFileID(filex.Data.ID).
			SetSpaceID(ctx.SpaceCtx().Space.ID).
			// SetIsInherited(false).
			SaveX(ctx)

		// must be set before writing to rw
		rw.Header().Set("HX-Trigger", event.TagUpdated.String())
	}

	if data.GroupTagID > 0 {
		parentTag := ctx.TenantCtx().TTx.Tag.
			Query().
			Where(tag.ID(data.GroupTagID)).
			WithChildren().
			OnlyX(ctx)
		listItem := qq.actions.AssignedTags.EditListItem.ListItem(
			ctx,
			data.FileID,
			parentTag,
		)

		// TODO is it possible to get rid of this with new mechanism
		//		to pass hx-target into form?
		// TODO deactivated on 24.02.2025 to fix on demand creation onmetadata tab,
		//  	may break other places
		// rw.Header().Set("HX-Retarget", "#"+listItem.ID)

		rw.AddRenderables(wx.NewSnackbarf("«%s» created and assigned.", tagx.Name))

		return qq.infra.Renderer().Render(
			rw,
			ctx,
			listItem,
		)
	}

	rw.AddRenderables(wx.NewSnackbarf("«%s» created and assigned.", tagx.Name))

	return qq.infra.Renderer().Render(rw, ctx,
		qq.actions.AssignedTags.Edit.ListView(
			ctx,
			&EditAssignedTagsData{
				FileID:      data.FileID,
				ParentTagID: data.GroupTagID,
			},
		),
	)
}
