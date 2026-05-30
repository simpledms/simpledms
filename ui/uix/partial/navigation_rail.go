package partial

import (
	"log"
	"sort"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	maintenant "github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	tenantmodel "github.com/simpledms/simpledms/model/main/tenant"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
)

// fab must be injected because it differs on each page...
func NewNavigationRail(
	ctx ctxx.Context,
	infra *common.Infra,
	active string,
	fabs []*wx.FloatingActionButton,
) *wx.NavigationRail {
	rail := &wx.NavigationRail{
		// must be after main block, otherwise margin is added on top
		// and z-index: 1 is necessary on fab
		FABs: fabs,
	}

	if isTenantPasskeyEnrollmentRequired(ctx) {
		rail.Items = tenantPasskeyEnrollmentNavigationRailItems(ctx)
		rail.FooterItems = append(rail.FooterItems, signOutNavigationRailItem(ctx))
		rail.SetActiveValue(active)
		return rail
	}

	rail.Items = primaryNavigationRailItems(ctx, infra)
	if ctx.IsSpaceCtx() {
		rail.CompactItems = spaceCompactNavigationRailItems(ctx, active)
	}
	rail.TopItems = expandedNavigationRailItems(ctx, infra)
	rail.FooterItems = footerNavigationRailItems(ctx)
	rail.SetActiveValue(active)

	return rail
}

func NewNavigationRailToggle() *wx.NavigationRailToggle {
	return &wx.NavigationRailToggle{}
}

// SpacesNavigationRailValue returns the active value for a tenant Spaces overview item.
func SpacesNavigationRailValue(tenantID string) string {
	return "spaces-" + tenantID
}

// SpaceNavigationRailValue returns the active value for a single space item.
func SpaceNavigationRailValue(spaceID string) string {
	return "space-" + spaceID
}

// TenantUsersNavigationRailValue returns the active value for a tenant Users item.
func TenantUsersNavigationRailValue(tenantID string) string {
	return "tenant-users-" + tenantID
}

// TenantSettingsNavigationRailValue returns the active value for a tenant Settings item.
func TenantSettingsNavigationRailValue(tenantID string) string {
	return "tenant-settings-" + tenantID
}

func primaryNavigationRailItems(ctx ctxx.Context, infra *common.Infra) []*wx.NavigationRailItem {
	if !ctx.IsMainCtx() {
		return []*wx.NavigationRailItem{signInNavigationRailItem(ctx)}
	}

	if ctx.IsSpaceCtx() {
		return spaceNavigationRailItems(ctx)
	}

	items := mainNavigationRailItems(ctx)
	items = appendNavigationDestinationItems(ctx, infra, items)
	return infra.PluginRegistry().ExtendNavigationRailItems(ctx, items)
}

func expandedNavigationRailItems(ctx ctxx.Context, infra *common.Infra) []*wx.NavigationRailItem {
	var items []*wx.NavigationRailItem

	if ctx.IsMainCtx() && ctx.IsSpaceCtx() {
		items = append(items, navigationRailSubheader("home", wx.T("Home").String(ctx)))
		items = append(items, mainNavigationRailItems(ctx)...)
		items = appendNavigationDestinationItems(ctx, infra, items)
		items = infra.PluginRegistry().ExtendNavigationRailItems(ctx, items)
	}

	if ctx.IsMainCtx() {
		if ctx.IsSpaceCtx() {
			items = append(items, currentTenantSpaceNavigationRailItems(ctx)...)
		} else {
			items = append(items, accountTenantNavigationRailItems(ctx)...)
		}
		items = append(items, pluginMenuNavigationRailItems(ctx, infra)...)
	}

	return items
}

func footerNavigationRailItems(ctx ctxx.Context) []*wx.NavigationRailItem {
	var items []*wx.NavigationRailItem

	if ctx.IsMainCtx() {
		items = append(items, signOutNavigationRailItem(ctx))
	}
	if !ctx.VisitorCtx().CommercialLicenseEnabled {
		items = append(items, aboutNavigationRailItem(ctx))
	}
	if ctx.IsMainCtx() && len(items) > 0 {
		items = append([]*wx.NavigationRailItem{navigationRailSubheader("misc", wx.T("Misc").String(ctx))}, items...)
	}

	return items
}

