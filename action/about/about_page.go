package about

import (
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/renderable"
	partial2 "github.com/simpledms/simpledms/ui/uix/partial"
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
		Navigation: partial2.NewNavigationRail(ctx, qq.infra, "about", fabs),
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
					wx.Pu(`This program is free software: you can redistribute it and/or modify
					it under the terms of the GNU Affero General Public License version 3 as
					published by the Free Software Foundation.`),
					wx.Pu(`This program is distributed in the hope that it will be useful,
					but WITHOUT ANY WARRANTY; without even the implied warranty of
					MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
					GNU Affero General Public License for more details.`),
					wx.Pu(`You should have received a copy of the GNU Affero General Public License
					along with this program. If not, see <https://www.gnu.org/licenses/>.`),

					wx.H(wx.HeadingTypeTitleLg, wx.Tu(`Additional terms under GNU Affero General Public License version 3 section 7`)),

					wx.Pu(`In accordance with Section 7(b) of the GNU Affero General Public License version 3, the following additional terms are added to preserve attribution notices under Section 5(d):`),

					wx.Pu(`1. A visible menu item labeled «About SimpleDMS», «Legal», «License Information», «About», or an equivalent term must be present in the main menu of the program linking to the attribution page described below.`),
					wx.Pu(`2. A visible link labeled «Powered by SimpleDMS», «Legal», «License Information», «About», or an equivalent term must be displayed on the login page of the program. The link must lead to the same attribution page described below or to https://simpledms.eu/open-source.`),

					wx.Pu(`The attribution page must not be removed or modified so as to delete or obscure the required attribution notices. However, the attribution page may be extended, including by adding:`),

					wx.Pu(`- additional copyright holders or contributors,`),
					wx.Pu(`- notices describing modifications made to the program, or`),
					wx.Pu(`- other legally required or informative notices, provided that the required attribution notices remain clearly identifiable and reasonably prominent.`),

					wx.Pu(`The attribution notices required:`),

					wx.Pu(`- may be presented in a manner consistent with the overall visual design of the program, but`),
					wx.Pu(`- must not be deliberately hidden, obscured, or rendered non-functional.`),

					wx.Pu(`This requirement does not apply where the Program is used exclusively without an interactive user interface.`),

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
		LeadingAltMobile: partial2.NewMainMenu(ctx, qq.infra),
		Title: &wx.AppBarTitle{
			Text: wx.T("About SimpleDMS"),
		},
		Actions: []wx.IWidget{},
	}
}
