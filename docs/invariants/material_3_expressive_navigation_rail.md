# Material 3 Expressive Navigation Rail Invariants

## Scope

These invariants apply to the SimpleDMS Material 3 Expressive navigation rail,
rail item model, mobile rail toggle, rail runtime, app-bar integration, plugin
navigation extension points, metadata destinations, side-sheet interactions, and
adaptive compact/medium/expanded layouts.

## Invariants

### Collapsed Server Default

Rule: The server-rendered rail must be usable in collapsed state before
JavaScript runs.

Why: Navigation must remain available during slow script loading, failed script
loading, and first paint.

Enforced in: `ui/widget/navigation_rail.gohtml`, `ui/widget/navigation_rail.go`,
and no server-rendered `navigation-rail-expanded` default state.

Verified by: Rendering checks and manual loading with JavaScript disabled or
delayed.

### Compact Collapsed Rail Default

Rule: Below the SimpleDMS `md=600px` breakpoint, the collapsed rail is the
default closed mobile navigation surface.

Why: The requested mobile behavior keeps the current collapsed rail as the
default and uses expanded rail as the menu replacement.

Enforced in: Compact selectors in `ui/widget/navigation_rail.gohtml` and the
navigation rail runtime.

Verified by: Compact viewport browser checks after reload and HTMX navigation.

### Compact Expanded Rail Is Modal

Rule: On compact/mobile screens, the expanded navigation rail overlays content
with a scrim and does not push, resize, or relayout page content.

Why: Compact screens do not have enough horizontal space for a standard expanded
rail beside content.

Enforced in: Compact-only expanded rail styles and runtime state handling.

Verified by: Compact browser checks for open, closed, scrim, scroll, and content
positioning states.

### Mobile Expansion Is Not Persistent

Rule: Compact/mobile expanded state must not be read from or written to
localStorage.

Why: A modal navigation surface should not unexpectedly cover content after
reload, HTMX navigation, or return visits.

Enforced in: Breakpoint-aware runtime logic using the `md=600px` media query.

Verified by: Open the rail on compact, reload or navigate, and confirm it starts
closed.

### Medium And Larger Rail Is Standard

Rule: On `md+` screens, the expanded rail is a standard non-modal rail that
occupies layout width beside content and does not show a scrim.

Why: Medium and larger screens can support persistent navigation without covering
body content.

Enforced in: `md` breakpoint selectors in the rail template and runtime.

Verified by: Browser checks at medium, expanded, large, and extra-large widths.

### Desktop Expansion Persistence

Rule: On `md+` screens, expanded/collapsed rail state and expanded group
disclosure are browser-local UI preferences stored in localStorage.

Why: Larger-screen users benefit from a persistent navigation density preference.

Enforced in: `ui/uix/web/assets/navigation_rail.js` using SimpleDMS-specific
storage keys.

Verified by: Toggle rail expansion and groups on desktop, reload, and confirm
state restores.

Edge cases: Unavailable localStorage must fail quietly because the state is only
a UI preference.

### Navigation Rail Is The Primary Navigation Surface

Rule: Standard app bars and rail-local menu buttons must use the expanded rail
toggle, not the old popup `NewMainMenu`, for primary navigation.

Why: M3 Expressive expanded rail replaces the navigation drawer/main menu and
prevents duplicate, divergent navigation sources.

Enforced in: App-bar `LeadingAltMobile` call sites, `AppBar.GetSearch()`, and
`partial.NewNavigationRail`.

Verified by: Code search for standard `partial.NewMainMenu(...)` app-bar usage
and manual mobile checks.

Edge cases: If `NewMainMenu` remains for a documented non-standard use, that use
must not be reachable as the default mobile navigation menu.

### Destination Parity

Rule: Removing the popup main menu must not remove a destination that was visible
to the current user under the same context and policy rules.

Why: The expanded rail is a replacement, not a navigation reduction.

Enforced in: `ui/uix/partial/navigation_rail.go`, rail item helpers, plugin rail
hooks, and migration away from menu-only destinations.

Verified by: Comparing old `NewMainMenu` output with new expanded rail output in
visitor, main, admin, Space, unlicensed, and plugin contexts.

### Required Passkey Enrollment Gate

Rule: Required tenant passkey enrollment must keep the same navigation
restrictions that `NewMainMenu` currently applies.

Why: Replacing the menu must not let users bypass required account/security
actions.

