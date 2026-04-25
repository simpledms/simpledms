package documenttype

import (
	"fmt"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/renderable"
	"github.com/simpledms/simpledms/core/ui/widget"
	actionx2 "github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type AddPropertyAttributeCmdData struct {
	DocumentTypeID int64 `validate:"required" form_attr_type:"hidden"`
}

type AddPropertyAttributeCmdFormData struct {
	AddPropertyAttributeCmdData `structs:",flatten"`
	// TODO name?
	PropertyID   int64 `validate:"required" structs:"-"` // used in a string below
	IsNameGiving bool
}

type AddPropertyAttributeCmd struct {
	infra   *common.Infra
	Actions *Actions
	*actionx2.Config
	*autil.FormHelper[AddPropertyAttributeCmdData]
}

func NewAddPropertyAttributeCmd(infra *common.Infra, actions *Actions) *AddPropertyAttributeCmd {
	config := actionx2.NewConfig(
		actions.Route("add-property-cmd"),
		false,
	)
	formHelper := autil.NewFormHelper[AddPropertyAttributeCmdData](
		infra,
		config,
		widget.T("Add field"),
	)
	return &AddPropertyAttributeCmd{
		infra:      infra,
		Actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *AddPropertyAttributeCmd) Data(documentTypeID int64) *AddPropertyAttributeCmdData {
	return &AddPropertyAttributeCmdData{
		DocumentTypeID: documentTypeID,
	}
}

func (qq *AddPropertyAttributeCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AddPropertyAttributeCmdFormData](rw, req, ctx)
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

	attributex, err := documentTypex.CreatePropertyAttribute(ctx, data.PropertyID, data.IsNameGiving)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.DocumentTypeAttributeCreated.String()) // TODO okay?

	propertyx := attributex.QueryProperty().OnlyX(ctx)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		widget.NewSnackbarf("Attribute «%s» added.", propertyx.Name),
	)
}

func (qq *AddPropertyAttributeCmd) FormHandler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormDataX[AddPropertyAttributeCmdFormData](rw, req, ctx, true)
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

func (qq *AddPropertyAttributeCmd) Form(
	ctx ctxx.Context,
	data *AddPropertyAttributeCmdFormData,
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
					qq.propertyList(ctx, hxTarget),
				},
			},
		},
	}

	return autil.WrapWidgetWithID(
		widget.T("Add field attribute"),
		widget.T("Save"),
		form,
		wrapper,
		widget.DialogLayoutStable,
		qq.popoverID(),
		qq.formID(),
	)

}

func (qq *AddPropertyAttributeCmd) popoverID() string {
	return "addPropertyAttributePopover"
}

func (qq *AddPropertyAttributeCmd) formID() string {
	return "addPropertyAttributeForm"
}

func (qq *AddPropertyAttributeCmd) propertyList(ctx ctxx.Context, hxTarget string) widget.IWidget {
	propertyListItems := qq.propertyListItems(ctx, hxTarget)

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.propertyListID(),
		},
		Children: &widget.List{
			Children: propertyListItems,
		},
	}
}

func (qq *AddPropertyAttributeCmd) propertyListID() string {
	return "propertyList"
}

func (qq *AddPropertyAttributeCmd) propertyListItems(ctx ctxx.Context, target string) interface{} {
	// TODO implement pagination

	var items []*widget.ListItem

	properties := ctx.SpaceCtx().Space.QueryProperties().AllX(ctx)

	if len(properties) == 0 {
		items = append(items, &widget.ListItem{
			Headline:       widget.T("No fields available yet."),
			SupportingText: widget.T("Please create a field first."), // TODO link
		})
		return items
	}

	for _, propertyx := range properties {
		items = append(items, &widget.ListItem{
			RadioGroupName: "PropertyID",
			RadioValue:     fmt.Sprintf("%d", propertyx.ID),
			Headline:       widget.Tu(propertyx.Name),
			SupportingText: widget.T(propertyx.Type.String()),
			Leading:        widget.NewIcon("tune"),
		})
	}

	return items
}
