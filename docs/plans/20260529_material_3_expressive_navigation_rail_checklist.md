# Material 3 Expressive Navigation Rail Checklist

## Material And Current-State Audit

- [x] Read M3 navigation rail specs.
- [x] Read M3 navigation rail guidelines.
- [x] Review local Material navigation rail reference.
- [x] Inspect current SimpleDMS rail builder and templates.
- [x] Inspect current SimpleDMS main menu destinations and gates.
- [x] Inspect app-bar `LeadingAltMobile` usages.
- [x] Inspect side-sheet positioning constraints.
- [x] Inspect Erdikon navigation rail implementation and docs for reference.

## Widget Model

- [x] Add `wx.NavigationRailItem` with key, value, label, icon, navigation,
  active, children, subheader, disabled, badge, and collapsible group fields.
- [x] Add `wx.NavigationRailToggle` for app-bar and rail-local toggles.
- [x] Add helper methods for active matching, stable DOM IDs, href derivation,
  disabled item behavior, and ARIA labels.
- [x] Keep a compatibility helper from `NavigationDestination` to
  `NavigationRailItem`.
- [x] Keep existing `NavigationBar` and `NavigationDestination` behavior intact
  unless a separate change intentionally migrates them.

## Rail Templates

- [x] Update `ui/widget/navigation_rail.gohtml` to render collapsed and expanded
  regions separately.
- [x] Render a compact modal scrim only when the rail is expanded below `md`.
- [x] Render the collapsed compact rail as the default closed mobile state.
- [x] Render the `md+` collapsed side rail at the SimpleDMS `80px` width.
- [x] Render the `md+` expanded rail as a standard non-modal rail beside content.
- [x] Render collapsed items with active indicator, icon, label, and optional
  small badge.
- [x] Render expanded leaves with hug-content active indicator and full-width
  target area.
- [x] Render expanded subheaders and collapsible groups.
- [x] Render disabled items without `href`, HTMX activation, or keyboard button
  behavior.
- [x] Render page-provided FABs in collapsed state and extended FABs in expanded
  state when possible.
- [x] Render compact collapsed rail items separately from `md+` collapsed items so
  mobile stays within the four-destination limit.
- [x] Render extended FAB labels from the FAB tooltip or label content.
- [x] Support primary and secondary FAB styling in rail action stacks.
- [x] Keep the app-bar rail toggle icon aligned while preserving its vertical
  touch target.
- [x] Keep selected background/state-layer separation so hover/focus does not
  remove the active indicator.

## Rail Runtime

- [x] Add `ui/uix/web/assets/navigation_rail.js`.
- [x] Include the runtime from `ui/widget/base.gohtml`.
- [x] Use SimpleDMS storage keys such as `simpledms.navigationRail.expanded`.
- [x] Use the SimpleDMS `md=600px` breakpoint in runtime media queries.
- [x] Keep mobile expanded state non-persistent.
- [x] Restore `md+` expanded state from localStorage after scripts load.
- [x] Persist `md+` expanded state changes to localStorage.
- [x] Persist expanded group disclosure state by stable item key.
- [x] Synchronize `aria-expanded` and toggle icons for all toggles that target the
  rail.
- [x] Close compact expanded rail through app-bar toggle, rail-local toggle,
  scrim click, Escape, and destination activation.
- [x] Reinitialize rails after HTMX node processing and swaps.
- [x] Recalculate compact versus `md+` behavior after breakpoint changes.

## Destination Parity

- [x] Rebuild `partial.NewNavigationRail` around `NavigationRailItem` helpers.
- [x] Add current Dashboard destination to the expanded rail everywhere
  `NewMainMenu` currently shows it.
- [x] Add current Account destination to the expanded rail where allowed.
- [x] Add current System destination for main admins.
- [x] Add current tenant/Space switching destinations from the main menu.
- [x] Add Space `Document types`, `Tags`, and `Fields` as direct destinations.
- [x] Add Space `Users` destination where the main menu currently shows it.
- [x] Add Space `Trash` destination where the main menu currently shows it.
- [x] Add Sign out as an expanded rail action when logged in.
- [x] Add About when `CommercialLicenseEnabled` is false.
- [x] Preserve visitor/sign-in navigation.
- [x] Preserve required tenant passkey enrollment restrictions from `NewMainMenu`.
- [x] Preserve admin-only System visibility.
- [x] Preserve tenant/Space authorization expectations; target handlers remain the
  security boundary.