Enforced in: Rail item construction before mobile app-bar entries are replaced.

Verified by: Rendering tests and manual checks with an account requiring tenant
passkey enrollment.

### Destination Authorization

Rule: Rendering or hiding a rail destination must not be treated as a security
boundary.

Why: Users can still request URLs directly. Authorization belongs in target page,
action, and command handlers.

Enforced in: Existing route/action authorization and command handlers.

Verified by: Existing authorization tests and direct requests to restricted
routes.

### Expanded Rail Sections

Rule: The expanded rail groups dashboard/global entry points under `Home`, tenant
space entries under `Spaces`, Space administration entries under `Manage space`, and
footer actions such as sign out/about under `Misc`.

Why: Expanded rail labels should make long navigation lists scannable without
affecting compact or collapsed rail item limits.

Enforced in: `ui/uix/partial/navigation_rail.go` subheader item construction and
`NavigationRailItemExpandedSubheader` rendering.

Verified by: Rendering tests for tenant-owner dashboard rails, Space rails, and
footer labels.

### Tenant User Management Destination

Rule: The dashboard/main navigation group shows a tenant `Users` destination only
when a dashboard context has exactly one initialized tenant owned by the account.
Generic tenant/no-Space contexts keep account, tenant-user, and system
destinations out of the primary rail group, except the tenant user management
page itself must show the current tenant's `Users` destination and mark it
active. Space contexts keep account, tenant-user, and system destinations out of
the active Space navigation group; Space `Users` is a separate Space-scoped
destination under `Manage space`.

Why: User management is tenant-scoped, so the rail must not expose an ambiguous
tenant-user destination when no current tenant can be inferred. Space navigation
should stay focused on the selected tenant/Space rather than mixing in unrelated
account and system destinations. The tenant user management page already has an
unambiguous current tenant, so hiding its `Users` destination leaves the selected
page without a matching rail/menu item.

Enforced in: `primaryNavigationRailItems`, `mainNavigationRailItems`,
`nilableManageTenantUsersNavigationRailItem`, `nilableManageUsersTenant`, and
`ManageUsersOfTenantPage` using the `tenant-users` active value.

Verified by: Rendering tests for owner and non-owner tenant assignments and for
tenant context without a selected Space, plus an explicit tenant user management
page check that the current tenant `Users` destination is present and active.

### Metadata Destinations Are Direct

Rule: In Space context, `Document types`, `Tags`, `Fields`, and Space `Users` are
direct rail destinations under `Manage space` in the expanded rail. `Trash` appears
immediately after `Inbox`. Compact/mobile collapsed rail keeps the legacy
`Metadata` entry on normal file pages and switches to `Files`, `Document types`,
`Tags`, and `Fields` while metadata pages are active.

Why: The current metadata rail item should be reimplemented so all subitems are
directly accessible.

Enforced in: Space rail item construction.

Verified by: Rendering checks for Browse, Inbox, document type, tag, and field
pages.

Edge cases: Expanded rail may group these items under a `Metadata` subheader or
collapsible group, but the children remain directly actionable and carry active
state.

### Compact Item Limit

Rule: The compact closed rail should render at most four actionable leaf
destinations for the current context.

Why: Compact/mobile screens do not have enough inline space for more than four
bottom destinations. Additional destinations remain available through the modal
expanded rail.

Enforced in: `NavigationRail.CompactNavigationItems()` and compact rail template
regions.

Verified by: Rendering tests and compact browser checks for main and Space
contexts.

### Medium Collapsed Item Limit

Rule: The `md+` collapsed side rail should render 3-7 actionable leaf
destinations for the current context whenever that many destinations are
available.

Why: M3 guidance limits collapsed rails to 3-7 primary destinations and keeps
secondary destinations in the expanded rail.

Enforced in: `NavigationRail.CollapsedItems()` and `md+` collapsed rail template
regions.

Verified by: Rendering tests and browser checks for main and Space contexts.

Edge cases: Contexts with fewer than three valid destinations, such as visitor or
restricted passkey enrollment states, may render fewer rather than inventing
invalid destinations.

### Single Active Destination

Rule: Exactly one leaf rail destination should be active for the current page
when a matching destination exists.

Why: The M3 active indicator communicates the current open page and should not
appear on multiple destinations at once.

Enforced in: Active-value matching on rail item trees.

Verified by: Rendering checks for all page active keys.

Edge cases: A parent group may visually indicate it contains the active child,
but it must not also render as the active leaf destination.

