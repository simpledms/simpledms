package documenttype

import (
	"fmt"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/common/attributetype"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AddPropertyAttributeData struct {
	DocumentTypeID int64 `validate:"required" form_attr_type:"hidden"`
}

type AddPropertyAttributeFormData struct {
	AddPropertyAttributeData `structs:",flatten"`
	// TODO name?
	PropertyID   int64 `validate:"required" structs:"-"` // used in a string below
	IsNameGiving bool
}

type AddPropertyAttribute struct {
	infra   *common.Infra
	Actions *Actions
	*actionx.Config
	*autil.FormHelper[AddPropertyAttributeData]
}

func NewAddPropertyAttribute(infra *common.Infra, actions *Actions) *AddPropertyAttribute {
	config := actionx.NewConfig(
		actions.Route("add-property"),
		false,
	)
	formHelper := autil.NewFormHelper[AddPropertyAttributeData](
		infra,
		config,
		wx.T("Add field"),
	)
	return &AddPropertyAttribute{
		infra:      infra,
		Actions:    actions,
		Config:     config,
		FormHelper: formHelper,
	}
}

func (qq *AddPropertyAttribute) Data(documentTypeID int64) *AddPropertyAttributeData {
	return &AddPropertyAttributeData{
		DocumentTypeID: documentTypeID,
	}
}

func (qq *AddPropertyAttribute) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AddPropertyAttributeFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	attributex := ctx.TenantCtx().TTx.Attribute.Create().
		SetType(attributetype.Field).
		SetDocumentTypeID(data.DocumentTypeID).
		SetPropertyID(data.PropertyID).
		SetIsNameGiving(data.IsNameGiving).
		SetSpaceID(ctx.SpaceCtx().Space.ID).
		SaveX(ctx)

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.DocumentTypeAttributeCreated.String()) // TODO okay?

	propertyx := attributex.QueryProperty().OnlyX(ctx)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		wx.NewSnackbarf("Attribute «%s» added.", propertyx.Name),
	)
}

func (qq *AddPropertyAttribute) FormHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormDataX[AddPropertyAttributeFormData](rw, req, ctx, true)
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

func (qq *AddPropertyAttribute) Form(
	ctx ctxx.Context,
	data *AddPropertyAttributeFormData,
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
					qq.propertyList(ctx, hxTarget),
				},
			},
		},
	}

	return autil.WrapWidgetWithID(
		wx.T("Add field attribute"),
		wx.T("Save"),
		form,
		wrapper,
		wx.DialogLayoutStable,
		qq.popoverID(),
		qq.formID(),
	)

}

func (qq *AddPropertyAttribute) popoverID() string {
	return "addPropertyAttributePopover"
}

func (qq *AddPropertyAttribute) formID() string {
	return "addPropertyAttributeForm"
}

func (qq *AddPropertyAttribute) propertyList(ctx ctxx.Context, hxTarget string) wx.IWidget {
	propertyListItems := qq.propertyListItems(ctx, hxTarget)

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.propertyListID(),
		},
		Children: &wx.List{
			Children: propertyListItems,
		},
	}
}

func (qq *AddPropertyAttribute) propertyListID() string {
	return "propertyList"
}

func (qq *AddPropertyAttribute) propertyListItems(ctx ctxx.Context, target string) interface{} {
	// TODO implement pagination

	var items []*wx.ListItem

	properties := ctx.SpaceCtx().Space.QueryProperties().AllX(ctx)

	if len(properties) == 0 {
		items = append(items, &wx.ListItem{
			Headline:       wx.T("No fields available yet."),
			SupportingText: wx.T("Please create a field first."), // TODO link
		})
		return items
	}

	for _, propertyx := range properties {
		items = append(items, &wx.ListItem{
			RadioGroupName: "PropertyID",
			RadioValue:     fmt.Sprintf("%d", propertyx.ID),
			Headline:       wx.Tu(propertyx.Name),
			SupportingText: wx.T(propertyx.Type.String()),
			Leading:        wx.NewIcon("tune"),
		})
	}

	return items
}
