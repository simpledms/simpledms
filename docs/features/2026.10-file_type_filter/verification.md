# File-type filter verification

Date: 2026-10-20

Implementation: the Filters side sheet ends with a File types tab, using the existing
filter-chip, badge, URL-state and query-island patterns. File types are filename
extensions.

## Checks

- `gofmt` applied to the changed Go files.
- `go generate ./i18n` succeeded; translation catalogs were regenerated.
- `CGO_ENABLED=1 go test -tags "sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_icu" ./server -run '^TestBrowseFileTypeFilters$' -count=1` passed.
- Required suite completed with exit code 1 because S3-dependent server tests could
  not connect to `localhost:7070`; the other reported package results passed:

  ```sh
  CGO_ENABLED=1 go test -tags 'sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_icu' ./... \
    -skip '^(TestConcurrentUploadFileCmdWithSlowS3|TestUploadFileCmdFailsWhenS3Unavailable|TestUploadFilesCmdFailsWhenS3Unavailable)$'
  ```

  The exclusions are the unrelated slow/bad-network tests prohibited by the local
  guide. Full output: `/tmp/opencode/simpledms-go-test.log`.
- Playwright CLI against a freshly built, SQLite-tagged disposable instance passed:
  final-tab placement, multiple selections, deselection, count badges, reload/tab
  persistence, Reset, and tab visibility at 390×844.
- Browser stress checks reproduced a dropped rapid selection, then passed after
  stable chip IDs and an explicit shared request queue were added to the chips and
  panel. Same-turn native checkbox clicks and a second click during list refresh
  both retained the requested selections in the URL, chips, badge and results.

Browser files were seeded as metadata in the disposable instance because S3 was
unavailable; upload/storage behavior was not part of this browser check. The test
server and browser were stopped afterwards.

The request-level regression coverage lives in
[`server/file_type_filter_test.go`](../../../server/file_type_filter_test.go).

## Folder-navigation follow-up

- `CGO_ENABLED=1 go test -tags 'sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_icu' ./server -run '^TestBrowseFolderNavigationPreservesSheetAndUpdatesItsScope$' -count=1` passed.
- The full suite, with the same exclusions above, still failed on the unavailable
  S3 endpoint at `localhost:7070`. Log: `/tmp/opencode/browse-folder-sheet-go-test.log`.
- Playwright CLI on the current SQLite-tagged build at 1440×900 passed folder-text,
  list-row, table-row, parent-button and breadcrumb navigation with Filters open.
  The selected tab survived, the panel targeted the destination folder, and Reset
  retained that folder. There was only one Filters dialog. A closed sheet stayed
  closed; navigating to Inbox removed the Files filter sheet.
- The disposable browser and server were stopped after verification.
