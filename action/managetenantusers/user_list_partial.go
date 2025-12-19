package managetenantusers

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type UserListPartialState struct{}

type UserListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewUserListPartial(infra *common.Infra, actions *Actions) *UserListPartial {
	return &UserListPartial{
		infra:   infra,
		actions: actions,
		Config:  actionx.NewConfig("user-list-partial", true),
	}
}

func (qq *UserListPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	state := autil.StateX[UserListPartialState](rw, req)
	return qq.infra.Renderer().Render(rw, ctx, qq.Widget(ctx, state))
}

func (qq *UserListPartial) Widget(ctx ctxx.Context, state *UserListPartialState) *wx.List {
	var listItems []*wx.ListItem

	listItems = append(listItems, &wx.ListItem{
		Headline: wx.T("Add a new user"), // TODO Create or add? system or real world perspective?
		Leading:  wx.NewIcon("add"),
		Type:     wx.ListItemTypeHelper,
		HTMXAttrs: qq.actions.CreateUserCmd.ModalLinkAttrs(
			qq.actions.CreateUserCmd.Data(
				tenantrole.User,
				"",
				"",
				"",
				ctx.MainCtx().Account.Language, // TODO okay?
			), ""),
	})

	// TODO filtered by tenant?
	users := ctx.TenantCtx().TTx.User.Query().Order(user.ByLastName(), user.ByFirstName()).AllX(ctx)

	for _, userx := range users {
		userm := model.NewUser(userx)

		leading := wx.NewIcon("person")
		if userx.Role == tenantrole.Owner {
			// TODO add tooltip...
			leading = wx.NewIcon("manage_accounts")
		}

		listItems = append(listItems, &wx.ListItem{
			Leading:        leading,
			Headline:       wx.Tu(userm.Name()),
			SupportingText: wx.Tu(userm.NameSecondLine()),
		})
	}

	return &wx.List{
		Widget: wx.Widget[wx.List]{
			ID: qq.id(),
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxTrigger: event.HxTrigger(
				event.UserCreated,
				event.UserUpdated,
				event.UserDeleted,
			),
			HxPost:   qq.Endpoint(),
			HxTarget: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
		Children: listItems,
	}
}

func (qq *UserListPartial) id() string {
	return "userListPartial"
}
