package documenttype

// package action

import (
	"fmt"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	"github.com/simpledms/simpledms/model/common/attributetype"
	"github.com/simpledms/simpledms/model/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type CreateAttributeData struct {
	DocumentTypeID int64 `validate:"required" form_attr_type:"hidden"`
}

type CreateAttributeFormData struct {
	CreateAttributeData `structs:",flatten"`
	Name                string `validate:"required" form_attrs:"autofocus"`
	TagID               int64  `validate:"required" structs:"-"`
	IsNameGiving        bool
}

// TODO rename to Add or CreateTagAttribute
type CreateAttribute struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[CreateAttributeData]
}

func NewCreateAttribute(infra *common.Infra, actions *Actions) *CreateAttribute {
	config := actionx.NewConfig(
		actions.Route("create-attribute"),
		false,
	)
	formHelper := autil.NewFormHelper[CreateAttributeData](
		infra,
		config,
		wx.T("Add attribute"),
	)
	return &CreateAttribute{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *CreateAttribute) Data(documentTypeID int64) *CreateAttributeData {
	return &CreateAttributeData{
		DocumentTypeID: documentTypeID,
	}
}

func (qq *CreateAttribute) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[CreateAttributeFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	// state := autil.StateX[PageState](rw, req)

	// documentTypex := ctx.TenantCtx().TTx.DocumentType.GetX(ctx, data.DocumentTypeID)

	attributex := ctx.TenantCtx().TTx.Attribute.Create().
		SetName(data.Name).
		SetTagID(data.TagID).
		SetType(attributetype.Tag).
		SetIsNameGiving(data.IsNameGiving).
		SetDocumentTypeID(data.DocumentTypeID).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		SaveX(ctx)

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.DocumentTypeAttributeCreated.String())

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		wx.NewSnackbarf("Attribute «%s» created.", attributex.Name),
	)
}

func (qq *CreateAttribute) FormHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormDataX[CreateAttributeFormData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	// TODO state?

	hxTarget := req.URL.Query().Get("hx-target")
	wrapper := req.URL.Query().Get("wrapper")

	return qq.infra.Renderer().Render(rw, ctx,
		qq.Form(
			ctx,
			data,
			actionx.ResponseWrapper(wrapper),
			hxTarget,
		),
	)
}

func (qq *CreateAttribute) Form(
	ctx ctxx.Context,
	data *CreateAttributeFormData,
	wrapper actionx.ResponseWrapper,
	hxTarget string,
) renderable.Renderable {
	form := &wx.Form{
		Widget: wx.Widget[wx.Form]{
			ID: qq.formID(),
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   "outerHTML",
		},
		Children: []wx.IWidget{
			&wx.Container{
				GapY: true,
				Child: []wx.IWidget{
					wx.NewFormFields(ctx, data),
					qq.tagsList(ctx, hxTarget),
				},
			},
		},
	}

	return autil.WrapWidgetWithID(
		wx.T("Add attribute"),
		wx.T("Save"),
		form,
		wrapper,
		wx.DialogLayoutStable,
		qq.popoverID(),
		qq.formID(),
	)

}

func (qq *CreateAttribute) popoverID() string {
	return "createAttributePopover"
}

func (qq *CreateAttribute) formID() string {
	return "createAttributeForm"
}

func (qq *CreateAttribute) tagsList(ctx ctxx.Context, hxTarget string) wx.IWidget {
	tagsListItems := qq.tagsListItems(ctx, hxTarget)

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.tagsListID(),
		},
		Children: &wx.List{
			Children: tagsListItems,
		},
	}
}

func (qq *CreateAttribute) tagsListID() string {
	return "tagsList"
}

func (qq *CreateAttribute) tagsListItems(ctx ctxx.Context, target string) interface{} {
	// TODO implement pagination

	var items []*wx.ListItem

	// TODO add value tags
	tagGroups := ctx.SpaceCtx().Space.QueryTags().Where(tag.TypeEQ(tagtype.Group)).AllX(ctx)

	if len(tagGroups) == 0 {
		items = append(items, &wx.ListItem{
			Headline:       wx.T("No tag groups available yet."),
			SupportingText: wx.T("Please create a tag group first."),
		})
		return items
	}

	for _, tagGroup := range tagGroups {
		icon := "label"
		if tagGroup.Icon != "" {
			icon = tagGroup.Icon
		}
		items = append(items, &wx.ListItem{
			RadioGroupName: "TagID",
			RadioValue:     fmt.Sprintf("%d", tagGroup.ID),
			Headline:       wx.Tu(tagGroup.Name),
			Leading:        wx.NewIcon(icon),
		})
	}

	return items
}
