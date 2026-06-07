# Dashboard, Account, And System Split Checklist

## Preparation

- [ ] Locate the existing Tenants page/menu contribution that currently appears on the dashboard.
- [x] Confirm whether the Tenants contribution can provide a rail destination directly.
- [x] Identify all visible strings added by the split for translation updates.

## Routes And Actions

- [x] Add `Account` route helpers under `ui/uix/route/dashboard.go`.
- [x] Add `System` route helpers under `ui/uix/route/dashboard.go`.
- [x] Register the Account page route in `server.registerCoreRoutes`.
- [x] Register the System page route in `server.registerCoreRoutes`.
- [x] Add `AccountPage` to `action/dashboard.Actions`.
- [x] Add `SystemPage` to `action/dashboard.Actions`.
- [x] Add account/system partial actions if separate partial endpoints are used.

## Dashboard Split

- [x] Extract open task grid rendering from `DashboardCardsPartial.Widget`.
- [x] Extract organisation grid rendering from `DashboardCardsPartial.Widget`.
- [x] Extract account/passkey grid rendering from `DashboardCardsPartial.Widget`.
- [x] Extract system admin grid rendering from `DashboardCardsPartial.Widget`.
- [x] Keep open tasks on Dashboard.
- [x] Keep organisations and related organisation content on Dashboard.
- [x] Remove account/passkey cards from Dashboard.
- [x] Remove system admin cards from Dashboard.

## Account Page

- [x] Render the account heading on Account.
- [x] Render passkey registration/recommendation cards on Account.
- [x] Render recovery-code and passkey credential cards on Account.
- [x] Render change password, edit account, register passkey, and regenerate backup code actions on Account.
- [x] Keep account page data scoped to the current logged-in account.
- [x] Move account-related HTMX refresh triggers to the Account partial/page.

## System Page

- [x] Render System page only for `mainrole.Admin`.
- [x] Return forbidden for direct non-admin System requests.
- [x] Move app status card to System.
- [x] Move global upload-limit action to System.
- [x] Move system-related HTMX refresh triggers to the System partial/page.
- [x] Verify app lock/unlock and passphrase dialogs still work from System.

## Main Navigation Rail

- [x] Add Dashboard destination for logged-in main context.
- [x] Add Account destination for logged-in main context.
- [x] Add System destination for system admins only.
- [x] Add Tenants destination for system admins only.
- [x] Preserve active state for Dashboard, Account, System, and Tenants.
- [x] Preserve current Space-context rail behavior.
- [x] Preserve the mobile/main menu flow.
- [x] Add or reuse a plugin hook so the existing Tenants page can contribute a rail destination.

## Organisation Actions

- [x] Move passkey enforcement action above each organisation card grid.
- [x] Move manage spaces action above each organisation card grid.
- [x] Move manage users action above each organisation card grid.
- [x] Move delete organisation action above each organisation card grid when available.
- [x] Move download backup action above each organisation card grid when available.
- [x] Keep action rows wrapping on narrow screens.
- [x] Keep all existing owner, initialization, SaaS mode, and endpoint checks.
- [x] Update tests that currently look for the passkey enforcement button in dashboard cards.

## Widgets And Templates

- [x] Decide whether to add a `wx.Grid` above-cards action slot or use a local wrapper.
- [x] If extending `wx.Grid`, add the field in `ui/widget/grid.go`.
- [x] If extending `wx.Grid`, render the action slot between heading and card grid in `grid.gohtml`.
- [x] Keep existing `Footer` behavior unchanged for other grids.
- [x] Use existing `wx.Row`, `wx.Button`, and `wx.Link` widgets for action rows.

## Translations

- [x] Add German translations for new strings with informal `du`, Swiss spelling, and `fuzzy: true`.
- [x] Add French translations for new strings with `fuzzy: true`.
- [x] Add Italian translations for new strings with `fuzzy: true`.
- [x] Mark each added translation with `Translated from English by Codex.`.
- [x] Do not edit generated `out.gotext.json` files.

## Tests

- [ ] Test Dashboard renders open tasks when applicable.
- [x] Test Dashboard renders organisation sections.
- [x] Test Dashboard no longer renders account/passkey sections.
- [ ] Test organisation action buttons render above cards.
- [x] Test Account renders passkey registration when no passkeys exist.
- [x] Test Account renders recovery-code and passkey credential cards when passkeys exist.
- [x] Test System is forbidden for non-admin direct requests.
- [ ] Test System navigation is hidden for non-admins.
- [x] Test System renders for system admins.
- [x] Test Tenants navigation is shown for system admins when the existing Tenants integration contributes it.
- [x] Test existing passkey enforcement command behavior still passes.

## Verification

- [x] Run targeted server integration tests for dashboard/account/system/navigation.
- [x] Run existing passkey enforcement tests.
- [x] Run `go test ./...`.
- [x] Run `go build ./...`.
- [x] Run `git diff --check`.

## Done Summary

- [x] Dashboard contains open tasks and organisation content only.
- [x] Account contains personal account and passkey management.
- [x] System contains admin-only system management.
- [x] Navigation rail exposes Dashboard, Account, admin-only System, and admin-only Tenants.
- [x] Organisation action buttons appear above the relevant organisation cards.
- [ ] Tests, build, and translations are complete.
