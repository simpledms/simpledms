# Condensed File Table Plan

## Outcome

Add an optional condensed table experience for the Files view and Inbox while keeping the
current `ListItem` list as the default/mobile-safe presentation. Users can switch between
list and table from the top app bar near the sorting button. Their choice is saved silently as
an account-wide preference.

## Confirmed Requirements

- The feature applies to both Files and Inbox from the start.
- The switch appears in the top app bar near the sort button.
- View mode is a per-user preference.
- View mode is account-wide across tenants and spaces.
- The URL must stay clean; toggling the view saves the preference silently.
- Table mode is available on mobile/narrow screens when the user chooses it.
- Row click, selection, context menu, and actions should behave like the current list rows.
- Table columns are configurable.
- Default table columns are `Name`, `Type`, `Date`, and `Size`.
- Available columns include built-in columns, `Tags`, and Space-specific metadata fields.
- Metadata field and tag column selections are saved per Space inside the account-level preference.

## Current Code Context

- Files list rendering lives in `action/browse/list_dir_partial.go`.
- Files row rendering lives in `action/browse/file_list_item_partial.go`.
- Inbox list rendering lives in `action/inbox/files_list_partial.go`.
- Inbox row rendering lives in `action/inbox/file_list_item_partial.go`.
- Sort menus are implemented in:
  - `action/browse/sort_list_context_menu_widget.go`
  - `action/inbox/sort_list_context_menu_widget.go`
- Existing rows use `wx.ListItem`, `wx.List`, `wx.Menu`, and `wx.MenuItem`.
- Existing state for search, filters, and sort is URL-derived via `autil.StateX`.
- There is no existing durable user preference model.
- Account data is stored in the main database through `db/entmain/schema/account.go`.
- Metadata fields are tenant/Space records through `Property` and `FilePropertyAssignment`.
- Tags are tenant/Space records through `Tag` and tag assignments.

## Domain Model

This is a supporting usability feature, not a core document-management invariant. Use a small
typed preference value object rather than introducing a complex domain model.

Recommended ubiquitous language:

- `FileListViewMode`: `list` or `table`.
- `FileListColumn`: a built-in table column such as `name`, `document_type`, `date`, `size`, or `tags`.
- `SpaceFileListColumns`: Space-scoped metadata and tag column selections.
- `FileListPreferences`: account-level preferences for Files and Inbox display.

## Persistence

Persist preferences on `entmain.Account` as a JSON field or JSON string field, for example
`file_list_preferences_json`, defaulting to `{}`.

Keep JSON decoding and validation in a typed model package, likely:

- `model/main/account/file_list_preferences.go`

The preference shape should support:

```json
{
  "view_mode": "table",
  "built_in_columns": ["name", "document_type", "date", "size"],
  "space_columns": {
    "space-public-id": {
      "property_ids": [123, 456],
      "show_tags": true
    }
  }
}
```

Use internal numeric IDs for property lookup once inside the server, but store Space keys by public ID
because preferences are external/user-facing account data and should not leak tenant database internals
across contexts.

Invalid, unknown, deleted, or inaccessible columns should be ignored at render time rather than causing
the listing to fail.

## Actions And Request Flow

Add one small command for saving display preferences. It can live in a shared package if that avoids
duplication, or in Browse/Inbox if that matches registration patterns better.

The command should:

- Require a logged-in account context.
- Accept a small form payload for one preference change at a time.
- Validate known view modes and column identifiers.
- Update the current account row in the main database.
- Update `ctx.MainCtx().Account` in memory after saving so the follow-up HTMX render uses the new value.
- Return no standalone UI unless a snackbar is useful.
- Trigger the relevant listing/app-bar refresh through existing HTMX follow-up query headers or `HX-Trigger`.

Do not add query parameters for view mode or columns.

## Rendering Approach

Keep list rendering intact and add table rendering as an alternate presentation over the same query results.

For Files:

