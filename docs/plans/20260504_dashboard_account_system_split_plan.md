# Dashboard, Account, And System Split Plan

## Business Domain And Outcome

SimpleDMS currently uses the Dashboard as a mixed landing page for work reminders,
organisation access, account security, and system administration. The split creates clearer
navigation for logged-in users and separates personal account management from system-level
administration.

Primary actors:

- Regular account members who need a focused Dashboard and Account page.
- Organisation owners who manage organisation-related actions from the Dashboard.
- System admins who need System and Tenants administration entries.

## Goal

Split the existing dashboard into `Dashboard`, `Account`, and `System` pages, add a main-context
navigation rail with those entries, and add a `Tenants` entry for system admins. Keep
organisation cards and related organisation content on Dashboard. Move organisation action
buttons above the organisation card grids.

## Constraints And Assumptions

- Open task cards such as set password and clear temporary password stay on Dashboard.
- Account/passkey management content moves to Account.
- System cards and global admin actions move to System.
- System page and System navigation are visible only to system admins.
- Tenants is a fourth navigation option only for system admins.
- The Tenants page already exists outside the local dashboard action package and is currently
  surfaced through dashboard main-menu plugin extension.
- Organisation content stays on Dashboard, including organisation cards, quota usage, manage
  spaces prompts, and Space cards.
- Organisation action buttons such as passkey enforcement, manage spaces, manage users, delete,
  and backup move above each organisation card grid.
- Prefer the existing Go widget, Go Template, HTMX, and route patterns. Do not introduce Lit.

## Ubiquitous Language

- `Dashboard`: logged-in landing page for open tasks and organisation access.
- `Account`: personal account and passkey management page.
- `System`: system admin page for global app state and global settings.
- `Tenants`: existing system-admin tenant management page exposed by the host/plugin layer.
- `Organisation`: current user-facing name for tenant-related dashboard sections.
- `Organisation actions`: owner/admin actions that apply to one organisation, rendered above
  that organisation's cards.

## Subdomains

- `Dashboard navigation`: supporting subdomain. It improves product structure but does not carry
  core document-management rules.
- `Account security`: supporting subdomain. Existing passkey/password logic should be reused and
  only moved to a new page boundary.
- `System administration`: supporting subdomain. Existing admin actions should move behind a
  clearer page and role check.
- `Tenant management`: external or plugin-provided supporting subdomain. Core should expose a
  navigation extension point instead of duplicating the existing Tenants page.

## Bounded Contexts

One application bounded context is sufficient for the local changes. The Tenants entry crosses
into the existing plugin/host integration that currently extends the main menu; keep that as an
extension boundary rather than moving tenant-management implementation into `action/dashboard`.

## Context Map

- Core SimpleDMS dashboard code is upstream for the shared page layout and navigation rail.
- The plugin/host tenant-management module is downstream for the actual Tenants page URL and
  behavior.
- Add or reuse a published navigation extension point so plugins can contribute main-context rail
  destinations without leaking plugin routes into core dashboard rendering.

## Modelling Approach

Use thin page/partial structs and transaction-script style request handlers. This feature is a UI
recomposition of existing data and commands, not a new domain model. Keep business checks in the
existing model methods such as `account.NewAccount`, `tenant.NewTenant`, `PasskeyPolicy`, and
system configuration helpers.

## Recommended Architecture

Create separate page and partial boundaries inside `action/dashboard` for the three local pages:

- `DashboardPage` renders open tasks and organisation grids.
- `AccountPage` renders the current account/passkey section.
- `SystemPage` renders the current system card(s) and global admin actions for `mainrole.Admin`.

Refactor the current `DashboardCardsPartial.Widget` into smaller render helpers before moving
content:

- open task grid builder
- organisation grid builder
- account grid builder
- system grid builder

Keep the route/action package small by reusing the existing button/card helpers where possible.
Avoid duplicating card construction across pages.

Add a main-context navigation rail path in `ui/uix/partial.NewNavigationRail` for logged-in
main-context pages. It should render:

- Dashboard
- Account
- System, only for `mainrole.Admin`
- Tenants, only for `mainrole.Admin` and supplied by the existing Tenants integration

