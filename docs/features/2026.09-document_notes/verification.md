# Document Notes Verification

Date: 2026-09-07. All six slices are implemented. See [plan](plan.md) and
[invariants](invariants.md). Checks below were executed against the implemented feature.

## Recorded Checks

The multiline Inbox merge warning clipping was reproduced in desktop/mobile browser
checks. Checkbox rows now use a minimum rather than fixed 48px height and preserve
the control width. The exact German warning fits naturally (80px on mobile), without
top clipping or overlap with search. Desktop/mobile merge journeys, the temporary-session
checkbox browser check, widget and Notes Go tests, build, and diff checks passed.
German screenshots were inspected at `/tmp/opencode/checkbox-german-{desktop,mobile}.png`.
Create-space checkboxes were checked manually; two existing automated space-management
journeys were blocked by an obsolete "Manage spaces" navigation locator. Nested tag
checkbox consumers were inspected in code but not browser-tested for this fix.

| Check | Recorded result |
| --- | --- |
| `go build ./...` | Passed, including after the final UI edits. |
| `go test ./... -skip 'TestConcurrentUploadFileCmdWithSlowS3\|TestUploadFileCmdFailsWhenS3Unavailable\|TestUploadFilesCmdFailsWhenS3Unavailable'` | Passed; server approximately 92s. Unrelated slow bad-network tests intentionally excluded. |
| `go vet ./action/browse ./action/inbox ./action/trash ./action/common ./model/tenant/file ./model/tenant/filesystem ./server ./core/ui/widget` | Passed. |
| `go test ./server -run 'TestDocumentNotes\|TestFileVersionFromInboxCmd' -count=1` | Passed. |
| `go test -race ./server -run 'TestDocumentNotesCmdHTTPConcurrentReplacement\|TestDocumentNotesCmdHTTP' -count=1` | Passed; this does not imply every planned race scenario exists. |
| `npm run test:e2e -- e2e/document_notes.spec.ts` | All 15 desktop/mobile tests passed after review fixes in 1.8m. |
| `go test ./server -run '^TestDocumentNotesUploadFileVersionPreservesNotes$' -count=1 -v` | Passed with encryption enabled and disabled; neither subtest skipped. |
| `git diff --check` | Passed. |
| `go generate .` | Ent generation passed. |
| `go generate ./...` | Ent passed, then failed in the pre-existing language enumer tool: binary export format `v` no longer supported. No generator/tool version changes attempted. |
| i18n `go generate` | Passed; de/fr/it catalogs refreshed with fuzzy translations and comments. Known unrelated missing-message warnings remain. |

The browser suite ran against a disposable instance at `http://localhost:7088` with
the notes-test account. No credentials belong in this record. It covers visible-control
create/edit/delete/replace, cancellation and invalid input, keyboard and rapid history
toggles, Inbox folder filing, all-history merge, Trash/restore, persistence, and overflow.

The final build/browser run includes explicit history-button synchronization and a
translated Notes toolbar aria-label.

## List-Item Context Menus

The follow-up presentation change uses the standard `ListItem` with unabridged
`DocumentNoteContent` in its content slot. Edit, Replace, and Delete are no longer
inline buttons; eligible current notes expose them through a context menu, opened
by right-click or a trailing vertical-three-dot icon button with an Actions tooltip.
Historical and Trash entries omit
the mutation menu. Existing author/owner checks are unchanged.

Browser coverage exercises right-click on desktop, the Actions button on mobile,
keyboard opening/activation, immediate Escape dismissal with focus return, and all
existing note lifecycle journeys. The Go suite passed again with the same three
unrelated bad-network tests excluded; scoped vet and translation generation passed.
Desktop/mobile menu screenshots were inspected in `/tmp/opencode/notes-menu-desktop.png`
and `/tmp/opencode/notes-menu-mobile.png`.

The subsequent icon-button refinement passed the focused Go rendering/permission
tests and both desktop/mobile keyboard menu journeys, including Escape focus return.

The Notes toolbar now uses icon-only `history` and `add` controls, with tooltips
and accessible names. History exposes a highlighted selected state and `aria-pressed`;
its inverse-state request replaces the switch while command refreshes retain the
current filter. The updated fifteen desktop/mobile tests passed, asserting
icon-only button contents, absence of a switch, keyboard toggling, and filter
retention. Focused Notes/widget Go tests, build, and scoped vet also passed.

The subsequent color correction follows the live Material 3 standard icon-button
spec: Primary icon/state layer when selected, On surface variant when unselected,
and an 8% hover layer without permanent container fill. A computed-style browser
test failed against the old selected color and passed after the correction in
both light and dark themes. That test plus desktop/mobile keyboard-menu regression
checks passed (three tests); focused Go rendering/widget tests and the build passed.

The follow-up review moved Add to the left and History to the right in both layouts.
It also corrected the note Actions icon button's accessible name; its tooltip alone
was attached to a wrapper and left assistive technology with the `more_vert` icon
ligature as the button name. Focused server rendering and desktop/mobile browser
checks now require the “Actions” label.

