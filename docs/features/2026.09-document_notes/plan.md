# Document Notes Plan

Status: All six slices implemented; verification recorded with remaining checks below

Date: 2026-09-07

Contract: [Specification](spec.md) and [durable invariants](invariants.md).
The specification's no-planning non-goal applied to its authoring step; this plan
is authorized by the subsequent planning request. Product scope is unchanged.

## Ordered Slices

1. [x] [Create and read a note in Browse](slices/01-create_browse_note.md)
2. [x] [Read existing notes in every document view](slices/02-read_notes_all_views.md)
3. [x] [Annotate an Inbox document and keep notes when filing it](slices/03-annotate_inbox_document.md)
4. [x] [Edit a current note with correct attribution](slices/04-edit_current_note.md)
5. [x] [Soft delete a note and reveal it with the history toggle](slices/05-delete_and_show_history.md)
6. [x] [Replace a note while retaining its history](slices/06-replace_note.md)

## Delivery Record

See [verification](verification.md) for commands, evidence boundaries, and remaining
optional/manual checks. Slice completion denotes delivered implementation, not completion
of every originally proposed independent manual walkthrough. The final build and all fifteen
desktop/mobile browser tests passed after the icon-only history/add toolbar and synchronization
aria-label edits.

Delivered differences from the plan:

- Reused the existing `TextArea`; no multiline extension to `TextField` was needed.
- Follow-up: notes now use `ListItem.Content` with a permission-aware context menu,
  rather than a custom row with inline actions. Right-click, touch, and keyboard
  menu journeys are covered by the expanded browser suite.
- Used explicit note `deleted_at`, not `SoftDeleteMixin`: its skip context can turn
  deletes into hard deletes. Current/history filtering is explicit.
- `DocumentNotes` owns validation, access checks, persistence, and transitions in the
  caller's transaction; no separate note repository or model package was introduced.
  Read rendering computes owner eligibility once rather than querying per row.
- Legacy text is materialized lazily on mutation or source merge, with no backfill.
- Notes use their own form-dialog auto-close, not global `closeDialog`, which also
  closed the parent mobile Details sheet. History refresh retains the rendered state
  in `hx-vals`; the history icon sends the inverse state.
- Inbox root filing required a resolved hidden `CurrentDirID` in the existing Move
  dialog and allowing filing when the root was already the parent; an HTTP test covers it.
- Coverage lives in five new `server/document_notes_*_test.go` files rather than
  modifying older merge tests, plus `e2e/document_notes.spec.ts`.
- One generated tenant migration adds the new table and indexes. No existing-table
  rebuild or main migration ships. The unrelated main WebDAV index drift captured by
  migration generation was removed and its checksum restored.
- Ent generation succeeded via `go generate .`; recursive generation later failed in
  the pre-existing language enumer tool. i18n generation succeeded separately; no tool
  versions were changed. See the verification record for exact caveats.

Each slice is one implementation session and includes its own persistence,
authorization, UI, translations, and tests where consumed. Do not extract a
foundation-only or final testing slice. Later actions remain absent, not stubbed,
until their slice is complete. Earlier journeys must continue to work.

## Relevant Integration Points

- The three `action/{browse,inbox,trash}/file_tabs_partial.go` files compose tabs.
  Inbox already reuses Browse partials; reuse a shared notes presentation with
  view-specific composition rather than three implementations.
- `action/browse/file_versions_partial.go` demonstrates event-driven partial
  refresh. Use registered commands through `server/router.go` transaction wrapping,
  thin handlers, and domain methods for note transitions.
- `common/file_repository.go` scopes document lookup to the current Space and
  distinguishes ordinary from include-deleted reads. The model-level repository
  interface is smaller; extend it only for a concrete consuming need.
- `ctxx/space_context.go:UserRoleInSpace` treats tenant owners as Space owners.
  `spacerole.SpaceRole` currently has User and Owner, not a working Viewer role.
- `db/enttenant/schema/{space_mixin,common_mixin,soft_delete_mixin}.go` provide
  existing conventions. `SkipSoftDelete` includes deleted rows on reads but
  changes delete operations to physical deletion; do not carry it into note deletes.
- `model/tenant/file/file_version_from_inbox_service.go` deletes the source
  permanently. Any new note reference must survive that operation from its first
  consuming slice, not wait for history support.
