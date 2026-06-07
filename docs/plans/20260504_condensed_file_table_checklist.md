# Condensed File Table Checklist

## Preferences

- [x] Add typed `FileListPreferences` model with defaults.
- [x] Add `FileListViewMode` values for `list` and `table`.
- [x] Add built-in column identifiers for `name`, `document_type`, `metadata`, `date`, and `size`.
- [x] Add per-Space metadata column selections keyed by Space public ID.
- [x] Add per-Space tag group column selections keyed by Space public ID.
- [x] Use typed `field.JSON` persistence instead of manual string JSON marshal/unmarshal.
- [x] Add fallback behavior for invalid preference values.
- [x] Add an account persistence field in `db/entmain/schema/account.go`.
- [x] Add SQLite-safe SQL default for existing account rows.
- [x] Regenerate ent main schema after the schema change.
- [ ] Run full `go generate ./...` successfully.

Note: full `go generate ./...` was attempted, but currently fails on an unrelated `enumer` export-format issue in `model/main/common/language`. Targeted ent generation for `db/entmain` succeeded.

## Preference Command

- [x] Add a command to update view mode and column selections.
- [x] Validate allowed view modes.
- [x] Validate built-in column names.
- [x] Validate Space-scoped property IDs against the current Space.
- [x] Validate Space-scoped tag group IDs against the current Space.
- [x] Save preferences on the current account.
- [x] Update `ctx.MainCtx().Account` after saving.
- [x] Trigger the needed HTMX refresh without adding URL parameters.
- [x] Register the command in the relevant `Actions` structs.

## App Bar UI

- [x] Add a view-mode `IconButton` near the sort button in Files.
- [x] Add a view-mode `IconButton` near the sort button in Inbox.
- [x] Add menu items for `List` and `Table`.
- [x] Add configurable built-in column menu items.
- [x] Add configurable `Tags` column menu item.
- [x] Add current Space metadata field column menu items.
- [x] Add current Space tag group column menu items.
- [x] Extend `wx.MenuItem` minimally for checkbox-style menu items.
- [x] Keep sort and view buttons next to each other in search app bars.
- [ ] Add reviewed translations for new visible strings in `messages.gotext.json`.

## Table Rendering

- [x] Add table rendering path for Files.
- [x] Add table rendering path for Inbox.
- [x] Preserve existing list rendering path.
- [x] Preserve file row click behavior and `#details` target.
- [x] Preserve directory row navigation behavior.
- [x] Preserve selected row styling.
- [x] Preserve row context menus.
- [x] Preserve load-more behavior in Files list mode.
- [x] Add table load-more row for Files table mode.
- [x] Render table mode on mobile when the user selected table mode.
- [x] Add built-in `Metadata` column for document-type attribute values.
- [x] Add one table column per selected tag group.

## Data Loading

- [x] Build rows from existing query results.
- [x] Batch-load document type data when visible.
- [x] Batch-load current file size data when visible.
- [x] Batch-load tags when visible.
- [x] Batch-load selected metadata assignments when visible.
- [x] Batch-load selected tag group assignments when visible.
- [x] Batch-load document-type metadata values when visible.
- [x] Ignore deleted or inaccessible metadata columns.
- [x] Ignore deleted, inaccessible, or non-group tag group columns.
- [x] Avoid row-by-row queries for optional table columns.

## Styling

- [x] Use dense table styling consistent with existing widget patterns.
- [x] Keep row height compact.
- [x] Truncate long filenames and metadata values safely.
- [x] Make selected states clear.
- [x] Keep desktop table horizontally scrollable for many configured columns.

## Tests

- [x] Test preference defaults.
- [x] Test invalid preference fallback.
- [ ] Test view mode toggling.
- [x] Test built-in column toggling.
- [x] Test tags column preference storage.
- [x] Test per-Space metadata column preference storage.
- [x] Test per-Space tag group column preference storage.
- [ ] Test metadata columns do not leak across Spaces.
- [ ] Test Files list mode still renders list items.
- [ ] Test Files table mode renders headers and rows.
- [ ] Test Inbox list mode still renders list items.
- [ ] Test Inbox table mode renders headers and rows.

## Verification

- [ ] Run `go generate ./...` successfully.
- [x] Run targeted ent generation for `db/entmain`.
- [x] Run targeted package tests for the preference model.
- [x] Run targeted action/server tests touched by the implementation.
- [x] Run `go test ./...`.
- [x] Run `go build ./...`.
- [x] Run `git diff --check`.

## Done Summary

- Account-wide typed JSON preferences are implemented with `field.JSON` on `Account.file_list_preferences`.
- Files and Inbox have a top-app-bar view menu for list/table mode and column selection.
- Table mode renders on desktop and mobile; narrow screens use horizontal table scrolling.
- Built-in columns, metadata fields, `Tags`, and tag group columns are configurable.
- The `Metadata` built-in column summarizes enabled document-type attributes as `Label: value` pairs.
- Optional table data is batch-loaded for visible rows.
- The core preference model has unit test coverage.
- Full test suite and build pass.