The Notes empty state now reuses the Fields tab's `EmptyState` and elevated Add
button pattern. Trash omits the creation action. Five focused browser checks passed
on desktop/mobile, covering empty-state creation, returning to empty after deletion,
history filtering, toolbar placement, and light/dark icon colors. Focused Notes HTTP
tests, build, and diff checks also passed. The full suite was not rerun for this change.

The subsequent empty-state/translation review found no confirmed production defect.
It added `TestDocumentNotesHTTPEmptyState` to protect creation controls in empty
Browse/Inbox versus read-only Trash, with history both off and on. That test and
the five focused desktop/mobile browser checks passed against the rebuilt application
with the new English wording. German, French, and Italian source/generated entries
were inspected; localized browser sessions were not exercised in this review.

## Compact Rows and Note Details

Rows now show only author and creation date in their secondary text, with edit
attribution/time in a tooltip. Deleted/Replaced status remains separately visible.
Clicking the row or activating its content button opens a read-only details dialog.
Replacement navigation swaps only the dialog content; replacing the open dialog
itself failed browser verification and was corrected before the final run.

- `npm run test:e2e -- e2e/document_notes.spec.ts`: all 17 desktop/mobile tests passed
  on the rebuilt application (2.3m), including details, keyboard opening, Actions-menu
  isolation, edit tooltip, replacement navigation, and existing lifecycle journeys.
- `go test ./server ./core/ui/widget -run 'TestDocumentNotes|TestListItem' -count=1`:
  passed, including details for legacy/historical/Trash notes, other-author reads,
  mismatched note/document rejection, and revoked-access rejection.
- Build, scoped widget/Browse vet, and `git diff --check`: passed.
- Existing translations are reused; no new translation keys or schema changes.

## Note Titles

The first `todo.md` item adds required title/body submissions, restores the visible
Note label, and changes rows to title/main plus body/secondary. Creator and other
metadata remain in the details dialog. Existing titleless notes use a display-only
translated "Note" fallback. This supersedes the earlier author/date secondary line.

- Generated migration `20260907210921_document_note_titles` adds only nullable
  `document_notes.title` with `ALTER TABLE`; no existing-table rebuild or unrelated
  main migration is included. Production migration tests preserve existing text,
  history, and titleless records.
- `npm run test:e2e -- e2e/document_notes.spec.ts`: all 17 desktop/mobile tests passed,
  covering title/body forms, prefilled edits/replacements, invalid titles, title/body
  rows, details metadata, and existing menu/history/lifecycle journeys. The first
  invocation exceeded the tool timeout; the complete rerun passed.
- `go test ./server -run 'TestDocumentNote' -count=1`: passed, including migration,
  titleless rendering, title validation/rollback, title history, merge, and version
  preservation. Main-agent verification reran and passed the same test selection.
- Affected widget/Browse/model tests, `go build ./...`, scoped vet, and staged and
  unstaged diff checks passed. Ent, migration, and i18n generation completed; title
  validation messages were translated into German, French, and Italian.

## Replaced Summaries and Details Layout

Replaced rows now omit the historical body and show the title plus an italic
reference to the immediate successor's title. Details retain full content and use
separated title/body, subdued metadata, and replacement-navigation sections.
All 17 desktop/mobile Notes browser tests passed, including chained navigation,
historical-body suppression/retention, keyboard interaction, and title fallbacks.
Focused Notes/widget Go tests, build, scoped vet, and diff checks passed. The new
reference text was translated into German, French, and Italian and catalogs regenerated.
Desktop/mobile screenshots and long-content layouts were inspected under
`/tmp/opencode/notes-reference-*`; wrapping and vertical scrolling worked without
horizontal overflow. No schema or shared-widget implementation changes were needed.

The creation-date/author tooltip todo is complete. Rows now compose their native
title from existing translated Created and Author metadata, retaining unknown-value
fallbacks; edited metadata remains in details. Two desktop/mobile create/edit/details
browser checks passed, along with focused HTTP history/escaping, unavailable-author,
and details tests and the build. No new translation keys were needed.

The details dialog now uses the note title instead of the generic Note heading,
without duplicating it in the body. Replacement navigation updates the heading
out of band while replacing only the content. Four desktop/mobile create/edit and
replacement-history browser checks, focused Notes Go tests, and build passed.

Deleted summaries now match replaced summaries: title plus italic status, without
retained body text in the row. Full text remains in details. Focused HTTP rendering
and details tests passed, as did four desktop/mobile history and filing/Trash/restore
browser checks against the rebuilt app on port 7091. The initial browser run on
7088 reached another test instance and was discarded; that instance was left alone.

The behavior-preserving component refactor moved `DocumentNoteContent` and its template
to `ui/uix/component`. Production and test template registration and Tailwind scanning
were updated. Focused Notes server tests, static-page/component checks, build, vet,
CSS generation, and diff checks passed. Four desktop/mobile create/details/history
browser checks passed against the rebuilt application on port 7091.

