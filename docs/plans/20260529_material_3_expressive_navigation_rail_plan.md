# Material 3 Expressive Navigation Rail Plan

## Outcome

Replace the current baseline-style navigation rail and popup main menu split with a
Material 3 Expressive navigation rail that supports collapsed and expanded states.
The collapsed rail remains the default on compact/mobile screens, and the mobile
app-bar menu button opens the expanded navigation rail instead of opening the
current popup `NewMainMenu`.

On larger screens, destinations that currently live only in the main menu move
into the expanded navigation rail. The Space metadata entry is reworked so
`Document types`, `Tags`, and `Fields` are directly reachable instead of being
hidden behind a single `Metadata` rail item.

## Material Guidance Read

Sources:

- https://m3.material.io/components/navigation-rail/specs
- https://m3.material.io/components/navigation-rail/guidelines
- Local Material reference: `/home/marco/.config/opencode/skills/material-design-3-expressive/reference/components/NavigationRail.md`

Key guidance applied to this plan:

- The baseline navigation rail is no longer recommended; use the M3 Expressive
  collapsed and expanded navigation rail variants instead.
- The collapsed rail should stay visible and contain 3-7 primary destinations.
- The expanded rail can reveal secondary destinations not visible when collapsed.
- The expanded rail should open from a menu icon, and the icon should change to
  communicate collapse when expanded.
- Standard expanded rails sit beside body content; modal expanded rails overlap
  body content and use a scrim where space is limited.
- The rail belongs on the leading edge of adaptive layouts and should not be
  placed inside body content panes.
- The rail should be the only visible primary navigation surface; do not show a
  competing navigation bar/drawer/menu at the same time.
- Active state is singular for the current page. The active indicator hugs the
  item contents in expanded rail, while the pointer target still spans the full
  rail width.
- FABs belong near the top of the rail. In expanded state, FABs should become
  extended FABs when possible.
- Labels should be short and meaningful, with no truncation. Avoid shrinking type
  to make labels fit.

Compact note: M3 normally recommends a navigation bar for compact windows. This
project intentionally keeps the existing collapsed compact rail as the default
because the requested behavior depends on it, and uses the expanded rail as the
mobile modal menu replacement.

## Current SimpleDMS Context

- `ui/uix/partial/navigation_rail.go` builds the current rail as a flat slice of
  `wx.NavigationDestination`.
- `ui/widget/navigation_rail.gohtml` renders a bottom-style compact rail and a
  baseline side rail on `md+` screens.
- `ui/widget/navigation_destination.gohtml` only supports leaf destinations and
  cannot model expanded-only sections, groups, subheaders, badges, disabled
  destinations, or footer destinations.
- `ui/uix/partial/main_menu.go` contains destinations that are not all present in
  the rail: Dashboard, Account, System, tenant/Space entries, Space metadata
  pages, Space users, Trash, Sign out, About, and plugin menu items.
- Many app bars set `LeadingAltMobile: partial.NewMainMenu(...)`, so the mobile
  menu entry point currently bypasses the rail.
- `AppBar.GetSearch()` copies `LeadingAltMobile` into search app bars, so the
  replacement must also cover pages with search mode.
- The current rail injects `MenuBtn: NewMainMenu(ctx, infra)`, so `NewMainMenu`
  appears both as rail menu button on `md+` and as app-bar mobile menu.
- `tailwind.config.js` defines Material window classes: compact below `md=600px`,
  medium from `600px`, expanded from `840px`, large from `1200px`, and
  extra-large from `1600px`.
- Side sheets are currently dialog-based and use fixed positioning on `lg+` in
  `ui/widget/dialog.gohtml`. The hard-coded `2xl:left-[1390px]` positioning must
  be reviewed because an expanded standard rail changes the content start edge
  and available viewport width.
- SimpleDMS currently has no dedicated navigation-rail runtime asset. Base assets
  are loaded through `ui/widget/base.gohtml` from `ui/uix/web/assets`.

## Erdikon Reference

The Erdikon implementation is a useful implementation template, especially:

- `core/ui/widget/navigation_rail.go`
- `core/ui/widget/navigation_rail_item.go`
- `core/ui/widget/navigation_rail.gohtml`
- `core/ui/widget/navigation_rail_item.gohtml`
- `core/ui/widget/navigation_rail_toggle.gohtml`
- `core/ui/web/assets/navigation_rail.js`
- `core/ui/uix/partial/navigation_rail.go`
- `docs/invariants/material-3-expressive-navigation-rail-invariants.md`