The current `NavigationRail` supports destinations already, so no new navigation widget is needed.
If the existing Tenants integration only contributes `wx.MenuItem` through
`PluginRegistry().ExtendMenuItems`, add a minimal plugin hook for main navigation destinations or a
small adapter that lets the tenant-management plugin expose its destination consistently.

For organisation action placement, prefer a small extension to `wx.Grid`, such as `Actions IWidget`
rendered between `Heading` and `Children`. This keeps action placement consistent and avoids
one-off wrapper markup in the dashboard partial. Existing `Footer` can remain for pages that need
actions below content.

## Schema And Migrations

No schema changes are required. This is a routing, rendering, and navigation restructuring task.

## Routes And Handlers

Extend `ui/uix/route/dashboard.go` with explicit main-context routes:

- `GET /dashboard/` for Dashboard.
- `GET /dashboard/account/` for Account.
- `GET /dashboard/system/` for System.

Keep dashboard action command endpoints under the existing dashboard action route prefix unless a
command clearly belongs to admin or auth already. Register the new pages in
`server.registerCoreRoutes`.

The System handler must enforce `mainrole.Admin` server-side. If a non-admin requests it directly,
return a forbidden HTTP error through existing error helpers.

The Tenants route should not be reinvented in core if it already exists in the plugin/host layer.
Instead, the navigation rail should receive a system-admin-only Tenants destination from the same
integration that currently adds it to the dashboard main menu.

## Template And HTMX Flow

Each page should render a full page using the existing `acommon.Page.Render` and `wx.MainLayout`
pattern. Navigation entries should use `HxGet` links so rail navigation swaps consistently with the
rest of the app.

Partials should refresh only the content they own:

- Dashboard partial refreshes on account/task/organisation events that affect open tasks or
  organisation cards.
- Account partial refreshes on passkey, password, and account update events.
- System partial refreshes on app lock/unlock, passphrase, app initialization, and upload limit
  events.

Review the current broad event list in `DashboardCardsPartial` and move each trigger to the page
that needs it. Avoid keeping one partial that refreshes all three pages for unrelated events.

For organisation actions above cards, render the action row between the organisation heading and
the card grid. The action row should still wrap on narrow screens and use existing elevated/tonal
button styling.

## Lit Boundaries

Lit is unnecessary. The existing server-rendered widgets and HTMX interactions cover page
navigation, commands, dialogs, confirmations, and partial refreshes.

## Design System And Components

Use the existing Material Design 3-flavored widgets:

- `wx.MainLayout` for page shell.
- `wx.NavigationRail` and `wx.NavigationDestination` for desktop rail and mobile bottom rail.
- `wx.AppBar` for page titles and mobile menu access.
- `wx.Grid`, `wx.Card`, `wx.Row`, `wx.Button`, and `wx.Link` for content sections and actions.

Recommended icons:

- Dashboard: `dashboard`
- Account: `account_circle` or existing account/person icon style
- System: `admin_panel_settings` or `settings`
- Tenants: `domain` or `apartment`

Keep the main menu available for mobile and preserve plugin-provided menu entries. If rail and menu
entries can diverge, document the intended difference and keep the Tenants destination available in
both places for system admins.

## CLI For Agents

No CLI is needed. This is a UI navigation and page composition change with no new repeatable
operator workflow.

## Auth And Permissions

- Dashboard requires the existing logged-in main context.
- Account requires the current logged-in account and must only render the current account's data.
- System requires `ctx.MainCtx().Account.Role == mainrole.Admin`.
- Tenants navigation requires system admin role and the existing tenant-management integration.
- Organisation action buttons must keep the current owner and initialization checks in
  `tenant.NewTenant(...).IsOwner(...)` and `tenantm.IsInitialized()`.
- Do not rely on hidden navigation for security. Handlers and commands must keep server-side
  permission checks.

## Failure Modes

- If passkey policy loading fails, keep the current safe fallback and log the error.
- If organisation space loading fails, return the error from Dashboard rather than rendering a
  partial page.
- If a non-admin requests System directly, return forbidden with a friendly HTTP error.
- If the Tenants integration is unavailable, omit the Tenants rail destination rather than
  rendering a broken link.