- `server/file_version_from_inbox_cmd_test.go` and the action harness exercise
  real SQLite-backed commands. Transaction rollback assertions must use the
  transaction-owning request path, not assume a directly called handler rolls back.
- `e2e/helpers.ts`, `e2e/browse_upload_filters.spec.ts`, and `e2e/README.md` provide
  login, Space creation, uploads, and environment setup. The existing Playwright
  config runs Desktop Chrome against an externally running app.

## Working Assumptions

- No blocking product questions remain. Follow the confirmed permissions and
  lifecycle decisions; no architecture or planner delegation is needed.
- Keep legacy `File.notes` as one unknown-author list entry while untouched.
  Materialize it into a normal note only on its first permitted mutation or when
  a version merge would destroy its source. Clear the legacy field only in that
  same successful transaction. This avoids a background backfill and keeps reads
  read-only. Use an explicit legacy selector with the document PublicID, never a
  client-supplied author or internal numeric identifier.
- Legacy note creation time is unknown: label it accordingly and place an
  undated legacy entry after dated entries. Do not present import time or the
  document upload time as the time someone authored the legacy text.
- Current-note text edits may follow existing last-successful-write behaviour;
  this feature adds no merge editor. Every mutation still rechecks document
  access, Trash status, ownership, and note state, so stale historical writes fail.
- Ordinary moves and Inbox version merges are within the current Space. Transfer
  reassigns notes, not authorship; it does not require permission to edit every
  transferred author's text because no note content is being edited.
- File transfers, soft deletion, and replacement are not text edits. Do not use
  generic row `updated_at` alone to decide whether to show an edited indicator.
- Reuse Material 3 tabs, lists, dialog/form controls, toolbar, icon buttons, and existing
  snackbar/error patterns. Add multiline widget support only if existing form
  widgets cannot provide it; `TextField` currently renders an input.
- Keep history state local to the displayed Notes tab and preserve it in partial
  refresh requests. No new account-wide preference or navigation model is needed.

## Verification Contract

Each slice's checklist includes these gates, with only its relevant tests added:

- Run focused Go model/action/server tests against actual SQLite, including
  authorization through public behaviour. Keep production code free of test seams.
- Run `go build ./...`, format touched Go files, and run `git diff --check`.
- For schema changes, edit schema sources, run `go generate ./...`, and generate
  migrations through `go run ./cmd/migrate <slice-specific-name>`. Never hand-edit
  generated files or migrations. Verify upgrade and fresh-schema paths, and follow
  the repository's additive SQLite/index sequencing rules for existing tables.
- Add notes browser coverage to `e2e/document_notes.spec.ts`. Run
  `npm run test:e2e -- e2e/document_notes.spec.ts`. Define desktop and mobile
  describe contexts inside this suite so the command exercises both without
  rerunning unrelated suites under a new global project. Use a desktop viewport
  and a touch/mobile context around 390px wide, with unique fixture data per run.
- Use the existing authenticated setup on a disposable development instance.
  Browser checks must traverse visible controls, not seed notes by bypassing the
  feature. Legacy fixtures and exhaustive role/failure cases belong in Go tests.
- Localize new labels/messages in each slice's `messages.gotext.json` changes,
  with translation comments, `fuzzy: true`, and repository language conventions.
- Record actual commands, results, manual checks, and unavailable prerequisites
  when implementing. Do not mark a slice complete merely because tests were added.
  Do not run unrelated slow bad-network tests.

## Coverage Map

Specification acceptance criteria are numbered in [spec.md](spec.md#acceptance-criteria).

| Slice | Primary coverage | Regression coverage extended there |
| --- | --- | --- |
| 01 | 2, 3, 11, 15 | 13: notes survive ordinary moves and file versions |
| 02 | 1, 12, 14 | 11, 13, 15; legacy and active-note merge safety |
| 03 | 2 in Inbox, 13, 16 for current notes | 1, 3, 11, 15 |
| 04 | 4, 10 | 3, 11, 12, 14, 15 |
| 05 | 7, 8 for deleted entries | 9, 10, 11, 12, 13, 15, 16 |
| 06 | 5, 6, 8 with mixed history, 9 | 10, 11, 12, 13, 15, 16 with replacement chains |
