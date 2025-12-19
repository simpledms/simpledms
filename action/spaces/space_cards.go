package spaces

import (
	"strings"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/event"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/renderable"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type SpaceCardsData struct{}

type SpaceCardsState struct{}

// TODO SpacesCards or SpaceCards
type SpaceCards struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSpaceCards(infra *common.Infra, actions *Actions) *SpaceCards {
	config := actionx.NewConfig(
		actions.Route("spaces-cards"),
		true,
	)
	return &SpaceCards{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SpaceCards) Data() *SpaceCardsData {
	return &SpaceCardsData{}
}

func (qq *SpaceCards) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	/*
		data, err := autil.FormData[SpacesCardsData](rw, req, ctx)
		if err != nil {
			return err
		}
		state := autil.StateX[SpaceCardsState](rw, req)

	*/

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx),
	)
}

func (qq *SpaceCards) Widget(
	ctx ctxx.Context,
	// data *SpacesCardsData,
	// state *SpaceCardsState,
) renderable.Renderable {
	spaces := ctx.TenantCtx().TTx.Space.Query().
		/*WithSpaceAssignment(func(query *ent.SpaceAssignmentQuery) {
			query.Where(spaceassignment.IsRootDir(true))
		}).*/
		Order(space.ByName()).
		AllX(ctx)

	var cards []*wx.Card

	for _, spacex := range spaces {
		cards = append(cards, qq.card(ctx, spacex, ctx.TenantCtx().Tenant.PublicID.String()))
	}

	htmxAttrs := wx.HTMXAttrs{
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
		var actions []wx.IWidget

		if ctx.TenantCtx().User.Role == tenantrole.Owner {
			actions = append(actions,
				qq.actions.CreateSpace.ModalLink(
					qq.actions.CreateSpace.Data("", ""),
					[]wx.IWidget{
						&wx.Button{
							Icon:  wx.NewIcon("add"),
							Label: wx.T("Create space"),
						},
					},
					"",
				),
			)
		}

		return &wx.Container{
			Widget: wx.Widget[wx.Container]{
				ID: qq.ID(),
			},
			Child: &wx.EmptyState{
				Icon:     wx.NewIcon("hub"),
				Headline: wx.T("No spaces available yet."),
				Actions:  actions,
			},
			HTMXAttrs: htmxAttrs,
		}
	}

	return &wx.Grid{
		Widget: wx.Widget[wx.Grid]{
			ID: qq.ID(),
		},
		Children:  cards,
		HTMXAttrs: htmxAttrs,
	}
}

func (qq *SpaceCards) card(
	ctx ctxx.Context,
	spacex *enttenant.Space,
	tenantID string,
) *wx.Card {
	var actions []*wx.Button

	heading := wx.H(wx.HeadingTypeTitleLg, wx.Tu(spacex.Name))

	isActive := ctx.IsSpaceCtx() && ctx.SpaceCtx().Space.ID == spacex.ID
	if isActive {
		// TODO change layout of card when active
		// TODO seems never the case because in TenantCtx...
		actions = append(actions, &wx.Button{
			Label:     wx.T("Browse"), // TODO or Selected or Active?
			StyleType: wx.ButtonStyleTypeOutlined,
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route.BrowseRoot(tenantID, spacex.PublicID.String()),
			},
		})

		// TODO use icon to indicate instead?
		heading = wx.H(wx.HeadingTypeTitleLg, wx.Tf("%s (%s)", spacex.Name, wx.T("active").String(ctx)))
	} else {
		actions = append(actions, &wx.Button{
			Label:     wx.T("Select"), // TODO Switch or activate? or Select?
			StyleType: wx.ButtonStyleTypeOutlined,
			HTMXAttrs: wx.HTMXAttrs{
				HxGet: route.BrowseRoot(tenantID, spacex.PublicID.String()),
			},
		})
	}

	var subhead *wx.Text
	if spacex.IsFolderMode {
		// subhead = wx.T("Folder mode") // TODO rename to Hybrid mode?
	} else {
		// subhead = wx.T("Default mode")
	}

	return &wx.Card{
		Style:    wx.CardStyleFilled,
		Headline: heading,
		Subhead:  subhead,
		// TODO Subhead:        wx.T("Show stats about user access (new manage button) and files (on view button)"),
		SupportingText: wx.Tu(spacex.Description),
		Actions:        actions,
		ContextMenu:    NewSpaceContextMenu(qq.actions).Widget(ctx, model.NewSpace(spacex)),
	}
}

func (qq *SpaceCards) ID() string {
	return "spaceCards"
}
