package partial

import (
	"log"
	"sort"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	maintenant "github.com/simpledms/simpledms/db/entmain/tenant"
	"github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	tenantmodel "github.com/simpledms/simpledms/model/main/tenant"
	route2 "github.com/simpledms/simpledms/ui/uix/route"
)

// fab must be injected because it differs on each page...
func NewNavigationRail(
	ctx ctxx.Context,
	infra *common.Infra,
	active string,
	fabs []*widget.FloatingActionButton,
) *widget.NavigationRail {
	rail := &widget.NavigationRail{
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
	if ctx.IsMainCtx() {
		rail.ExpandedSelector = spaceCombobox(ctx, active)
	}
	rail.TopItems = expandedNavigationRailItems(ctx, infra)
	rail.FooterItems = footerNavigationRailItems(ctx, infra, active)
	rail.SetActiveValue(active)

	return rail
}

func NewNavigationRailToggle() *widget.NavigationRailToggle {
	return &widget.NavigationRailToggle{}
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

func primaryNavigationRailItems(ctx ctxx.Context, infra *common.Infra) []*widget.NavigationRailItem {
	if !ctx.IsMainCtx() {
		return []*widget.NavigationRailItem{signInNavigationRailItem(ctx)}
	}

	if ctx.IsSpaceCtx() {
		return spaceNavigationRailItems(ctx)
	}

	items := mainNavigationRailItems(ctx)
	items = appendNavigationDestinationItems(ctx, infra, items)
	return infra.PluginRegistry().ExtendNavigationRailItems(ctx, items)
}

func expandedNavigationRailItems(ctx ctxx.Context, infra *common.Infra) []*widget.NavigationRailItem {
	var items []*widget.NavigationRailItem

	if ctx.IsMainCtx() && ctx.IsSpaceCtx() {
		homeItems := appendNavigationDestinationItems(ctx, infra, nil)
		homeItems = infra.PluginRegistry().ExtendNavigationRailItems(ctx, homeItems)
		if len(homeItems) > 0 {
			items = append(items, navigationRailSubheader("home", widget.T("Home").String(ctx)))
			items = append(items, homeItems...)
		}
	}

	if ctx.IsMainCtx() {
		if ctx.IsSpaceCtx() {
			items = append(items, pluginMenuNavigationRailItems(ctx, infra)...)
		} else {
			items = append(
				items,
				accountTenantNavigationRailItems(ctx, pluginMenuNavigationRailItems(ctx, infra))...,
			)
		}
	}

	return items
}

func footerNavigationRailItems(
	ctx ctxx.Context,
	infra *common.Infra,
	active string,
) []*widget.NavigationRailItem {
	var items []*widget.NavigationRailItem

	if ctx.IsMainCtx() {
		items = append(items, signOutNavigationRailItem(ctx))
	}
	if !ctx.VisitorCtx().CommercialLicenseEnabled {
		items = append(items, aboutNavigationRailItem(ctx))
	}
	if ctx.IsMainCtx() && len(items) > 0 {
		items = append(
			[]*widget.NavigationRailItem{navigationRailSubheader("misc", widget.T("Misc").String(ctx))},
			items...,
		)
	}

	return infra.PluginRegistry().ExtendNavigationRailFooterItems(
		ctx,
		items,
		active,
	)
}

func mainNavigationRailItems(ctx ctxx.Context) []*widget.NavigationRailItem {
	var items []*widget.NavigationRailItem
	if ctx.IsSpaceCtx() {
		return items
	}

	items = append(items, accountNavigationRailItem(ctx))
	if ctx.MainCtx().Account.Role == mainrole.Admin {
		items = append(items, systemNavigationRailItem(ctx))
	}

	return items
}

func spaceNavigationRailItems(ctx ctxx.Context) []*widget.NavigationRailItem {
	tenantID := ctx.SpaceCtx().TenantID
	spaceID := ctx.SpaceCtx().SpaceID

	return []*widget.NavigationRailItem{
		pageNavigationRailItem(
			"browse",
			widget.T("Files").String(ctx),
			"folder_open",
			route2.BrowseRoot(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"inbox",
			widget.T("Inbox").String(ctx),
			"inbox",
			route2.InboxRoot(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"trash",
			widget.T("Trash").String(ctx),
			"delete",
			route2.TrashRoot(tenantID, spaceID),
		),
		navigationRailSubheader("manage", widget.T("Manage space").String(ctx)),
		pageNavigationRailItem(
			"document-types",
			widget.T("Document types").String(ctx),
			"category",
			route2.ManageDocumentTypes(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"tags",
			widget.T("Tags").String(ctx),
			"label",
			route2.ManageTags(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"fields",
			widget.T("Fields").String(ctx),
			"tune",
			route2.ManageProperties(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"manage-users",
			widget.T("Users").String(ctx),
			"person",
			route2.ManageUsersOfSpace(tenantID, spaceID),
		),
	}
}

func spaceCompactNavigationRailItems(ctx ctxx.Context, active string) []*widget.NavigationRailItem {
	tenantID := ctx.SpaceCtx().TenantID
	spaceID := ctx.SpaceCtx().SpaceID

	if isMetadataNavigationRailActive(active) {
		return []*widget.NavigationRailItem{
			pageNavigationRailItem(
				"browse",
				widget.T("Files").String(ctx),
				"folder_open",
				route2.BrowseRoot(tenantID, spaceID),
			),
			pageNavigationRailItem(
				"document-types",
				widget.T("Document types").String(ctx),
				"category",
				route2.ManageDocumentTypes(tenantID, spaceID),
			),
			pageNavigationRailItem(
				"tags",
				widget.T("Tags").String(ctx),
				"label",
				route2.ManageTags(tenantID, spaceID),
			),
			pageNavigationRailItem(
				"fields",
				widget.T("Fields").String(ctx),
				"tune",
				route2.ManageProperties(tenantID, spaceID),
			),
		}
	}

	return []*widget.NavigationRailItem{
		pageNavigationRailItem(
			"browse",
			widget.T("Files").String(ctx),
			"folder_open",
			route2.BrowseRoot(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"inbox",
			widget.T("Inbox").String(ctx),
			"inbox",
			route2.InboxRoot(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"trash",
			widget.T("Trash").String(ctx),
			"delete",
			route2.TrashRoot(tenantID, spaceID),
		),
		pageNavigationRailItem(
			"metadata",
			widget.T("Metadata").String(ctx),
			"database",
			route2.ManageDocumentTypes(tenantID, spaceID),
		),
	}
}

func isMetadataNavigationRailActive(active string) bool {
	return active == "document-types" || active == "tags" || active == "fields"
}

func tenantPasskeyEnrollmentNavigationRailItems(ctx ctxx.Context) []*widget.NavigationRailItem {
	return []*widget.NavigationRailItem{dashboardNavigationRailItem(ctx)}
}

func spaceCombobox(ctx ctxx.Context, active string) *widget.Combobox {
	spacesByTenant, err := ctx.MainCtx().ReadOnlyAccountSpacesByTenant()
	if err != nil {
		log.Println(err)
		return nil
	}

	items := []*widget.MenuItem{
		{
			LeadingIcon: "dashboard",
			Label:       widget.T("Dashboard"),
			IsSelected:  active == "dashboard",
			HTMXAttrs: widget.HTMXAttrs{
				HxGet: route2.Dashboard(),
			},
		},
	}
	tenants := make([]*entmain.Tenant, 0, len(spacesByTenant))
	for tenantx := range spacesByTenant {
		tenants = append(tenants, tenantx)
	}
	sort.Slice(tenants, func(i, j int) bool { return tenants[i].Name < tenants[j].Name })

	for _, tenantx := range tenants {
		spaces := spacesByTenant[tenantx]
		sort.Slice(spaces, func(i, j int) bool { return spaces[i].Name < spaces[j].Name })
		if len(spaces) > 0 {
			items = append(items, &widget.MenuItem{
				Label:       widget.Tu(tenantx.Name),
				IsSubheader: true,
			})
		}
		for _, spacex := range spaces {
			spaceID := spacex.PublicID.String()
			items = append(items, &widget.MenuItem{
				LeadingIcon: "workspaces",
				Label:       widget.Tu(spacex.Name),
				IsSelected:  ctx.IsSpaceCtx() && ctx.SpaceCtx().SpaceID == spaceID,
				HTMXAttrs: widget.HTMXAttrs{
					HxGet: route2.BrowseRoot(tenantx.PublicID.String(), spaceID),
				},
			})
		}
	}

	placeholder := widget.T("Dashboard").String(ctx) + " / " + widget.T("Select space").String(ctx)
	selectedIcon := widget.NewIcon("dashboard")
	if active == "dashboard" {
		placeholder = widget.T("Dashboard").String(ctx)
	}
	if ctx.IsSpaceCtx() {
		placeholder = ctx.SpaceCtx().Space.Name
		selectedIcon = widget.NewIcon("workspaces")
	}

	return &widget.Combobox{
		Input: &widget.Input{
			Placeholder:  placeholder,
			LeadingIcon:  selectedIcon,
			TrailingIcon: widget.NewIcon("expand_more"),
		},
		Menu: &widget.Menu{
			Items:              items,
			EmptyLabel:         widget.T("No matches found."),
			MatchesAnchorWidth: true,
			IsAutoPopover:      true,
		},
	}
}

func accountTenantNavigationRailItems(
	ctx ctxx.Context,
	leadingItems []*widget.NavigationRailItem,
) []*widget.NavigationRailItem {
	items := append([]*widget.NavigationRailItem{}, leadingItems...)
	tenants, err := ctx.MainCtx().Account.QueryTenants().
		Order(maintenant.ByName()).
		All(ctx)
	if err != nil {
		log.Println(err)
		return accountTenantNavigationRailSection(ctx, items)
	}

	for _, tenantx := range tenants {
		tenantID := tenantx.PublicID.String()
		children := tenantNavigationRailChildren(ctx, tenantx)
		if len(children) == 0 {
			continue
		}

		items = append(items, &widget.NavigationRailItem{
			Key:           "tenant-" + tenantID,
			Label:         widget.Tu(tenantx.Name).String(ctx),
			Icon:          "business",
			Children:      children,
			IsCollapsible: true,
		})
	}

	return accountTenantNavigationRailSection(ctx, items)
}

func accountTenantNavigationRailSection(
	ctx ctxx.Context,
	items []*widget.NavigationRailItem,
) []*widget.NavigationRailItem {
	if len(items) == 0 {
		return items
	}

	return append(
		[]*widget.NavigationRailItem{navigationRailSubheader("organizations", widget.T("Organizations").String(ctx))},
		items...,
	)
}

func tenantNavigationRailChildren(
	ctx ctxx.Context,
	tenantx *entmain.Tenant,
) []*widget.NavigationRailItem {
	if tenantx == nil {
		return []*widget.NavigationRailItem{}
	}

	tenantID := tenantx.PublicID.String()
	tenantm := tenantmodel.NewTenant(tenantx)
	if !tenantm.IsInitialized() {
		return []*widget.NavigationRailItem{}
	}

	items := []*widget.NavigationRailItem{
		pageNavigationRailItem(
			SpacesNavigationRailValue(tenantID),
			widget.T("Spaces").String(ctx),
			"hub",
			route2.SpacesRoot(tenantID),
		),
	}
	if canManageTenantUsers(ctx, tenantx) {
		items = append(
			items,
			pageNavigationRailItem(
				TenantUsersNavigationRailValue(tenantID),
				widget.T("Users").String(ctx),
				"person",
				route2.ManageUsersOfTenant(tenantID),
			),
			pageNavigationRailItem(
				TenantSettingsNavigationRailValue(tenantID),
				widget.T("Settings").String(ctx),
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
	items []*widget.NavigationRailItem,
) []*widget.NavigationRailItem {
	for _, destination := range infra.PluginRegistry().ExtendNavigationDestinations(ctx, nil) {
		items = append(items, widget.NewNavigationRailItemFromDestination(destination))
	}
	return items
}

func pluginMenuNavigationRailItems(
	ctx ctxx.Context,
	infra *common.Infra,
) []*widget.NavigationRailItem {
	var items []*widget.NavigationRailItem
	for _, item := range infra.PluginRegistry().ExtendMenuItems(ctx, nil) {
		if item == nil || item.IsDivider || item.Label == nil || item.DownloadLinkURL != "" {
			continue
		}
		label := item.Label.String(ctx)
		if label == "" {
			continue
		}
		value := "plugin-menu-" + label
		if item.RadioValue != "" {
			value = item.RadioValue
		}
		items = append(items, &widget.NavigationRailItem{
			HTMXAttrs:  item.HTMXAttrs,
			Key:        value,
			Value:      value,
			Label:      label,
			Icon:       item.LeadingIcon,
			IsDisabled: item.IsDisabled,
		})
	}
	return items
}

func dashboardNavigationRailItem(ctx ctxx.Context) *widget.NavigationRailItem {
	return pageNavigationRailItem(
		"dashboard",
		widget.T("Dashboard").String(ctx),
		"dashboard",
		route2.Dashboard(),
	)
}

func accountNavigationRailItem(ctx ctxx.Context) *widget.NavigationRailItem {
	return pageNavigationRailItem(
		"account",
		widget.T("Account").String(ctx),
		"account_circle",
		route2.Account(),
	)
}

func systemNavigationRailItem(ctx ctxx.Context) *widget.NavigationRailItem {
	return pageNavigationRailItem(
		"system",
		widget.T("System").String(ctx),
		"settings",
		route2.System(),
	)
}

func signInNavigationRailItem(ctx ctxx.Context) *widget.NavigationRailItem {
	return pageNavigationRailItem(
		"sign-in",
		widget.T("Sign in [subject]").String(ctx),
		"login",
		"/",
	)
}

func signOutNavigationRailItem(ctx ctxx.Context) *widget.NavigationRailItem {
	return &widget.NavigationRailItem{
		Key:   "sign-out",
		Value: "sign-out",
		Label: widget.T("Sign out").String(ctx),
		Icon:  "logout",
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: route2.SignOutCmd(),
		},
	}
}

func aboutNavigationRailItem(ctx ctxx.Context) *widget.NavigationRailItem {
	return pageNavigationRailItem(
		"about",
		widget.T("About SimpleDMS").String(ctx),
		"info",
		route2.AboutPage(),
	)
}

func navigationRailSubheader(key string, label string) *widget.NavigationRailItem {
	return widget.NewNavigationRailLabel(key, label)
}

func pageNavigationRailItem(key string, label string, icon string, href string) *widget.NavigationRailItem {
	return &widget.NavigationRailItem{
		Key:   key,
		Value: key,
		Label: label,
		Icon:  icon,
		HTMXAttrs: widget.HTMXAttrs{
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