Important Erdikon behaviors to adapt, not blindly copy:

- Use a dedicated `NavigationRailItem` model and keep a conversion path from
  existing `NavigationDestination` only for compatibility during migration.
- Render separate collapsed and expanded item templates.
- Use a rail toggle widget for app bars instead of a popup main menu widget.
- Use JavaScript only for local UI state: expanded/collapsed state, group
  disclosure, ARIA synchronization, scrim/Escape dismissal, and HTMX
  reinitialization.
- Keep compact expansion non-persistent while allowing `md+` expansion and group
  disclosure to persist in `localStorage`.
- Use `simpledms` storage keys and SimpleDMS breakpoints. Erdikon uses a
  `768px` breakpoint; SimpleDMS must use the existing `600px` `md` breakpoint.

## Goals

- Use the M3 Expressive collapsed and expanded navigation rail variants.
- Keep the collapsed compact rail as the default mobile navigation surface.
- Make the mobile app-bar menu button open the expanded rail as a modal overlay.
- Remove the popup main menu from standard mobile navigation paths.
- Move current main-menu destinations into the expanded rail on larger screens.
- Preserve all current visibility and policy rules, including required tenant
  passkey enrollment restrictions.
- Make `Document types`, `Tags`, and `Fields` direct rail destinations.
- Keep rail-triggered server changes on existing HTMX/action handlers.
- Keep side sheets usable when the standard expanded rail is open on larger
  screens.
- Preserve no-JavaScript usefulness: the server-rendered rail starts collapsed and
  usable before runtime code runs.

## Non-Goals

- Do not redesign SimpleDMS information architecture beyond removing popup-menu
  duplication and exposing metadata destinations directly.
- Do not introduce a separate navigation drawer component.
- Do not replace backend-driven HTMX navigation with client-side routing.
- Do not persist mobile expanded state server-side or in localStorage.
- Do not implement browser-history or Android predictive-back integration in the
  first implementation pass.
- Do not rewrite unrelated app bars, dialogs, side sheets, or layout widgets
  beyond changes required for the expanded rail.

## Destination Model

Add a dedicated `wx.NavigationRailItem` model inspired by Erdikon. It should
support at least:

- Stable `Key` and `Value` fields for active matching and persisted group state.
- `Href` and `HTMXAttrs` for normal links and HTMX actions.
- `Label`, `Icon`, and active state.
- `Children` for expanded-only groups.
- `IsSubheader` for expanded section labels.
- `IsDisabled` so unavailable destinations can be visible but non-interactive.
- Optional badge fields for future required-action indicators.
- `IsCollapsible` and `IsExpandedByDefault` for expanded rail groups.

Keep `NavigationDestination` and `NavigationBar` compatibility until the rail
migration is complete. The new rail can include a helper like
`NewNavigationRailItemFromDestination` so existing plugin hooks and tests can be
migrated incrementally.

## Rail Composition

Use these item areas:

- Collapsed items: the 3-7 highest-priority leaf destinations for the current
  context.
- Expanded primary items: all top-level destinations for the current context.
- Expanded groups: lower-priority or secondary destinations such as Space
  metadata, tenant/Space switching, and account actions.
- Expanded footer items: low-frequency destinations such as About and Sign out
  when they are not better represented in an Account group.
- FAB/action slot: page-provided FABs in collapsed mode and extended FABs in
  expanded mode.

Recommended Space context composition:

- `md+` collapsed rail: `Files`, `Inbox`, `Document types`, `Tags`, `Fields`,
  `Users`, and `Trash`, subject to authorization and available routes. This
  stays within the M3 3-7 collapsed item guidance and makes metadata subitems
  direct.
- Compact/mobile collapsed rail: keep the previous behavior with `Files`,
  `Inbox`, and `Metadata` on normal Space pages, then `Files`, `Document types`,
  `Tags`, and `Fields` on metadata pages.
- Expanded rail: the same Space destinations, plus Dashboard, Account, System for
  admins, tenant/Space switching, Sign out, About when commercial licensing is
  disabled, and plugin-provided destinations.

Recommended main/dashboard context composition:

- Collapsed rail: `Dashboard`, `Account`, `System` for admins, and the most
  important tenant/Space management destination if available.