- If the account must complete tenant passkey enrollment, keep the setup-required behavior visible
  from Dashboard and ensure Account does not expose bypass actions.

## Tests

Add or update integration tests around rendered pages and fragments:

- Dashboard renders open tasks and organisation sections but no account/passkey section.
- Dashboard renders organisation action buttons above the organisation cards.
- Account renders passkey registration/recovery/credential content.
- Account reacts to password/passkey/account update events through the right partial trigger.
- System is hidden from non-admin navigation and direct requests are forbidden.
- System renders app status and upload-limit actions for system admins.
- Main navigation rail includes Dashboard and Account for logged-in users.
- Main navigation rail includes System and Tenants for system admins when the Tenants integration
  contributes a destination.
- Existing passkey enforcement command tests continue to pass after the button moves.

Run targeted tests first, then the full suite if feasible:

- `go test ./server -run 'TestDashboard|TestAccount|TestSystem|TestNavigation'`
- `go test ./...`
- `go build ./...`

## Implementation Phases

1. Extract dashboard section builders from `DashboardCardsPartial` without changing behavior.
2. Add Account and System routes, pages, partials, and route registration.
3. Move account/passkey content to Account and system-admin content to System.
4. Add main-context navigation rail destinations and active states.
5. Add a plugin/host navigation destination hook for Tenants if the current Tenants page is only
   exposed through main-menu extension.
6. Move organisation action rows above organisation cards with a small `wx.Grid` extension or a
   local section wrapper.
7. Add translations for new visible strings in `messages.gotext.json` with `fuzzy: true`.
8. Add or update integration tests and run verification.

## Rejected Alternatives

- Creating a new dashboard domain model: rejected because this is page composition over existing
  account, tenant, and system models.
- Duplicating the existing Tenants page in core: rejected because the page already exists and is
  integrated through the host/plugin layer.
- Replacing the main menu with the rail: rejected because the current mobile and plugin menu flow
  should remain available.
- Adding client-side state with Lit: rejected because HTMX and server-rendered pages already fit.

## Implementer Handoff

Preserve these terms: `Dashboard`, `Account`, `System`, `Tenants`, `Organisation`, and
`Organisation actions`.

Likely files to touch:

- `action/dashboard/dashboard_page.go`
- `action/dashboard/dashboard_cards_partial.go`
- `action/dashboard/actions.go`
- new `action/dashboard/account_page.go`
- new `action/dashboard/account_cards_partial.go` or equivalent
- new `action/dashboard/system_page.go`
- new `action/dashboard/system_cards_partial.go` or equivalent
- `ui/uix/route/dashboard.go`
- `ui/uix/partial/navigation_rail.go`
- `ui/uix/partial/main_menu.go`, only if shared destination helpers are useful
- `ui/widget/grid.go` and `ui/widget/grid.gohtml`, if adding an above-cards action slot
- `pluginx/*`, if a navigation rail extension hook is needed for the existing Tenants page
- `server/server.go`
- `server/action_integration_test.go`
- `i18n/**/messages.gotext.json` for new visible strings

Ordered implementation steps:

1. Extract current dashboard card-building logic into helpers that return open task grids,
   organisation grids, account grids, and system grids.
2. Keep Dashboard rendering open tasks plus organisation grids only.
3. Add Account page and move account/passkey grids there.
4. Add System page and move admin-only system grids there with direct handler permission checks.
5. Add navigation rail destinations for Dashboard and Account in main context.
6. Add admin-only System destination and active state.
7. Add admin-only Tenants destination from the existing Tenants integration; add a plugin hook if
   the core rail cannot currently see that destination.
8. Move organisation action rows above each organisation grid's cards.
9. Split HTMX refresh triggers so each partial refreshes only for relevant events.
10. Update tests and translations.

Required verification:

- Run targeted server integration tests covering dashboard, account, system, and navigation.
- Run existing passkey enforcement tests because the button location changes.
- Run `go test ./...` and `go build ./...` before handoff.

Open dependency:

- The exact Tenants destination source is in the existing tenant-management integration, not in the
  local dashboard files inspected here. The implementation should locate that plugin/menu extension
  and expose the same destination to the rail for system admins.