### Disabled Items Are Non-Interactive

Rule: Disabled rail items must not expose `href`, HTMX activation attributes, or
keyboard button activation behavior.

Why: Disabled destinations may be discoverable, but must not activate
accidentally.

Enforced in: Rail item helper methods and item templates.

Verified by: Rendering checks and manual keyboard/pointer checks.

### HTMX Remains The Server-State Path

Rule: Any rail-triggered server-side state change must use HTMX and existing
action handlers.

Why: The backend remains the source of truth for application state; rail
JavaScript manages only local UI state.

Enforced in: Rail item `HTMXAttrs`, route/action handlers, and avoiding business
state mutation in `navigation_rail.js`.

Verified by: Code review and manual sign-out/navigation checks.

### Mobile Destination Activation Closes The Rail

Rule: Activating a rail destination on compact/mobile closes the expanded rail
before or during navigation.

Why: After choosing a destination, the user should see the resulting page or HTMX
update instead of a stale modal overlay.

Enforced in: Runtime click handling for rail anchors and HTMX buttons when below
`md=600px`.

Verified by: Compact browser checks with normal anchors, `HxGet`, and `HxPost`
items.

Edge cases: Disclosure group toggles are not destination activation and must not
close the overlay.

### Mobile Dismissal Paths

Rule: Compact/mobile expanded rail must close through the app-bar toggle,
rail-local toggle, scrim click, and Escape key.

Why: Touch, mouse, and keyboard users need predictable dismissal paths for a
modal surface.

Enforced in: Runtime event handling and synchronized ARIA state.

Verified by: Manual compact pointer and keyboard checks.

Edge cases: Browser-back dismissal is intentionally outside the first
implementation pass.

### Rail Width Is A Layout Contract

Rule: The current rail width must be exposed as a CSS custom property and kept in
sync with compact, collapsed `md+`, and expanded `md+` states.

Why: Content and fixed large-screen surfaces such as side sheets need a stable
way to account for the rail's inline size.

Enforced in: Rail runtime and layout/side-sheet styles.

Verified by: Browser checks at compact, medium, large, `xl`, and `2xl` widths.

### Side Sheets Respect Expanded Rail Width

Rule: Large-screen side sheets must not overlap the standard expanded rail or use
hard-coded positions that become wrong when the rail expands.

Why: Expanded navigation changes available content width and the content start
edge.

Enforced in: `ui/widget/dialog.gohtml`, `ui/widget/main_layout.gohtml`, and any
side-sheet positioning styles that use the rail width custom property.

Verified by: Manual browse/details/filter side-sheet checks with collapsed and
expanded rails.

### Active Indicator And State Layers Stay Separate

Rule: Selected rail item background/indicator must remain separate from hover,
focus, and pressed state-layer overlays.

Why: Hover and focus states must not replace or hide the selected indicator.

Enforced in: Collapsed and expanded rail item template structure.

Verified by: Visual checks for selected items across pointer and keyboard states.

### Labels Stay Readable

Rule: Rail labels must be short, readable, and not truncated with ellipses.

Why: M3 rail labels are an accessibility and comprehension aid, not decorative
metadata.

Enforced in: Label choices, wrapping classes, and item template constraints.

Verified by: Browser checks across supported languages and breakpoints.

Edge cases: If an existing label such as `Document types` cannot be shortened
without translation work, controlled two-line wrapping is preferable to
truncation or smaller type.

### Plugin Rail Items Are First-Class

Rule: Plugin-provided navigation should target rail items rather than popup menu
items for primary navigation.

Why: The expanded rail is the primary navigation surface after the migration.

Enforced in: A rail item extension hook and compatibility bridge during
migration.

Verified by: Plugin rendering tests and code search for menu-only plugin
destinations.

## Cross-References

- `docs/plans/20260529_material_3_expressive_navigation_rail_plan.md`
- `docs/plans/20260529_material_3_expressive_navigation_rail_checklist.md`
- `ui/uix/partial/navigation_rail.go`
- `ui/uix/partial/main_menu.go`
- `ui/widget/navigation_rail.go`
- `ui/widget/navigation_rail.gohtml`
- `ui/widget/navigation_destination.go`
- `ui/widget/navigation_destination.gohtml`
- `ui/widget/app_bar.go`
- `ui/widget/app_bar.gohtml`
- `ui/widget/dialog.gohtml`