- Expanded rail: all current main menu destinations, including all visible tenant
  and Space entries, Sign out, About, and plugin-provided destinations.

Required tenant passkey enrollment composition:

- Match current `NewMainMenu` behavior. The replacement rail should expose only
  the allowed restricted destinations, currently Dashboard and Sign out, plus any
  required Account destination if it is part of completing the required action.

## Metadata Destinations

Remove the current metadata-mode rail switch in which normal Space pages show a
single `Metadata` item and metadata pages replace the rail with `Files`,
`Document types`, `Tags`, and `Fields`.

Instead:

- Always model `Document types`, `Tags`, and `Fields` as direct
  `NavigationRailItem` leaves in Space context.
- Mark the exact leaf active for each metadata page.
- Do not mark a synthetic `Metadata` parent active when a child is active.
- In expanded state, optionally group the three leaves under a `Metadata`
  subheader or collapsible group, but each child remains directly actionable.
- In `md+` collapsed state, render direct leaf destinations rather than a
  non-actionable group.
- In compact/mobile collapsed state, keep the legacy `Metadata` entry that opens
  `Document types`, then switch to the metadata page rail.

## Mobile Behavior

Compact/mobile means below SimpleDMS' current `md=600px` breakpoint.

- Server-render collapsed rail by default.
- App-bar mobile leading action renders a rail toggle, not `NewMainMenu`.
- Tapping the app-bar toggle opens the expanded rail as a modal overlay from the
  leading edge.
- The modal rail uses a scrim and does not push or resize page content.
- The compact bottom rail remains the closed state when the expanded overlay is
  not open.
- The compact bottom rail shows at most four destinations; additional
  destinations remain available in the expanded modal rail.
- In Space context, the compact bottom rail uses the previous metadata-mode
  behavior rather than simply taking the first four full rail destinations.
- Mobile expanded state is never restored from `localStorage` after reload, HTMX
  swap, or navigation.
- Close the mobile expanded rail on toggle, scrim click, Escape, and destination
  activation.
- Keep normal anchors and HTMX attributes intact so navigation and commands keep
  the current request flow.
- Keep focus visible and synchronize `aria-expanded`, `aria-controls`, and
  hidden/modal semantics.

## Medium And Larger Behavior

Medium and larger means `md+`, currently `600px` and wider.

- The collapsed side rail width uses the current SimpleDMS `80px` rail width.
- The expanded standard rail should occupy layout width beside content. Use a
  stable width around `360px`, matching the Erdikon implementation and M3 drawer
  replacement intent.
- The expanded rail is non-modal on `md+`; do not show a scrim.
- Persist `md+` expanded/collapsed state in browser-local `localStorage` using a
  SimpleDMS-specific key such as `simpledms.navigationRail.expanded`.
- Persist expanded group disclosure state in browser-local `localStorage` using
  stable item keys.
- Keep the server-rendered default collapsed, then let runtime restore the
  browser preference after scripts load.
- Ensure window resize across `md` recalculates mode and does not leave compact
  modal state active on larger screens.

## Side Sheets And Layout Width

Expanded standard rail changes the inline start edge of the content area. Side
sheets rendered outside the main flex layout or fixed to the viewport must avoid
overlapping the rail or anchoring to stale hard-coded coordinates.

Implementation should:

- Introduce a CSS custom property for the current rail width, for example
  `--simpledms-navigation-rail-width`.
- Set it to `0px` on compact screens, `80px` for collapsed `md+`, and the
  expanded rail width for expanded `md+`.
- Use that property when positioning large-screen side sheets if they are fixed
  to the viewport.
- Review `ui/widget/dialog.gohtml` side-sheet classes, especially `lg:fixed`,
  `2xl:left-[1390px]`, and `2xl:right-auto`.
- Preserve current modal/fullscreen side-sheet behavior below `lg`.
- Manually verify browse details, file details, tag filters, property filters,
  and document-type filters with collapsed and expanded rails.

## Plugin And Extension Hooks

Current extension points split menu and rail destinations:

- `ExtendMenuItems` appends popup menu items.
- `ExtendNavigationDestinations` appends current rail destinations.

The expanded rail should become the primary extension target. Prefer adding a new
`ExtendNavigationRailItemsHook` returning `[]*wx.NavigationRailItem`, then bridge
existing `ExtendNavigationDestinations` into rail items during migration.

