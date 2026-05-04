# Condensed File Table Checklist

## Preferences

- [ ] Add typed `FileListPreferences` model with defaults.
- [ ] Add `FileListViewMode` values for `list` and `table`.
- [ ] Add built-in column identifiers for `name`, `document_type`, `date`, `size`, and `tags`.
- [ ] Add per-Space metadata column selections keyed by Space public ID.
- [ ] Add safe JSON marshal/unmarshal helpers.
- [ ] Add fallback behavior for invalid JSON and unknown values.
- [ ] Add an account persistence field in `db/entmain/schema/account.go`.
- [ ] Run `go generate ./...` after the schema change.

## Preference Command

- [ ] Add a command to update view mode and column selections.
- [ ] Validate allowed view modes.
- [ ] Validate built-in column names.
- [ ] Validate Space-scoped property IDs against the current Space.
- [ ] Save preferences on the current account.
- [ ] Update `ctx.MainCtx().Account` after saving.
- [ ] Trigger the needed HTMX refresh without adding URL parameters.
- [ ] Register the command in the relevant `Actions` structs.

## App Bar UI

- [ ] Add a view-mode `IconButton` near the sort button in Files.
- [ ] Add a view-mode `IconButton` near the sort button in Inbox.
- [ ] Add menu items for `List` and `Table`.
- [ ] Add configurable built-in column menu items.
- [ ] Add configurable `Tags` column menu item.
- [ ] Add current Space metadata field column menu items.
- [ ] Extend `wx.MenuItem` minimally for checkbox-style menu items.
- [ ] Add translations for new visible strings.

## Table Rendering

- [ ] Add table rendering path for Files.
- [ ] Add table rendering path for Inbox.
- [ ] Preserve existing list rendering path.
- [ ] Preserve file row click behavior and `#details` target.
- [ ] Preserve directory row navigation behavior.
- [ ] Preserve selected row styling.
- [ ] Preserve row context menus.
- [ ] Preserve load-more behavior in Files.
- [ ] Render current list on mobile when table mode is selected.

## Data Loading

- [ ] Build row view models from existing query results.
- [ ] Batch-load document type data when visible.
- [ ] Batch-load current file size data when visible.
- [ ] Batch-load tags when visible.
- [ ] Batch-load selected metadata assignments when visible.
- [ ] Ignore deleted or inaccessible metadata columns.
- [ ] Avoid row-by-row queries for optional table columns.

## Styling

- [ ] Use dense Material 3 table styling consistent with existing widgets.
- [ ] Keep row height compact but touch-safe where required.
- [ ] Truncate long filenames and metadata values safely.
- [ ] Make selected and hover states clear.
- [ ] Keep desktop table readable with many configured columns.

## Tests

- [ ] Test preference defaults.
- [ ] Test invalid preference fallback.
- [ ] Test view mode toggling.
- [ ] Test built-in column toggling.
- [ ] Test tags column toggling.
- [ ] Test per-Space metadata column toggling.
- [ ] Test metadata columns do not leak across Spaces.
- [ ] Test Files list mode still renders list items.
- [ ] Test Files table mode renders headers and rows.
- [ ] Test Inbox list mode still renders list items.
- [ ] Test Inbox table mode renders headers and rows.

## Verification

- [ ] Run `go generate ./...`.
- [ ] Run targeted package tests for the preference model.
- [ ] Run targeted action/server tests touched by the implementation.
- [ ] Run `go test ./...`.
- [ ] Run `go build ./...`.