The application component was subsequently consolidated into the existing
`ui/uix/partial` package, removing the overlapping `component` package. Template
registration and Tailwind scanning follow the new path. Focused Notes server and
static-page tests, build, scoped vet, diff checks, and four desktop/mobile
create/details/history browser checks passed against the rebuilt app on port 7091.

## Textarea Label Alignment

Browser comparison reproduced an 8px vertical mismatch and a second native focus
outline. Labeled textareas now match TextField's 16px content inset and retain only
the outer focus outline; unlabeled OCR/recovery-code styling is unchanged.

- `npm run build:css`, `go build -o /tmp/opencode/simpledms-textarea .`,
  `go test ./core/ui/widget ./action/browse`, scoped widget/Browse vet, and
  `git diff --check`: passed.
- The new `matches textfield label layout` browser test failed before the fix on
  desktop/mobile. Both now pass for empty, keyboard-focused, populated, and long
  multiline input in light/dark themes. The test waits for label transitions before
  recording the reference geometry/colors, avoiding a transient-color comparison.
- The existing `creates a note from the Fields-style empty state` and
  `creates plain-text notes` browser tests also passed on desktop/mobile (four tests).
  The final layout check used `E2E_LOGIN_EMAIL=notes-test@example.com npm run test:e2e --
  --config /tmp/opencode/textarea-playwright.config.ts e2e/document_notes.spec.ts
  -g 'matches textfield label layout'`. The temporary config reuses existing
  authentication against the rebuilt app on 7088.
- `go test ./server -run
  '^Test(PasskeyRecoveryCodesDialogConsumesToken|AdminPasskeyRecoveryCmdReturnsCodesForAdmin)$'
  -count=1`: passed for existing recovery-code consumers.
- Desktop/mobile screenshots inspected: `/tmp/opencode/textarea-fixed-desktop.png`
  and `/tmp/opencode/textarea-fixed-mobile.png`. Full suites were not rerun for this fix.

## Schema and Test Coverage

- Generated tenant migration `20260907090139` adds only `document_notes` and its indexes.
  Production migration replay tests passed for fresh and populated databases. No legacy
  backfill is performed; legacy `File.notes` remains until its first mutation/source merge.
- Unrelated main WebDAV index drift captured by `cmd/migrate` was removed and the original
  main checksum restored. No main migration is included.
- `server/document_notes_model_test.go`: lifecycle/persistence, permissions and scope,
  attribution, lazy legacy handling, transfer retry/rollback, and real Inbox merge.
- `server/document_notes_cmd_test.go`: HTTP rendering/escaping, errors/permissions,
  all three tab sets in both folder modes, two competing replacements, and rollback after
  a real SQLite constraint failure during legacy replacement. The losing concurrent
  request may return a SQLite BUSY-derived 500 rather than a domain 409; exactly one
  successor must commit. Removed-author rendering preserves stored author/editor IDs
  while showing unknown attribution, without bypassing the deleted-user filter.
- `server/document_notes_filing_test.go`: HTTP Inbox-root filing, missing-parent restore
  into Inbox, and non-folder Mark as done, retaining current/deleted/replaced history.
- `server/document_notes_migration_test.go`: fresh/populated production migration replay.
- `server/document_notes_versions_test.go`: ordinary `UploadFileVersionCmd` increments
  the version while retaining all note states, identities, authorship, edit attribution,
  replacement links, and legacy text, with file encryption enabled and disabled.

## Visual Inspection and Remaining Optional/Manual Verification

CLI-driven visual checks inspected desktop 1440×1000 and mobile 390×844 screenshots:
`/tmp/opencode/notes-desktop.png` and `/tmp/opencode/notes-mobile.png`. Layout was readable
and scrolling worked. These are temporary artifacts, not committed evidence or a full
independent manual acceptance walkthrough.

- [x] Rerun build and notes browser suite after the final minor UI edits.
- [ ] Independent upgraded legacy-data walkthrough, including unknown author/date display.
- [ ] Independent owner editing another author's note and non-folder filing walkthroughs.
  These areas have Go request/fixture coverage, not independent manual/browser coverage.
- [x] Dedicated ordinary new-version upload with current/deleted/replaced notes present.
- [ ] Explicit edit-versus-replace and delete-versus-replace concurrent request tests;
  the implemented concurrent test is replacement-versus-replacement.
- [ ] Independent long-history/long-name and keyboard dialog focus/return audit in both
  layouts; automated keyboard history switching and visual readability are narrower checks.
- [ ] Independent manual cancellation of the revised Inbox merge confirmation.
  The desktop/mobile merge tests already cancel the confirmation before reopening
  and submitting it; this remaining item is only an independent manual walkthrough.

Unchecked compound items in slice checklists retain the original proposed coverage where
not fully evidenced. They are remaining optional/manual verification, not unimplemented
note actions. Recursive generation remains blocked by the unrelated enumer incompatibility.