Do not use `ExtendMenuItems` as the long-term source for rail destinations. If
menu-only plugin behavior still exists, migrate it to rail item hooks or document
the exception.

## Implementation Phases

1. Add `NavigationRailItem`, `NavigationRailToggle`, collapsed/expanded rail
   template support, and a compatibility conversion from `NavigationDestination`.
2. Add a small `navigation_rail.js` runtime asset and include it from
   `ui/widget/base.gohtml`.
3. Rebuild `partial.NewNavigationRail` around item helpers and destination groups,
   preserving current policy gates and adding all main-menu-only destinations.
4. Rework metadata navigation so `Document types`, `Tags`, and `Fields` are direct
   destinations in collapsed and expanded Space rails.
5. Replace app-bar `LeadingAltMobile: partial.NewMainMenu(...)` usages with a rail
   toggle, including search app bars through `AppBar.GetSearch()`.
6. Remove `MenuBtn: NewMainMenu(ctx, infra)` from the rail. Delete or deprecate
   `NewMainMenu` only after code search proves no standard navigation path uses it.
7. Add side-sheet layout adjustments for expanded `md+` rail width.
8. Add rendering tests where practical and run browser checks at compact, medium,
   expanded, large, and extra-large widths.

## Acceptance Criteria

- On compact/mobile, tapping the app-bar menu button opens an expanded navigation
  rail overlay instead of a popup menu.
- On compact/mobile, the collapsed rail remains the default closed state.
- On compact/mobile, the expanded rail closes via toggle, scrim click, Escape, and
  destination activation.
- On compact/mobile, reload and HTMX navigation do not restore the expanded rail
  as open.
- On `md+`, the rail can expand into a standard non-modal rail beside content.
- On `md+`, all destinations currently available in the main menu are available
  in the expanded rail when the same visibility rules allow them.
- `Document types`, `Tags`, and `Fields` are directly actionable rail
  destinations.
- The selected page has one active rail destination and `aria-current="page"`.
- Disabled destinations do not expose activation attributes.
- Side sheets remain correctly positioned with collapsed and expanded rails.
- Rail-triggered navigation and commands still use existing HTMX/action routes.

## Verification

- Code search: no standard app bar still uses `partial.NewMainMenu(...)` for
  `LeadingAltMobile`.
- Code search: no `NewMainMenu` popup is reachable from the rail menu button.
- Render checks for visitor, main dashboard, admin dashboard, Space browse,
  Inbox, metadata pages, Space users, Trash, and About.
- Render checks for plugin-provided navigation destinations.
- Browser compact width check for overlay open/close, scrim click, Escape,
  destination activation, scroll, and no persisted open state.
- Browser `md+` check for collapsed rail, expanded rail, persisted expanded state,
  persisted group disclosure, and resize across `md`.
- Browser large/extra-large check for side-sheet placement with collapsed and
  expanded rails.
- Keyboard check for app-bar toggle, rail-local toggle, focus indicators, group
  toggles, Escape dismissal, and disabled items.
- `gofmt -w` on changed Go files.
- `go test ./...`.
- `go build ./...`.
- Rebuild frontend assets if generated Tailwind or asset bundles change.

## Risks And Mitigations

- Risk: Removing `NewMainMenu` loses a destination.
  Mitigation: Implement menu-to-rail parity first and add rendering checks for all
  main menu contexts.
- Risk: Required passkey enrollment users can navigate to screens previously
  hidden by `NewMainMenu`.
  Mitigation: Move the passkey gate into rail item construction before replacing
  mobile app-bar entries.
- Risk: Expanded desktop state opens as a modal after resizing from mobile.
  Mitigation: Branch runtime behavior by the `md=600px` media query and reset
  compact modal state on breakpoint changes.
- Risk: Expanded rail and side sheets compete for horizontal space.
  Mitigation: Track rail width through a CSS custom property and test side sheets
  at `lg`, `xl`, and `2xl`.
- Risk: Long labels such as `Document types` truncate in collapsed rail.
  Mitigation: Use short labels where translations allow it, otherwise allow
  controlled two-line wrapping and verify no ellipsis is used.
- Risk: Plugin items still target `ExtendMenuItems` only.
  Mitigation: Add a rail item hook and migrate plugin/navigation tests to the new
  hook, with a temporary compatibility bridge if needed.
