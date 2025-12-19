package about

import (
	acommon "github.com/simpledms/simpledms/app/simpledms/action/common"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/renderable"
	"github.com/simpledms/simpledms/app/simpledms/ui/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/httpx"
)

type AboutPage struct {
	acommon.Page
	infra   *common.Infra
	actions *Actions
}

func NewAboutPage(infra *common.Infra, actions *Actions) *AboutPage {
	return &AboutPage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *AboutPage) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	return qq.Render(rw, req, ctx, qq.infra, "About", qq.Widget(ctx))
}

func (qq *AboutPage) Widget(ctx ctxx.Context) renderable.Renderable {
	var fabs []*wx.FloatingActionButton

	mainLayout := &wx.MainLayout{
		Navigation: partial.NewNavigationRail(ctx, "about", fabs),
		Content: &wx.DefaultLayout{
			AppBar: qq.appBar(ctx),
			Content: &wx.Column{
				HTMXAttrs:        wx.HTMXAttrs{},
				GapYSize:         wx.Gap4,
				NoOverflowHidden: true,
				// TODO impl a markdown renderer instead
				Children: []wx.IWidget{
					wx.P("Copyright (c) 2023–present Marco Beierer"),
					&wx.Column{
						GapYSize:         wx.Gap2,
						NoOverflowHidden: true,
						Children: []wx.IWidget{
							wx.P("SimpleDMS is an Open Source project by Marco Beierer. The official websites are:"),
							&wx.Link{
								Href:  "https://simpledms.eu",
								Child: wx.T("simpledms.eu"),
							},
							&wx.Link{
								Href:  "https://simpledms.ch",
								Child: wx.T("simpledms.ch"),
							},
							&wx.Link{
								Href:  "https://www.marcobeierer.com",
								Child: wx.T("marcobeierer.com"),
							},
							&wx.Link{
								Href:  "https://www.marcobeierer.ch",
								Child: wx.T("marcobeierer.ch"),
							},
						},
					},
					&wx.Column{
						GapYSize:         wx.Gap2,
						NoOverflowHidden: true,
						Children: []wx.IWidget{
							wx.P("The project is hosted on GitHub. You find the repository with the source code at:"),
							&wx.Link{
								Href:  "https://github.com/simpledms/simpledms",
								Child: wx.T("https://github.com/simpledms/simpledms"),
							},
						},
					},

					wx.T(`If you got your copy from someone else or using a hosted version by 
					another provider, the vendor must provide you a list with all modifications and
					a copy of the modified source code.`),

					wx.H(wx.HeadingTypeHeadlineSm, wx.T("License")),
					wx.P(`This program is free software: you can redistribute it and/or modify
					it under the terms of the GNU Affero General Public License version 3 as
					published by the Free Software Foundation.`),
					wx.P(`This program is distributed in the hope that it will be useful,
					but WITHOUT ANY WARRANTY; without even the implied warranty of
					MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
					GNU Affero General Public License for more details.`),
					wx.P(`You should have received a copy of the GNU Affero General Public License
					along with this program. If not, see <https://www.gnu.org/licenses/>.`),
					// TODO or headline?
					wx.P(`Additional terms under GNU Affero General Public License version 3 section 7:`).SetBold(),
					wx.P(`All copies of the program, in both source code and executable form, must
					preserve the "Powered by SimpleDMS" attribution notice on each user interface
					screen. Clicking the notice must direct the user to https://simpledms.eu or https://simpledms.ch.`),
					wx.P(`This notice must be visible to all users without additional interaction, and
					must not be removed, obscured, or altered.`),
					wx.P(`All copies of the program, in both source code and executable form, must
					preserve the "About SimpleDMS" menu item in the main menu. The content of
					the linked about page must not be modified.`),
					wx.P(`This obligation also applies to all derivative works and any copies of
					derivative works.`),

					// TODO explain rights (get source code) and obligations of provider (list modficiations)
					// (link to page explaining more)

					// TODO list with open source project it depends on (or directly in repo?)
				},
			},
		},
	}

	return mainLayout
}

func (qq *AboutPage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "info",
		},
		LeadingAltMobile: partial.NewMainMenu(ctx),
		Title: &wx.AppBarTitle{
			Text: wx.T("About SimpleDMS"),
		},
		Actions: []wx.IWidget{},
	}
}