- [x] Add or migrate plugin extension hooks for rail items.
- [x] Bridge existing `ExtendNavigationDestinations` plugins during migration if
  needed.

## Metadata Navigation

- [x] Keep compact/mobile Space rail behavior compatible with the previous
  `Files`, `Inbox`, `Metadata` default and metadata-mode rail.
- [x] Remove the synthetic `Metadata` rail item from expanded and `md+` collapsed
  Space rail output.
- [x] Mark `Document types` active only on document type pages.
- [x] Mark `Tags` active only on tag pages.
- [x] Mark `Fields` active only on property/field pages.
- [x] Keep all three metadata destinations directly actionable in collapsed Space
  rails.
- [x] Keep expanded rail metadata destinations as direct leaves instead of an
  active parent `Metadata` destination.

## Mobile Entry Points

- [x] Replace all `LeadingAltMobile: partial.NewMainMenu(...)` usages with the
  rail toggle.
- [x] Verify `AppBar.GetSearch()` carries the rail toggle into search app bars.
- [x] Remove `MenuBtn: NewMainMenu(ctx, infra)` from `NewNavigationRail`.
- [x] Keep `NewMainMenu` only if a documented non-standard use remains.
- [x] Code search confirms no standard mobile app bar opens the popup main menu.

## Side Sheets And Layout

- [x] Add a CSS custom property for current rail width.
- [x] Set rail width property to `0px` on compact screens.
- [x] Set rail width property to `80px` for collapsed `md+` rail.
- [x] Set rail width property to expanded rail width for expanded `md+` rail.
- [x] Review `ui/widget/main_layout.gohtml` and `ui/widget/narrow_layout.gohtml`
  with expanded `md+` rail.
- [x] Adjust `ui/widget/dialog.gohtml` side-sheet positioning to account for
  expanded rail width on large screens.
- [x] Remove or replace hard-coded side-sheet coordinates that break with the
  expanded rail.
- [x] Preserve current fullscreen/modal side-sheet behavior below `lg`.

## Tests

- [x] Add targeted rail model assertions for main dashboard rail items.
- [x] Add targeted rail model assertions for admin-only System rail item.
- [x] Add targeted rail model assertions for Space rail items: Files, Inbox,
  Document types, Tags, Fields, Users, and Trash.
- [x] Add targeted rail model assertions for About visibility when commercial
  licensing is disabled.
- [x] Add targeted rail model assertions for required tenant passkey enrollment
  restrictions.
- [x] Confirm no disabled rail items were introduced, so dedicated disabled-item
  assertions are not required for this change.
- [x] Add targeted assertions for plugin-provided rail items and legacy
  `ExtendNavigationDestinations` bridge items.
- [x] Add code-search coverage that standard app bars use the rail toggle instead
  of `NewMainMenu` where practical.

## Browser Verification

- [x] Compact dashboard: rail opens from app bar and closes via toggle, scrim,
  Escape, and destination activation.
- [x] Compact Space browse: collapsed default shows `Files`, `Inbox`, and
  `Metadata`; metadata pages show `Files`, `Document types`, `Tags`, and
  `Fields`.
- [ ] Compact search app bars: leading mobile button opens the rail, not the old
  popup menu.
- [x] Compact reload/HTMX navigation: expanded rail is not restored open.
- [x] Medium width: collapsed rail renders as a side rail, not a bottom overlay.
- [x] Medium width: expanded rail is non-modal and no scrim is visible.
- [x] Desktop width: expanded state persists after reload.
- [ ] Desktop width: group disclosure persists after reload.
- [x] Resize across `md=600px`: modal and persisted behaviors reset correctly.
- [x] Large/extra-large browse details: side sheets are positioned correctly with
  collapsed and expanded rails.
- [ ] Keyboard: focus indicators, group toggles, `aria-expanded`, Escape, and
  disabled items behave correctly.

## Verification Commands

- [x] Run `gofmt -w` on changed Go files.
- [x] Rebuild Tailwind/assets if generated CSS or asset bundles change.
- [x] Run `go test ./...`.
- [x] Run `go build ./...`.
- [x] Run `git diff --check`.
