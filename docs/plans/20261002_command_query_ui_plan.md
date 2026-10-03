# Command/query UI refactoring plan and execution record

Started: 2026-10-02. Updated: 2026-10-03. Written before production edits.

Related: [audit](../specs/20261002_command_query_ui_audit.md),
[ADR](../adrs/2026.10-command_query_ui_flow.md),
[contract](../../AGENTS.md#command--query-ui-flow).

## Implementation slices

Implementation, scoped checks and the requested follow-up browser pass are delivered.
The follow-up record below supersedes the initial browser blockers; checked items
do not assert exhaustive coverage of every possible interaction.

- [x] **Infrastructure:** default write-response buffering in the router; drain
  queued feedback once; migrate and remove `QueryHeader`/`X-Query-*` dispatch.
  Keep read-only POST classification and streaming reads. Inspect manual commands
  individually. Verify rollback, commit failure, explicit secret response, and
  actual POST payload decoding. Generic runtime decoding is used in this checkout.
- [x] **Browse/Inbox lifecycle and preferences:** move, complete, upload, type
  assignment, list preferences, and nested folder creation emit file/capability
  notifications. Reuse list/Inbox/metadata/chooser queries; distinguish parent
  lifecycle refresh from child metadata refresh. Preserve directory, selected
  public ID, search, source/tag/property filters, sort, tab and side sheet; query
  chooses next Inbox record when selected file leaves the queue. Verify filtered
  empty/nonempty lists and repeated commands after morph.
- [x] **Inbox staged imports:** move pending-upload consumption from page queries
  into a context-selection command. Reuse ingestion service and tenant/Space
  authorization. Explicit exception: navigate to the selected Space after commit.
  GET/POST Inbox queries must not convert staged files. Verify reader rejection
  and replay/empty token behavior without external URL imports.
- [x] **Tagging:** assign/unassign/toggle/create-and-assign and subtag commands
  return feedback/events only. Add assignment/subtag chooser subscriptions, reuse
  their POST queries, preserve expansion/focus and file identity; remove direct
  row targets. Keep manage-tag view toggling read-only. Verify empty chooser,
  nested creation, repeated toggles, and read/write permission checks.
- [x] **Document types/fields:** reuse document-type/list/detail queries to
  retain selection and clear deleted detail; separate parent type changes from
  attribute child changes. Remove custom form replacement targets. Property
  CRUD retains existing event/list chain. Verify selected rename/deletion and
  repeated attribute edits with actual POST payloads.
- [x] **Spaces/users/settings/credentials/auth:** retain existing CRUD islands;
  add missing navigation/dashboard and standalone completion subscriptions.
  Credential issuance keeps only one-time secret result and emits destination-
  carrying capability event; query preserves status filter/tab. Remove obsolete
  OOB overview helper. Keep authentication/language/native exceptions documented.
  Verify temporary credential issuance/revocation, account form completion, Space
  name propagation and owner/reader behavior; do not send test communications.
- [x] **Navigation:** reuse existing morph and rail runtime, preserving only
  client-owned expanded/group state while destinations/context remain server-owned.
  Test raw boosted links, desktop/mobile, and frame-by-frame rail width; refresh
  URLs must retain tenant/Space context.
- [x] **Final browser closeout:** the second source audit and follow-up defect/state
  checks are complete; see the follow-up record for actual coverage and limitations.
  The original verification scope was every changed event/listener/
  query, actual partial payload, standalone completion, helper rendering, target,
  query writes, filters, context and duplicate reload. Delegate tests/browser checks.
  Establish baseline evidence for any unrelated failure in an isolated worktree.

## Execution / verification record

- Initial worktree: clean (`git status --short`). No user changes overwritten.
- Initial inspection: repository/global guidance; architecture-selection,
  effective-go, HTMX and Playwright skills; existing ADRs/plans, router, rendering,
  widgets, capabilities and browser helpers. `gopls` document symbols and `go doc`
  used for navigation. No production edits preceded this plan.
- First vertical-slice check (delegated):
  `go test ./server -run 'Test(MarkAsDone|.*CommandQuery)' -count=1 .` passed before
  expanding into credential/navigation changes.
- Registration audit correction: `SelectDirMakeDirCmd` and `SelectDirCmd` are
  unused prototypes, not a reachable nested workflow. The active move form creates
  a new destination as part of the move; no separate create/select event is needed.
  Browser Space roles are Owner/User; read-only MCP credentials are distinct from
  browser roles. Do not invent a browser Reader fixture.
- `AGENTS.md` was excluded by `.gitignore`; removed that entry so the requested
  authoritative guidance can be included in the repository changes.
- Final passing commands (delegated):
  - `go test ./action/... ./core/ui/widget ./ui/... ./util/httpx`
  - `go test ./server -run 'Test(MarkAsDone|UpdateFileListPreferences|InboxQuery|RegisteredCommandBuffers|BufferedCommand|CreateWebDAVCredential|CreateMCPCredential|MCPCredential|MCPBrowser|FileVersionFromInbox|EditSpaceCmd|SpaceCreatePermissions|SpaceDeletePermissions|SpaceUser)' -count=1`
  - `go test ./server -run 'TestConsumeUploadsCmd' -count=1`
  - `go test ./core/ui/widget ./ui/... ./util/httpx`
  - `go test ./action/browse ./action/inbox` after the final context guards.
  - `go test ./action/browse ./action/auth` after the final stable-island ID and
    one-time-result comment.
  - `git diff --check`
- Builds succeeded with `go build -o /tmp/opencode/simpledms-last-check .` and
  a SQLite-tagged build of `/tmp/opencode/simpledms-ui-final-check`. Generated DB
  code and migrations were not modified. Unrelated slow bad-network tests were
  not run.
- Regression checks exercise commit visibility at the first response write,
  handler/commit failure suppression, obsolete-header rejection, actual Browse,
  Inbox and credential POST payloads, one-time secret/list separation, unconsumed
  staged uploads on GET/POST queries, and explicit-route import authorization.
  During development, test assumptions about native dialog markup, serialized
  filter names, finished-upload fixtures and page registration were corrected.
- Browser builds and Playwright CLI checks used a dedicated instance on port 18764,
  blank mail/OCR/preview endpoints, and isolated metadata. Verified Space CRUD and
  rail invalidation, document-type selected rename/delete, Tag CRUD, and MCP/WebDAV
  create/edit/revoke with separate list POSTs and visible one-time secret dialogs.
  Invalid blank creation did not submit a command. No secrets were printed.
- Browser feedback found and fixed invalid evaluated `hx-vals` object syntax.
  Credential result dialogs now supply creation feedback themselves: a redundant
  OOB snackbar was mounted in the form dialog just before its replacement.
- Desktop navigation samples stayed at 360 px across 12 animation frames; compact
  navigation at 390 px retained the current tenant/Space and had no horizontal overflow.
- A unique bucket on local VersityGW enabled actual Uppy uploads, metadata fields,
  nested tag creation/assignment, rename, preferences, filtered empty lists, Inbox
  completion/next selection, deletion and Trash restore. Commands and separate
  query POSTs were observed; file metadata updates issued one request per region.
  Browser findings fixed afterward: close-detail search URL preservation,
  processable empty detail roots, retained Inbox search controls, and guards on
  outgoing list/detail refreshes during context changes.
- On the confirmed `/tmp/opencode/simpledms-ui-checked` process, a synthetic
  `fileListPreferencesUpdated` notification issued a real Browse list query and
  preserved the Tags tab with the side sheet both open and closed. The synthetic
  trigger is distinguished from the earlier actual preference-control checks:
  the selected-detail layout hides that toolbar at the tested width. Side-sheet
  initialization now skips already-initialized retained nodes. An Inbox-root
  `fileUploaded` notification issued its real POST query with HTTP 200 and no
  attached stale detail listener or new console error.
- Baseline evidence: clean detached worktree at
  `d26bcfacce0b1f78bb72a95f9dc0359ad73c366b`, separately built and served on 18765.
  Both baseline and current reproduced (1) Tag deletion after soft-deleting an
  assigned file failing with a foreign-key constraint, and (2) merge renaming a
  target to a still-existing source name failing the unique constraint on
  `(files.space_id, files.name, files.parent_id)`. These domain defects were not
  changed by this UI migration. A later Mark-as-done retry also collided with a
  previously filed test filename; it released no success event/query. Earlier
  non-colliding completion/next-selection checks succeeded.

### Initial closeout limitations (superseded by the follow-up below)

- The final race guard requires a settled-transition browser rerun. The last
  current-source check captured no stale preview request, but dispatched its
  notification after the URL changed and before Inbox content settled; outgoing
  Browse queries were still observed. List guards were added afterward. A final
  isolated retry could not establish a running server: launch timed out and
  readiness returned connection refused. The latest guards compile and scoped
  package tests pass, but this browser timing case is **not signed off**.
- Focus/scroll retention across every refreshed form, and all standalone
  wrapper-none custom forms, were not exhaustively browser-tested. The actual
  standalone sign-in/import transports retain their explicit navigation behavior.
- A successful browser merge was blocked by the baseline collision above; the
  successful merge/event contract is covered by the scoped server tests.

### Cleanup

- Task-owned metadata (`tmp/opencode/simpledms-e2e-meta`, `-1811`, `-20261003`, and
  `.tmp/command-query-final-check`), baseline worktree and temporary S3 buckets were
  removed. The buckets were `simpledms-browser-20261002-a81cd9`,
  `simpledms-browser-replay-20261003-7f3a1c`, and
  `simpledms-baseline-20261003-3b7e0f`; the final listing found none of them.
- Test credentials were revoked in the completed credential journey; removal of
  isolated databases removed the later fixtures as well. No test server or browser
  session remains. Other worktrees and the existing VersityGW container were retained.
- Remaining artifacts only: `/tmp/opencode/simpledms-last-check`,
  `/tmp/opencode/simpledms-last-check.log`, `/tmp/opencode/simpledms-ui-final-check`,
  and `/tmp/opencode/simpledms-ui-final-check.log`. Earlier temporary logs were lost
  when the environment restarted; the baseline observations above were recorded
  before that restart.
- No commits or pushes were made. `.env` was unchanged.
  Absolute `-meta` paths are joined beneath the working directory by NewServer;
  later runs used an explicit repository-relative path. No existing business
  records or external communication services were used.

## Follow-up: defect fixes and another verification pass — 2026-10-03

The user requested fixes for both reproduced domain defects and another audit/run.
Existing unrelated changes were preserved; no schema, generated code, migrations,
configuration files, commits or pushes were introduced in this follow-up.

### Changes

- `EntTagRepository.DeleteTag` removes the Tag's assignment rows in the caller's
  transaction, including assignments on files in Trash. Files, unrelated assignments,
  Space isolation and existing attribute/group constraints are retained.
- `FileVersionFromInboxService` checks the incoming filename against **other filed
  documents**, retaining the target's name if occupied. The source Inbox record
  itself is excluded by the existing partial unique index; the original diagnosis
  that it alone caused the collision was too broad. Non-colliding merges keep their
  existing filename behavior.
- Merge cleanup now removes source tag/field assignments and WebDAV upload aliases
  before the source is hard-deleted, after transferring notes. Source validity is
  checked before mutation. This closes an additional foreign-key failure exposed
  when the incoming source has classification data.
- The repeat UI source pass found two small omissions: Manage Tags' Delete control
  now explicitly uses swap-none, and retry-preview no longer requires a replacement
  target. Existing widgets and event/query ownership are retained.

### Regression and package checks

Added `server/tag_deletion_regression_test.go` and
`server/inbox_merge_regression_test.go`. Tests exercise real command registration
and actual follow-up POST partials, including active/trashed assignments, unrelated
and cross-Space records, protected attribute references, colliding filed names,
target classification retention, source metadata removal, notes and versions.

- Red: `go test ./server -run 'TestTagDeletionRegression|TestInboxMergeRegression' -count=1`
  reproduced the merge's unique-constraint failure. After correcting fixture order
  (assign before trashing), `go test ./server -run '^TestTagDeletionRegression' -count=1`
  reproduced the assignment foreign-key failure at the command endpoint.
- Green: `go test ./server -run 'TestTagDeletionRegression|TestInboxMergeRegression|TestFileVersionFromInboxCmd' -count=1`.
- Final passing package check:
  `go test ./model/tenant/file ./model/tenant/tagging ./action/browse ./action/inbox ./action/tagging ./action/managetags ./core/ui/widget`.
- Final passing contract check:
  `go test ./server -run 'Test(TagDeletionRegression|InboxMergeRegression|FileVersionFromInboxCmd|MarkAsDone|ConsumeUploadsCmd|RegisteredCommandBuffers|BufferedCommand|UpdateFileListPreferences|InboxQuery|CreateWebDAVCredential|CreateMCPCredential|MCPCredential|MCPBrowser|SpaceUser)' -count=1`.
- `git diff --check` passed. Verification was delegated; unrelated slow bad-network
  suites were not run.

### Browser replay

The successful sandbox ran from `/tmp/opencode/cq-isolated-browser`, clearing
inherited SimpleDMS environment variables and avoiding the repository `.env`.
It used local HTTP port 18764, a dedicated local VersityGW bucket, and disabled mail,
OCR and preview conversion. This resolved the earlier configuration/authentication
setup failures. No existing business data or external communications were used.

Actual rendered controls verified:

- Collision merge: command HTTP 200, `fileVersionMerged, closeDialog`, swap-none,
  feedback-only response; exactly one parent-list POST HTTP 200. Target retained
  its identity/name and showed two versions; merged source disappeared.
- Tag deletion with active and trashed assignments: command HTTP 200/`tagDeleted`,
  separate tag-list POST HTTP 200; files remained, and the trashed file was restored.
- Show assigned → edit chooser → unassign/reassign worked after replacements. Each
  mutation refreshed the count, assignment island and file list once.
- Nested create-and-assign returned to the existing chooser with parent and new
  child visible through the query response.
- A delayed text-property command and query retained the input, value and caret.
  After creating enough real fields to make the sheet scrollable, the final run
  retained focus on `file-property-13` and scrollTop **729** (scrollHeight 1264,
  clientHeight 535); one field query and one file-list query returned HTTP 200.
- Filename search survived detail navigation and closing the preview in both URL
  and input. Open and closed Fields sheets retained their state/tab after a
  **synthetic** `fileListPreferencesUpdated` notification issued a real parent query.
- Browse → Inbox navigation completed without a stale preview request. After
  waiting for `#inboxListWrapper`, a **synthetic** `fileUploaded` notification issued
  exactly one Inbox file-list POST HTTP 200 and no Browse POSTs.

The final binary was built with
`go build -tags "sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_icu" -o /tmp/opencode/simpledms-defect-final .`;
the process executable was verified, SHA-256
`261de91b8e234f14511832717a08a3cf31cda8b0b2479be87c8d17dd80245d9a`.
Its tag-deletion control had explicit swap-none and no control-owned replacement
target. No application JavaScript exceptions occurred in the final flows; a
favicon 404 and sign-in autocomplete advisory were observed.

Limitations: source metadata/note transfer is covered by the HTTP regression rather
than the browser merge. Retry-preview's target change was source/package-checked;
the sandbox had no failed PDF-preview state. Synthetic notifications above verify
state-refresh listeners, not a new upload or preference-control submission.

### Follow-up cleanup

The test process and browser were stopped. The isolated `meta` directory and bucket
`cq-defects-20261003-isolated` were deleted, and the bucket's absence was confirmed.
The existing VersityGW container was retained. There are no surviving temporary
business records. Temporary binaries, launcher scripts and logs remain under
`/tmp/opencode`, including `simpledms-defect-final`, `simpledms-defect-recheck`,
`start_cq_browser.py` and `cq-isolated-browser/server.log`.