func mainNavigationRailItems(ctx ctxx.Context) []*wx.NavigationRailItem {
	items := []*wx.NavigationRailItem{
		dashboardNavigationRailItem(ctx),
	}
	if ctx.IsSpaceCtx() {
		return items
	}

	items = append(items, accountNavigationRailItem(ctx))
	if ctx.MainCtx().Account.Role == mainrole.Admin {
		items = append(items, systemNavigationRailItem(ctx))
	}

	return items
}

func spaceNavigationRailItems(ctx ctxx.Context) []*wx.NavigationRailItem {
	tenantID := ctx.SpaceCtx().TenantID
	spaceID := ctx.SpaceCtx().SpaceID

	return []*wx.NavigationRailItem{
		pageNavigationRailItem(
			"browse",
			wx.T("Files").String(ctx),
			"folder_open",
			route2.BrowseRoot(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"inbox",
			wx.T("Inbox").String(ctx),
			"inbox",
			route2.InboxRoot(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"trash",
			wx.T("Trash").String(ctx),
			"delete",
			route2.TrashRoot(tenantID, spaceID),
		),
		navigationRailSubheader("manage", wx.T("Manage space").String(ctx)),
		pageNavigationRailItem(
			"document-types",
			wx.T("Document types").String(ctx),
			"category",
			route2.ManageDocumentTypes(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"tags",
			wx.T("Tags").String(ctx),
			"label",
			route2.ManageTags(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"fields",
			wx.T("Fields").String(ctx),
			"tune",
			route2.ManageProperties(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"manage-users",
			wx.T("Users").String(ctx),
			"person",
			route2.ManageUsersOfSpace(tenantID, spaceID),
		),
	}
}

func spaceCompactNavigationRailItems(ctx ctxx.Context, active string) []*wx.NavigationRailItem {
	tenantID := ctx.SpaceCtx().TenantID
	spaceID := ctx.SpaceCtx().SpaceID

	if isMetadataNavigationRailActive(active) {
		return []*wx.NavigationRailItem{
			pageNavigationRailItem(
				"browse",
				wx.T("Files").String(ctx),
				"folder_open",
				route2.BrowseRoot(tenantID, spaceID),
			),
			pageNavigationRailItem(
				"document-types",
				wx.T("Document types").String(ctx),
				"category",
				route2.ManageDocumentTypes(tenantID, spaceID),
			),
			pageNavigationRailItem(
				"tags",
				wx.T("Tags").String(ctx),
				"label",
				route2.ManageTags(tenantID, spaceID),
			),
			pageNavigationRailItem(
				"fields",
				wx.T("Fields").String(ctx),
				"tune",
				route2.ManageProperties(tenantID, spaceID),
			),
		}
	}

	return []*wx.NavigationRailItem{
		pageNavigationRailItem(
			"browse",
			wx.T("Files").String(ctx),
			"folder_open",
			route2.BrowseRoot(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"inbox",
			wx.T("Inbox").String(ctx),
			"inbox",
			route2.InboxRoot(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"trash",
			wx.T("Trash").String(ctx),
			"delete",
			route2.TrashRoot(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"metadata",
			wx.T("Metadata").String(ctx),
			"database",
			route2.ManageDocumentTypes(tenantID, spaceID),
		),
	}
}

func isMetadataNavigationRailActive(active string) bool {
	return active == "document-types" || active == "tags" || active == "fields"
}

func tenantPasskeyEnrollmentNavigationRailItems(ctx ctxx.Context) []*wx.NavigationRailItem {
	return []*wx.NavigationRailItem{dashboardNavigationRailItem(ctx)}
}

func currentTenantSpaceNavigationRailItems(ctx ctxx.Context) []*wx.NavigationRailItem {
	spacesByTenant, err := ctx.MainCtx().ReadOnlyAccountSpacesByTenant()
	if err != nil {
		log.Println(err)
		return []*wx.NavigationRailItem{}
	}

	tenants := make([]*entmain.Tenant, 0, len(spacesByTenant))
	for tenantx := range spacesByTenant {
		tenants = append(tenants, tenantx)
	}
	sort.Slice(tenants, func(i, j int) bool {
		return tenants[i].Name < tenants[j].Name
	})

	var items []*wx.NavigationRailItem
	for _, tenantx := range tenants {
		spaces := spacesByTenant[tenantx]
		if len(spaces) == 0 {
			continue
		}
		sort.Slice(spaces, func(i, j int) bool {
			return spaces[i].Name < spaces[j].Name
		})

		children := make([]*wx.NavigationRailItem, 0, len(spaces))
		for _, spacex := range spaces {
			spaceID := spacex.PublicID.String()
			icon := "check_box_outline_blank"
			if ctx.IsSpaceCtx() && ctx.SpaceCtx().SpaceID == spaceID {
				icon = "check_box"
			}

			children = append(children, pageNavigationRailItem(
				SpaceNavigationRailValue(spaceID),
				wx.Tu(spacex.Name).String(ctx),
				icon,
				route2.BrowseRoot(tenantx.PublicID.String(), spaceID),
			))
		}

		items = append(items, &wx.NavigationRailItem{
			Key:                 "tenant-spaces-" + tenantx.PublicID.String(),
			Label:               wx.Tu(tenantx.Name).String(ctx),
			Icon:                "business",
			Children:            children,
			IsCollapsible:       true,
			IsExpandedByDefault: isCurrentSpaceTenant(ctx, tenantx),
		})
	}
	if len(items) == 0 {
		return items
	}

	return append(
		[]*wx.NavigationRailItem{navigationRailSubheader("spaces-by-organization", wx.T("Spaces by Organization").String(ctx))},
		items...,
	)
}

func isCurrentSpaceTenant(ctx ctxx.Context, tenantx *entmain.Tenant) bool {
	return ctx.IsSpaceCtx() && tenantx.PublicID.String() == ctx.SpaceCtx().TenantID
}

func accountTenantNavigationRailItems(ctx ctxx.Context) []*wx.NavigationRailItem {
	tenants, err := ctx.MainCtx().Account.QueryTenants().
		Order(maintenant.ByName()).
		All(ctx)
	if err != nil {
		log.Println(err)
		return []*wx.NavigationRailItem{}
	}

	var items []*wx.NavigationRailItem
	for _, tenantx := range tenants {
		tenantID := tenantx.PublicID.String()
		children := tenantNavigationRailChildren(ctx, tenantx)
		if len(children) == 0 {
			continue
		}

		items = append(items, &wx.NavigationRailItem{
			Key:                 "tenant-" + tenantID,
			Label:               wx.Tu(tenantx.Name).String(ctx),
			Icon:                "business",
			Children:            children,
			IsCollapsible:       true,
			IsExpandedByDefault: true,
		})
	}

	if len(items) == 0 {
		return items
	}

	return append(
		[]*wx.NavigationRailItem{navigationRailSubheader("organizations", wx.T("Organizations").String(ctx))},
		items...,
	)
}

func tenantNavigationRailChildren(
	ctx ctxx.Context,
	tenantx *entmain.Tenant,
) []*wx.NavigationRailItem {
	if tenantx == nil {
		return []*wx.NavigationRailItem{}
	}

	tenantID := tenantx.PublicID.String()
	tenantm := tenantmodel.NewTenant(tenantx)
	if !tenantm.IsInitialized() {
		return []*wx.NavigationRailItem{}
	}

	items := []*wx.NavigationRailItem{
		pageNavigationRailItem(
			SpacesNavigationRailValue(tenantID),
			wx.T("Spaces").String(ctx),
			"hub",
			route2.SpacesRoot(tenantID),
		),
	}
	if canManageTenantUsers(ctx, tenantx) {
		items = append(
			items,
			pageNavigationRailItem(
				TenantUsersNavigationRailValue(tenantID),
				wx.T("Users").String(ctx),
				"person",
				route2.ManageUsersOfTenant(tenantID),
			),
			pageNavigationRailItem(
				TenantSettingsNavigationRailValue(tenantID),
				wx.T("Settings").String(ctx),
				"settings",
				route2.OrganizationSettings(tenantID),
			),
		)
	}

	return items
}

func canManageTenantUsers(ctx ctxx.Context, tenantx *entmain.Tenant) bool {
	if tenantx == nil {
		return false
	}
	tenantm := tenantmodel.NewTenant(tenantx)
	accountm := account.NewAccount(ctx.MainCtx().Account)
	return tenantm.IsOwner(accountm) && tenantm.IsInitialized()
}

func appendNavigationDestinationItems(
	ctx ctxx.Context,
	infra *common.Infra,
	items []*wx.NavigationRailItem,
) []*wx.NavigationRailItem {
	for _, destination := range infra.PluginRegistry().ExtendNavigationDestinations(ctx, nil) {
		items = append(items, wx.NewNavigationRailItemFromDestination(destination))
	}
	return items
}

func pluginMenuNavigationRailItems(
	ctx ctxx.Context,
	infra *common.Infra,
) []*wx.NavigationRailItem {
	var items []*wx.NavigationRailItem
	for _, item := range infra.PluginRegistry().ExtendMenuItems(ctx, nil) {
		if item == nil || item.IsDivider || item.Label == nil || item.DownloadLinkURL != "" {
			continue
		}
		label := item.Label.String(ctx)
		if label == "" {
			continue
		}
		items = append(items, &wx.NavigationRailItem{
			HTMXAttrs:  item.HTMXAttrs,
			Key:        "plugin-menu-" + label,
			Value:      "plugin-menu-" + label,
			Label:      label,
			Icon:       item.LeadingIcon,
			IsDisabled: item.IsDisabled,
		})
	}
	return items
}

func dashboardNavigationRailItem(ctx ctxx.Context) *wx.NavigationRailItem {
	return pageNavigationRailItem(
		"dashboard",
		wx.T("Dashboard").String(ctx),
		"dashboard",
		route2.Dashboard(),
	)
}

func accountNavigationRailItem(ctx ctxx.Context) *wx.NavigationRailItem {
	return pageNavigationRailItem(
		"account",
		wx.T("Account").String(ctx),
		"account_circle",
		route2.Account(),
	)
}

func systemNavigationRailItem(ctx ctxx.Context) *wx.NavigationRailItem {
	return pageNavigationRailItem(
		"system",
		wx.T("System").String(ctx),
		"settings",
		route2.System(),
	)
}

func signInNavigationRailItem(ctx ctxx.Context) *wx.NavigationRailItem {
	return &wx.NavigationRailItem{
		Key:   "sign-in",
		Value: "sign-in",
		Label: wx.T("Sign in [subject]").String(ctx),
		Icon:  "login",
		Href:  "/",
	}
}

func signOutNavigationRailItem(ctx ctxx.Context) *wx.NavigationRailItem {
	return &wx.NavigationRailItem{
		Key:   "sign-out",
		Value: "sign-out",
		Label: wx.T("Sign out").String(ctx),
		Icon:  "logout",
		HTMXAttrs: wx.HTMXAttrs{
			HxPost: route2.SignOutCmd(),
		},
	}
}

func aboutNavigationRailItem(ctx ctxx.Context) *wx.NavigationRailItem {
	return pageNavigationRailItem(
		"about",
		wx.T("About SimpleDMS").String(ctx),
		"info",
		route2.AboutPage(),
	)
}

func navigationRailSubheader(key string, label string) *wx.NavigationRailItem {
	return &wx.NavigationRailItem{
		Key:         key,
		Label:       label,
		IsSubheader: true,
	}
}

func pageNavigationRailItem(key string, label string, icon string, href string) *wx.NavigationRailItem {
	return &wx.NavigationRailItem{
		Key:   key,
		Value: key,
		Label: label,
		Icon:  icon,
		HTMXAttrs: wx.HTMXAttrs{
			HxGet: href,
		},
	}
}

func isTenantPasskeyEnrollmentRequired(ctx ctxx.Context) bool {
	if !ctx.IsMainCtx() {
		return false
	}
	if !ctx.VisitorCtx().IsTemporarySession {
		return false
	}

	accountm := account.NewAccount(ctx.MainCtx().Account)
	passkeyPolicy, err := accountm.PasskeyPolicy(ctx)
	if err != nil {
		log.Println(err)
		passkeyPolicy = account.NewPasskeyPolicy(false, false, false)
	}
	return passkeyPolicy.IsTenantPasskeyEnrollmentRequired()
}
