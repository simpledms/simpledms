package documenttype

// package action

import (
	"fmt"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/widget"
	actionx2 "github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/tag"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	"github.com/simpledms/simpledms/model/tenant/tagging/tagtype"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type CreateAttributeCmdData struct {
	DocumentTypeID int64 `validate:"required" form_attr_type:"hidden"`
}

type CreateAttributeCmdFormData struct {
	CreateAttributeCmdData `structs:",flatten"`
	Name                   string `validate:"required" form_attrs:"autofocus"`
	TagID                  int64  `validate:"required" structs:"-"`
	IsNameGiving           bool
}

// TODO rename to Add or CreateTagAttribute
type CreateAttributeCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
	*autil.FormHelper[CreateAttributeCmdData]
}

func NewCreateAttributeCmd(infra *common.Infra, actions *Actions) *CreateAttributeCmd {
	config := actionx2.NewConfig(
		actions.Route("create-attribute-cmd"),
		false,
	)
	formHelper := autil.NewFormHelper[CreateAttributeCmdData](
		infra,
		config,
		widget.T("Add attribute"),
	)
	return &CreateAttributeCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *CreateAttributeCmd) Data(documentTypeID int64) *CreateAttributeCmdData {
	return &CreateAttributeCmdData{
		DocumentTypeID: documentTypeID,
	}
}

func (qq *CreateAttributeCmd) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[CreateAttributeCmdFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	documentTypex, err := documenttypemodel.QueryByID(
		ctx,
		ctx.SpaceCtx().Space.ID,
		data.DocumentTypeID,
	)
	if err != nil {
		return err
	}

	attributex, err := documentTypex.CreateTagAttribute(ctx, data.Name, data.TagID, data.IsNameGiving)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.DocumentTypeAttributeCreated.String())

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		widget.NewSnackbarf("Attribute «%s» created.", attributex.Name),
	)
}

func (qq *CreateAttributeCmd) FormHandler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormDataX[CreateAttributeCmdFormData](rw, req, ctx, true)
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
			actionx2.ResponseWrapper(wrapper),
			hxTarget,
		),
	)
}

func (qq *CreateAttributeCmd) Form(
	ctx ctxx.Context,
	data *CreateAttributeCmdFormData,
	wrapper actionx2.ResponseWrapper,
	hxTarget string,
) renderable.Renderable {
	form := &widget.Form{
		Widget: widget.Widget[widget.Form]{
			ID: qq.formID(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost:   qq.Endpoint(),
			HxTarget: hxTarget,
			HxSwap:   "outerHTML",
		},
		Children: []widget.IWidget{
			&widget.Container{
				GapY: true,
				Child: []widget.IWidget{
					widget.NewFormFields(ctx, data),
					qq.tagsList(ctx, hxTarget),
				},
			},
		},
	}

	return autil.WrapWidgetWithID(
		widget.T("Add attribute"),
		widget.T("Save"),
		form,
		wrapper,
		widget.DialogLayoutStable,
		qq.popoverID(),
		qq.formID(),
	)

}

func (qq *CreateAttributeCmd) popoverID() string {
	return "createAttributePopover"
}

func (qq *CreateAttributeCmd) formID() string {
	return "createAttributeForm"
}

func (qq *CreateAttributeCmd) tagsList(ctx ctxx.Context, hxTarget string) widget.IWidget {
	tagsListItems := qq.tagsListItems(ctx, hxTarget)

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.tagsListID(),
		},
		Children: &widget.List{
			Children: tagsListItems,
		},
	}
}

func (qq *CreateAttributeCmd) tagsListID() string {
	return "tagsList"
}

func (qq *CreateAttributeCmd) tagsListItems(ctx ctxx.Context, target string) interface{} {
	// TODO implement pagination

	var items []*widget.ListItem

	// TODO add value tags
	tagGroups := ctx.SpaceCtx().Space.QueryTags().Where(tag.TypeEQ(tagtype.Group)).AllX(ctx)

	if len(tagGroups) == 0 {
		items = append(items, &widget.ListItem{
			Headline:       widget.T("No tag groups available yet."),
			SupportingText: widget.T("Please create a tag group first."),
		})
		return items
	}

	for _, tagGroup := range tagGroups {
		icon := "label"
		if tagGroup.Icon != "" {
			icon = tagGroup.Icon
		}
		items = append(items, &widget.ListItem{
			RadioGroupName: "TagID",
			RadioValue:     fmt.Sprintf("%d", tagGroup.ID),
			Headline:       widget.Tu(tagGroup.Name),
			Leading:        widget.NewIcon(icon),
		})
	}

	return items
}
