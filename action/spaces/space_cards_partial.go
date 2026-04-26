package spaces

import (
	"strings"

	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/model/common/tenantrole"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	spacemodel "github.com/simpledms/simpledms/model/tenant/space"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type SpaceCardsPartialData struct{}

type SpaceCardsPartialState struct{}

// TODO SpacesCards or SpaceCardsPartial
type SpaceCardsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSpaceCardsPartial(infra *common.Infra, actions *Actions) *SpaceCardsPartial {
	config := actionx.NewConfig(
		actions.Route("space-cards-partial"),
		true,
	)
	return &SpaceCardsPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SpaceCardsPartial) Data() *SpaceCardsPartialData {
	return &SpaceCardsPartialData{}
}

func (qq *SpaceCardsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	/*
		data, err := autil.FormData[SpacesCardsData](rw, req, ctx)
		if err != nil {
			return err
		}
		state := autil.StateX[SpaceCardsPartialState](rw, req)

	*/

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx),
	)
}

func (qq *SpaceCardsPartial) Widget(
	ctx ctxx.Context,
	// data *SpacesCardsData,
	// state *SpaceCardsPartialState,
) renderable.Renderable {
	spaces := ctx.AppCtx().TTx.Space.Query().
		/*WithSpaceAssignment(func(query *ent.SpaceAssignmentQuery) {
			query.Where(spaceassignment.IsRootDir(true))
		}).*/
		Order(space.ByName()).
		AllX(ctx)

	var cards []*widget.Card

	for _, spacex := range spaces {
		cards = append(cards, qq.card(ctx, spacex, ctx.TenantCtx().Tenant.PublicID.String()))
	}

	htmxAttrs := widget.HTMXAttrs{
		HxTrigger: strings.Join([]string{
			event.SpaceCreated.Handler(),
			event.SpaceUpdated.Handler(),
			event.SpaceDeleted.Handler(),
		}, ", "),
		HxPost:   qq.Endpoint(),
		HxTarget: "#" + qq.ID(),
		HxSwap:   "outerHTML",
	}

	if len(spaces) == 0 {
		var actions []widget.IWidget

		if ctx.AppCtx().User.Role == tenantrole.Owner {
			actions = append(actions,
				qq.actions.CreateSpaceDialog.ModalLink(
					qq.actions.CreateSpaceDialog.Data("", ""),
					[]widget.IWidget{
						&widget.Button{
							Icon:  widget.NewIcon("add"),
							Label: widget.T("Create space"),
						},
					},
					"",
				),
			)
		}

		return &widget.Container{
			Widget: widget.Widget[widget.Container]{
				ID: qq.ID(),
			},
			Child: &widget.EmptyState{
				Icon:     widget.NewIcon("hub"),
				Headline: widget.T("No spaces available yet."),
				Actions:  actions,
			},
			HTMXAttrs: htmxAttrs,
		}
	}

	return &widget.Grid{
		Widget: widget.Widget[widget.Grid]{
			ID: qq.ID(),
		},
		Children:  cards,
		HTMXAttrs: htmxAttrs,
	}
}

func (qq *SpaceCardsPartial) card(
	ctx ctxx.Context,
	spacex *enttenant.Space,
	tenantID string,
) *widget.Card {
	var actions []*widget.Button

	heading := widget.H(widget.HeadingTypeTitleLg, widget.Tu(spacex.Name))

	isActive := ctx.IsSpaceCtx() && ctx.SpaceCtx().Space.ID == spacex.ID
	if isActive {
		// TODO change layout of card when active
		// TODO seems never the case because in TenantCtx...
		actions = append(actions, &widget.Button{
			Label:     widget.T("Files"), // TODO or Selected or Active?
			StyleType: widget.ButtonStyleTypeOutlined,
			HTMXAttrs: widget.HTMXAttrs{
				HxGet: route.BrowseRoot(tenantID, spacex.PublicID.String()),
			},
		})

		// TODO use icon to indicate instead?
		heading = widget.H(widget.HeadingTypeTitleLg, widget.Tf("%s (%s)", widget.Tu(spacex.Name).String(ctx), widget.T("active").String(ctx)))
	} else {
		actions = append(actions, &widget.Button{
			Label:     widget.T("Select"), // TODO Switch or activate? or Select?
			StyleType: widget.ButtonStyleTypeOutlined,
			HTMXAttrs: widget.HTMXAttrs{
				HxGet: route.BrowseRoot(tenantID, spacex.PublicID.String()),
			},
		})
	}

	var subhead *widget.Text
	if spacex.IsFolderMode {
		// subhead = wx.T("Folder mode") // TODO rename to Hybrid mode?
	} else {
		// subhead = wx.T("Default mode")
	}

	return &widget.Card{
		Style:    widget.CardStyleFilled,
		Headline: heading,
		Subhead:  subhead,
		// TODO Subhead:        wx.T("Show stats about user access (new manage button) and files (on view button)"),
		SupportingText: widget.Tu(spacex.Description),
		Actions:        actions,
		ContextMenu:    NewSpaceContextMenuWidget(qq.actions).Widget(ctx, spacemodel.NewSpace(spacex)),
	}
}

func (qq *SpaceCardsPartial) ID() string {
	return "spaceCards"
}