- Continue using `ListDirFileQueryService.Query` for filtering, sorting, recursion, pagination, and path context.
- Convert query results into a small row view model that can render either list rows or table rows.
- Preserve existing directory-first behavior and `Load more` behavior.

For Inbox:

- Continue using `FilesListPartial.filesQuery`.
- Convert query results into the same or equivalent row view model.

For mobile:

- Render table mode when selected, even on narrow screens.
- Rely on horizontal scrolling when configured columns do not fit.

## Table Components

Prefer adding small reusable widgets under `ui/widget` only if they stay generic and simple:

- `Table`
- `TableHeader`
- `TableRow`
- `TableCell`

If generic widgets would introduce too much abstraction, create file-list-specific table widgets in the
Browse/Inbox action packages first.

Table rows must support:

- Same HTMX attributes as current list rows.
- Same selected-state styling as current list rows.
- Same context menu script behavior as `ListItem` rows, or a small shared helper for context menus.
- Dense Material 3 styling, with clear hover/state layer and readable truncated cells.

## App Bar Controls

Add a view-mode `IconButton` next to the sort `IconButton` in both app bars.

The view menu should include:

- `List` radio item.
- `Table` radio item.
- Divider.
- Checkbox-style built-in column items.
- Checkbox-style `Tags` item.
- Checkbox-style metadata field items for the current Space.

`wx.MenuItem` currently supports radio items but not checkbox items. Extend it minimally with fields like:

- `CheckboxName`
- `CheckboxValue`
- `IsChecked`

Keep the template change small and compatible with existing menu items.

## Column Data Loading

Avoid N+1 queries in table mode.

Batch-load only the visible optional columns:

- Sizes: load current/latest file version and stored file data only when `Size` is visible.
- Tags: load tags for visible file IDs only when `Tags` is visible.
- Metadata fields: load assignments for visible file IDs and selected property IDs only.
- Document type: use existing `WithDocumentType` or a batch load when `Type` is visible.

Directories should render empty cells for file-only data such as size and document type unless a clear
directory value already exists.

## Auth And Permissions

Preference changes require the current logged-in account. Listing data must continue to use the existing
tenant and Space privacy rules.

Do not let saved metadata/property IDs bypass Space access. Always resolve selected columns against the
current Space before rendering.

## Failure Modes

- Invalid preference JSON falls back to default preferences and should be logged.
- Unknown view mode falls back to list.
- No selected columns falls back to default table columns.
- Deleted or inaccessible metadata fields are ignored.
- Table mode on mobile may require horizontal scrolling.
- Preference save failures should surface through the existing HTTP error/snackbar flow.

## Tests

Add focused tests where the existing harness supports it:

- Unit tests for preference parsing, defaults, and invalid values.
- Unit tests for toggling built-in, tag, and per-Space metadata columns.
- Handler or rendering tests that verify list mode still renders existing list items.
- Handler or rendering tests that verify table mode renders table headers and rows.
- Tests that metadata columns are ignored when they do not belong to the current Space.

Run verification:

- `go generate ./...`
- `go test ./model/main/account`
- `go test ./action/inbox ./server`
- `go test ./...`
- `go build ./...`

Adjust package-specific commands if tests are added in different packages.

## Implementation Phases

1. Add typed preferences and persistence on `Account`.
2. Add preference update command and app bar menu controls.
3. Add table rendering for built-in columns in Files and Inbox.
4. Add configurable `Tags` and metadata columns with batched data loading.
5. Add responsive horizontal scrolling and polish dense table styling.
6. Add tests and run generation/build verification.

## Rejected Alternatives

- Browser-only `localStorage`: rejected because the backend should remain the source of truth.
- URL-only state: rejected because the requirement is a silent saved user preference.
- SPA-style data grid: rejected because server-rendered HTMX fits the existing architecture.
- Fully generic table/column engine first: rejected as too much abstraction for the first version.
