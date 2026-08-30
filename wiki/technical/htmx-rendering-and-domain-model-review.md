# HTMX Rendering and Domain Model Review

Review snapshot: 2026-08-30

This review covers server-rendered HTML, HTMX request and response flows,
browser-side rendering behavior, and business rules that should move from
actions or schedulers into existing model types. Priorities combine security,
data-integrity risk, user-visible correctness, and implementation leverage.

## Priority summary

| Priority | Meaning | Main themes |
| --- | --- | --- |
| P0 | Address immediately | Stored XSS, action dispatch bypass, cross-Space metadata, stale OCR |
| P1 | High risk or high user impact | Sessions, history, races, core invariants |
| P2 | Important consistency work | Response contracts, accessibility, lifecycle consolidation |
| P3 | Opportunistic cleanup | Duplicate checks, small model APIs, test and UI polish |

## Current strengths

- The backend remains the source of truth. Actions render HTML rather than
  maintaining a second client-side state model.
- Many pages share full-page and fragment composition through
  `action/common/page.go:27-45`. Browse, Inbox, Trash, and some older pages
  still duplicate this decision and should converge on it incrementally.
- Commands commonly emit named browser events from `ui/uix/event/events.go`,
  and stable fragments listen for the events that affect them.
- Browse and Inbox already return small pagination and target-specific
  fragments instead of replacing the complete page
  (`action/inbox/files_list_partial.go:151-205`,
  `action/browse/list_dir_partial.go:146-239`).
- Debounced list searches use replacement synchronization
  (`action/inbox/files_list_partial.go:329-355`,
  `action/browse/list_dir_partial.go:474-498`).
- A global HTMX progress indicator is inherited from the page body
  (`core/ui/widget/base.gohtml:158-176`).
- HTML source preview already has a safe text-only handler with `nosniff` and a
  sandbox CSP (`action/download/preview.go:49-85`). It should become the only
  HTML preview path.
- Server integration tests, file-list benchmarks, and focused Playwright tests
  provide a useful base for regression coverage.

## P0: Address immediately

### Block active stored documents from same-origin inline rendering

Uploaded active content can currently execute under the SimpleDMS origin:

- `FilePreview.IsPreviewable` treats every `text/*` response as previewable
  (`core/ui/widget/file_preview.go:42-45`).
- Previewable files are loaded into an unsandboxed `<object>`
  (`core/ui/widget/file_preview.gohtml:34-103`).
- When PDF conversion is unavailable, the preview falls back to the original
  inline download URL (`action/browse/file_preview_partial.go:67-87`).
- `StreamDownload` accepts `?inline=1`, returns the stored MIME type, and sets
  neither a sandbox CSP nor `X-Content-Type-Options: nosniff`
  (`action/common/download_helper.go:37-50`).

An uploaded HTML or other active document can therefore run same-origin script
when another authorized user previews it. The raw inline URL is also directly
constructible, so fixing only the preview widget is insufficient.

Recommendation:

1. Make the download handler decide inline eligibility from a narrow server-side
   MIME allowlist; do not trust the `inline=1` query parameter for active types.
2. Force HTML, XHTML, SVG-as-document, XML, and other active formats to
   attachment, or serve them through a dedicated inert preview response.
3. Route HTML preview through the existing `OriginalSourceHandler`, which
   returns `text/plain`, `nosniff`, and `Content-Security-Policy: sandbox`.
4. Add `nosniff` to every stored-file response.
5. Use a separate untrusted-content origin only if rich active-document
   previews become a product requirement.

Smallest regression test: upload HTML containing script, request both the
preview and hand-built inline URL, and prove script cannot access the parent,
read same-origin protected responses, or issue authenticated mutations.

### Remove client-selected synthetic action dispatch

`server/router.go:167-241` accepts `X-Query-Endpoint` and `X-Query-Data` from
the browser, looks up any registered action, rewrites the current request, and
calls that action's handler directly. The selected action does not pass through
its route wrapper, method declaration, setup-session allowlist, or transaction
selection. An unknown endpoint can also dereference a missing map entry.

There is a concrete setup-session tunnel: an allowed endpoint such as
`RegeneratePasskeyCodesCmd`
(`action/auth/regenerate_passkey_codes_cmd.go:26-39`) can name a normally
disallowed command in `X-Query-Endpoint`. Setup-session checks apply only to the
outer path (`server/router.go:847-855`). Tenant and Space context for non-GET
HTMX requests is also derived from client-controlled `HX-Current-URL`
(`server/router.go:811-836`).

Recommendation:

1. Reject synthetic dispatch during setup sessions immediately.
2. Reject unknown endpoints and every endpoint outside an exact server-side
   allowlist for the outer command. Do not use `Config.IsReadOnly()` as the
   boundary: manual-transaction actions report read-only while performing
   writes (`util/actionx/common.go:64-69`).
3. Permit only a server-declared command to side-effect-free partial pairing if
   an immediate compatibility fix is required.
4. Remove the protocol incrementally: commands return snackbar/OOB feedback and
   `HX-Trigger`; affected partials issue ordinary requests after commit.
5. Keep all authorization and Space-scoping checks in the ordinary route path.

Smallest regression test: send an allowed setup-session command with a
disallowed command in `X-Query-Endpoint` and prove the second command does not
run. Also cover an unknown endpoint, an unauthorized action pairing, and a
manual-transaction write action such as archive extraction.

### Enforce same-Space metadata relations in model APIs

Several forms restrict visible choices to the current Space, but mutation does
not consistently revalidate submitted relations:

- File Document Type selection writes the submitted ID directly
  (`action/browse/select_document_type_cmd.go:48-54`).
- Tag Attribute creation reaches the model without a same-Space/type check
  (`action/documenttype/create_attribute_cmd.go:69-84`,
  `model/tenant/documenttype/document_type.go:97-134`).
- Property Attribute creation has the same gap
  (`action/documenttype/add_property_attribute_cmd.go:59-75`,
  `model/tenant/documenttype/document_type.go:136-170`).

UI filtering is not an invariant boundary. A crafted request can relate a File
or Document Type to metadata from another Space in the same tenant.

Recommendation:

```go
func (qq *File) SelectDocumentType(ctx ctxx.Context, documentTypeID int64) error
func (qq *DocumentType) CreateTagAttribute(...) error
func (qq *DocumentType) CreatePropertyAttribute(...) error
```

Each method should resolve the referenced entity in `qq.Data.SpaceID` inside
the same tenant write transaction. A Tag Attribute must also reference a group
Tag. Test all operations with entities from two Spaces.

### Make OCR results conditional on the processed File version

`scheduler/apply_ocr.go:95-137` reads a current version, performs external
work, and later updates the File unconditionally. A new version can be added
while OCR is running. The stale result can overwrite the reset performed by
`model/tenant/filesystem/s3_file_system.go:2271-2300` and expose/search text
from the wrong version.

Recommendation:

```go
func (qq *File) ResetOCR(ctx ctxx.Context) error
func (qq *File) RecordOCRSuccess(
	ctx ctxx.Context,
	storedFileID int64,
	content string,
	at time.Time,
) error
func (qq *File) RecordOCRFailure(ctx ctxx.Context, storedFileID int64, at time.Time) error
```

Keep Tika and object-store I/O outside transactions. Record the result in one
short conditional write only if `storedFileID` is still the latest version.
Test inserting version 2 before recording OCR for version 1.

## P1: High priority

### Do not expose rendered success before transaction commit

Handlers render into the real response at `server/router.go:340`, while main
and tenant transactions commit later at `server/router.go:356-369`. If a
commit fails, success HTML, events, and headers may already be visible and the
status can no longer be changed reliably.

Recommendation:

- Buffer responses for write actions until every required commit succeeds.
- Do not buffer read-only streaming downloads or long external I/O.
- Move toward command responses containing only feedback and events, followed
  by ordinary post-commit partial requests.
- Add a forced commit-failure test proving no success fragment or event reaches
  the browser.

Buffering fixes false success rendering, not cross-database atomicity. The
tenant-user process below still needs idempotency and reconciliation.

### Correct session failure and HTMX redirect behavior

Session handling has three separate correctness issues:

- Both a missing/expired session (`server/router.go:761-768`) and no cookie
  (`server/router.go:793-799`) use an ordinary `303`. HTMX can follow it inside
  XHR and swap the sign-in fragment into a dialog, row, or list target.
- Every session lookup error, including database failure or cancellation, is
  treated as session expiry (`server/router.go:1031-1056`).
- Invalid-cookie cleanup also deletes through the request transaction, which
  may be read-only (`server/router.go:1020-1024`).
- In SaaS mode, invalidating an account without an active tenant assignment
  deletes sessions through `mainTx`, which may be read-only
  (`server/router.go:1064-1081`). Renewal already uses a writable connection at
  `server/router.go:1091-1097`.

Recommendation:

- Return `HX-Redirect: /` for HTMX in both unauthenticated branches; retain a
  normal redirect for browser navigation.
- Convert only `entmain.IsNotFound(err)` into session expiry. Propagate all
  other lookup errors as server failures.
- Move active-session lookup, creation, renewal, and deletion semantics into
  the existing `SessionService` and use `ReadWriteConn` for invalidation.
- Keep cookie transport and HTMX response headers in the router.

Test expiry from a dialog and list refresh, a simulated lookup failure, and
invalid-cookie and SaaS invalidation during a read-only GET.

### Disable sensitive HTMX history snapshots and stop manual state replacement

The authenticated `#content` element is the HTMX history snapshot
(`core/ui/widget/base.gohtml:168-178`). The active HTMX 2.0.8 bundle keeps ten
snapshots in `sessionStorage` and defaults
`historyRestoreAsHxRequest` to true
(`ui/uix/web/assets/vendor/chunk-UFGKKXHW.js:1`). Filenames and metadata can
remain visible through Back after sign-out, and a cache miss can request a
fragment where HTMX expects restoration content.

Search, filters, and side sheets also call `history.replaceState(null, ...)`
directly (`core/ui/widget/base.gohtml:80-99,214-244`,
`core/ui/widget/dialog.gohtml:157-170,223-246`). HTMX history entries use
`{htmx: true}` and maintain a separate current-history path. Replacing state
with `null` can make Back/Forward change the URL without restoring matching
DOM.

Recommendation:

- Set `historyCacheSize: 0` and `historyRestoreAsHxRequest: false` explicitly in
  the HTMX config. If caching remains, put `hx-history="false"` on every
  authenticated screen.
- Clear `htmx-history-cache` during sign-out as defense in depth.
- Stop calling `history.replaceState` for server-owned state. Return
  `HX-Replace-Url` from the request that applied the filter or side-sheet state.
- Preserve the active bundle settings with a browser test; do not rely on
  dependency defaults.

Test navigation through more than ten protected pages, sign-out then Back, and
Back/Forward after search, filter, and side-sheet changes. URL, controls, title,
and rendered data must always agree.

### Prevent preview polling from changing history or winning a tab race

Pending PDF previews poll every five seconds without joining the tab request's
synchronization group (`action/browse/file_preview_partial.go:112-126,217-234`).
The polling element declares `hx-push-url="false"`, but every status response
with `CurrentDirID` emits `HX-Push-Url`
(`action/browse/file_preview_status_partial.go:63-89`). A slow poll can revert a
newer Original/Preview choice and add duplicate history entries.

Recommendation:

- Emit `HX-Push-Url` only when `data.PushURL` is true for an explicit tab click.
- Put polling and tab requests in the same `#browsePreviewTabs:replace`
  synchronization group.
- Stop polling when the element is removed or the conversion reaches a terminal
  state.

Test by delaying a poll, selecting Original, then releasing the poll. The tab
and URL must remain on Original and one Back action must leave the preview.

### Prevent duplicate form submissions and duplicate submit IDs

The shared form has no synchronization or disabled-element guard
(`core/ui/widget/form.gohtml:3-20`). Dialogs render the same cached submit
button in mobile and desktop locations
(`core/ui/widget/dialog.gohtml:94-110`, `core/ui/widget/dialog.go:79-103`).
`Widget.GetID` caches the ID (`core/ui/widget/widget.go:27-33`), so both buttons
receive the same DOM ID. Scripts can disable or focus the hidden copy while the
visible submit remains active.

Recommendation:

- Add `hx-sync="this:drop"` and `hx-disabled-elt` to mutating forms.
- If two responsive controls remain necessary, render distinct Button
  instances with distinct IDs and disable both through their common form.
- Prefer one semantic submit control when the layout can position it
  responsively without duplication.
- Keep database uniqueness and idempotency checks; browser guards are not
  invariant enforcement.

Test a delayed create-Space request with rapid double submission. Assert one
request, one Space, unique IDs, and a disabled visible submit control.

### Make tenant User projection workflows recoverable

`model/main/tenantuser/tenant_user_service.go:29-69` creates Account and
membership state in the main database, creates a tenant User projection,
generates a password, and queues invitation mail. Main commits before tenant
(`server/router.go:356-369`). If tenant commit fails, the Account and invitation
can persist without a tenant User, while retry is rejected because the email
already exists. Deletion has the inverse cross-database risk
(`model/main/tenantuser/tenant_user_service.go:74-111`).

Recommendation:

- Treat main Account and membership as authoritative.
- Make tenant User projection creation and deletion idempotent.
- Queue invitation only after the projection is confirmed durable.
- Add a narrow reconciliation operation for missing or stale projections.
- Do not add distributed transactions, an event bus, or an outbox until a real
  asynchronous-delivery requirement appears.

Test tenant commit failure after main commit, then retry/reconcile to a valid
User without duplicate Account, membership, password, or invitation.

### Move File lifecycle transitions into `File`

Inbox, deletion, and restoration decisions are duplicated in actions:

- `action/inbox/move_file_cmd.go:47-62`
- `action/inbox/mark_as_done_cmd.go:57-66`
- `action/inbox/assign_file_cmd.go:108-118`
- `action/browse/delete_file_cmd.go:57-70`
- `action/trash/restore_file_cmd.go:59-94`

Assigning a File does not perform the same explicit Inbox check as the other
paths. ZIP extraction also deletes directly instead of reusing one lifecycle
operation (`action/browse/unzip_archive_cmd.go:303-311`).

Recommendation:

```go
func (qq *File) LeaveInbox(ctx ctxx.Context) error
func (qq *File) SoftDelete(ctx ctxx.Context, deleter *enttenant.User) error
func (qq *File) Restore(ctx ctxx.Context, rootDirID int64) (restoredToInbox bool, err error)
```

Keep next-item selection, navigation, snackbars, and HTMX events in actions.
Test non-Inbox transitions, non-empty directory deletion, and restoration when
the original parent no longer exists.

### Apply filename validation to rename and move

Upload and directory creation use `filenamex.IsAllowed`, but move-with-rename
and rename persist submitted names without it
(`model/tenant/filesystem/file_system.go:173-179,190-205`). This allows names
that other creation paths reject and can break path, display, or WebDAV
assumptions.

Recommendation:

- Reuse `filenamex.IsAllowed` in both existing methods before mutation.
- Keep collision and descendant checks in `FileSystem`; they are already model
  concerns.
- Add one table-driven test using the same accepted/rejected names for create,
  rename, and move-with-rename.

### Model typed property assignment and cardinality

Property parsing, typed-column mapping, money conversion, required-value rules,
and read-then-create persistence are split across:

- `action/browse/set_file_property_cmd.go:73-111`
- `action/browse/add_file_property_value_cmd.go:72-147`
- `action/browse/functions.go:104-179`

`db/enttenant/schema/file_property_assignment.go:15-41` permits several value
columns and has no unique File/Property constraint.

Recommendation:

- Add one concrete `AssignmentValue` value object that guarantees exactly one
  typed value and owns money conversion.
- Add `Property.SetFileValue` and `Property.RemoveFileValue`.
- Add a unique File/Property index, including Space only if required by schema
  ownership.
- Keep HTTP field-presence checks in the form handler.

Use one table-driven integration test for every field type, removal, money
rounding, and repeated/concurrent assignment.

### Enforce all Tag topology and assignment invariants

`TagService` currently accepts relations that the UI tries to hide:

- `Create` accepts an arbitrary parent without proving it is a same-Space group
  (`model/tenant/tagging/tag_service.go:26-38`).
- File assignment does not reject group Tags or prove File, Tag, and assignment
  Space agree (`model/tenant/tagging/tag_service.go:58-120`).
- `MoveToGroup` does not reject self, non-group, cross-Space, or cyclic moves
  (`model/tenant/tagging/tag_service.go:123-148`).
- `AssignSubTag` does not validate same-Space endpoints, source type, or
  self-links (`model/tenant/tagging/tag_service.go:150-165`).
- Tag assignment lacks a unique File/Tag constraint
  (`db/enttenant/schema/tag_assignment.go:57-66`).

Recommendation:

- Strengthen the existing service methods rather than adding another service.
- Add `TagType.CanAssignToFile()` only if it removes repeated checks.
- Add a unique File/Tag index and make assignment idempotent.
- Resolve whether cycles are legal before implementing graph traversal.

Test every invalid topology, two Spaces, duplicate assignment, and valid Simple
and Super Tag flows.

### Make Idiomorph swaps opt-in

`core/ui/widget/htmx_attrs.go:122-151` changes every explicit
`innerHTML`/`outerHTML` swap and every default GET swap into a `morph:` swap.
Source comments note cost and navigation breakage; Inbox already works around
retained listeners (`action/inbox/files_list_partial.go:126-129`).

Recommendation:

- Return native HTMX swap values by default.
- Add an explicit `UseMorph` option only for measured focus or state
  preservation cases.
- Test navigation, focused inputs, dialogs, repeated list refreshes, and
  Back/Forward before removing existing workarounds.

## P2: Improve consistency and maintainability

### Define one response contract for each action kind

The current command flow can combine mutation, renderables, browser events,
URL headers, and immediate synthetic partial dispatch. Adopt these contracts
for new and touched code:

| Kind | Response contract |
| --- | --- |
| `*Page` | Full page for normal navigation; primary fragment for HTMX navigation |
| `*Partial` | One stable replaceable fragment |
| `*Widget` | Never routed directly |
| `*Cmd` | Snackbar/OOB feedback plus `HX-Trigger`; no primary HTML fragment |
| `*Dialog` | Dialog fragment on form load; field errors on validation failure |

Do not rewrite all actions at once. Apply the contract per changed flow, then
remove `X-Query-Endpoint` after its final caller is migrated.

### Use one owner request for each target and URL state

Inbox and Browse mix custom `?hx-target=%23...` parameters with native
`HX-Target` (`action/inbox/files_list_partial.go:151-203`,
`action/browse/list_dir_partial.go:146-239`). Inbox source changes trigger both
an outer `#innerContent` request and an inner synchronized list request
(`action/inbox/files_list_partial.go:261-279,329-355`). Concurrent Browse Tag
toggles each derive state from the current URL
(`action/browse/toggle_tag_filter_cmd.go:51-69`). WebDAV status filtering uses
the same unsynchronized event pattern
(`action/dashboard/webdav_credential_filter_dialog.go:68-94`,
`action/dashboard/webdav_credential_list_partial.go:114-130`).

Recommendation:

- Use native `HX-Target` for response selection.
- Prefer dedicated endpoints when list body, load-more rows, and controls have
  materially different data or swaps.
- Give each state transition one owner target; do not refresh an ancestor and
  descendant concurrently for the same event.
- Serialize additive toggles or submit the complete selected set.
- Add `Vary: HX-Request, HX-Target` if these responses become cacheable.

Test two rapid filter changes with the first response delayed. URL, controls,
and results must reflect the final combined selection.

### Keep document title and language synchronized

`<title>` and `<html lang>` exist only in the full Base
(`core/ui/widget/base.gohtml:4-8,65`). HTMX page responses omit Base, and the
language update command emits only an account event. Browser-tab titles,
history titles, and screen-reader language can remain from the previous page or
language.

Recommendation:

- Include a title in every Page fragment and keep HTMX `ignoreTitle: false`.
- Perform a full refresh after the infrequent language change so the root
  `lang` attribute and all translated chrome agree.
- Centralize the full-page/fragment decision instead of duplicating it in
  Browse, Inbox, Trash, and Document Type pages.

### Render semantic, keyboard-operable controls

`HTMXAttrs` adds `role="link"` based on request attributes
(`core/ui/widget/htmx_attrs.go:54-69`,
`core/ui/widget/htmx_attrs.gohtml:23`). This can override native button
semantics while clickable rows and menu `<div>` elements remain unfocusable
(`core/ui/widget/table.gohtml:47-60`,
`core/ui/widget/menu_item.gohtml:3-10`). Hidden radio and checkbox inputs also
remove native keyboard interaction (`core/ui/widget/menu_item.gohtml:52-75`).

Recommendation:

- Remove generic role generation from `HTMXAttrs`.
- Let each widget render a real `<a href>`, `<button>`, radio, or checkbox.
- If a table row remains interactive, provide a focusable link in its primary
  cell rather than making `<tr>` emulate a link.
- Add keyboard-only tests for rows, menus, sorting, view selection, and dialogs.

### Make loading, error, and success feedback accessible

The inherited indicator is useful, but targets do not expose `aria-busy`.
Preview uses a selector for a missing `.js-preview-status` element
(`action/browse/file_preview_partial.go:224-234`). Server and client snackbars
lack a live status role (`core/ui/widget/snackbar.gohtml:34-48`,
`ui/uix/web/assets/snackbar_runtime.js:81-100`). Timeout errors are handled,
but transport failures such as `htmx:sendError` are not
(`core/ui/widget/base.gohtml:409-430`).

Recommendation:

- Toggle `aria-busy` on the target region during requests.
- Fix or remove the preview-specific indicator override.
- Give snackbars an appropriate `role="status"` or `role="alert"` and live
  region behavior.
- Route timeout, send, and network failures through the same client snackbar.

### Reconcile dialog state and release browser resources

Side-sheet URL state is cleared for explicit buttons and backdrop clicks, but
not for native Escape/`close` events
(`core/ui/widget/dialog.gohtml:149-202`). Dialogs also lack an
`aria-labelledby` relationship to their heading. Upload dialogs create Uppy,
Webcam, Audio, and ScreenCapture instances without calling `destroy()`
(`core/ui/widget/file_upload.gohtml:95-184`).

Recommendation:

- Clear side-sheet URL state on `cancel` and `close`, and restore focus to the
  invoker.
- Label each dialog from its heading.
- Store Uppy on the upload root and destroy it on dialog close and
  `htmx:beforeCleanupElement`; stop active media tracks.
- Test Escape, reload, Back/Forward, focus return, and repeated upload-dialog
  opening.

### Avoid duplicate and eager PDF rendering

PDF.js fallback code inside `<object>` executes even when the browser's native
PDF object succeeds. It fetches and renders every page immediately and does not
cancel loading or render tasks when the preview is swapped away
(`core/ui/widget/file_preview.gohtml:35-97`). Large PDFs can be fetched and
rendered twice while detached canvases continue work.

Recommendation:

- Choose one renderer at runtime instead of starting both.
- Initialize PDF.js only when needed, render visible pages lazily, and cancel
  work on `htmx:beforeCleanupElement`.
- Test request count and cleanup with a large delayed PDF.

### Consolidate lower-risk model policies

The following changes improve model ownership but are P2 because current entry
points already provide some protection:

#### Passkey removal

Policy is split between `action/auth/delete_passkey_cmd.go:61-85` and
`action/auth/clear_passkeys_cmd.go:43-53`. Add
`PasskeyService.DeleteOwnCredential` and `Account.ClearPasskeysIfAllowed`.

#### Preview lifecycle

Transitions are split between
`action/browse/retry_pdf_preview_cmd.go:68-89` and
`scheduler/convert_previews.go:143-540`. Add `Claim`, `MarkReady`,
`ResetForRetry`, and `RecordFailure` to the existing model package.

#### Upload failure

`util/uploadx/upload_failure_helpers.go:16-86` performs unconditional state
updates. Add conditional failure methods to the existing upload model.

#### WebDAV lifecycle

Rules are split across `server/webdav/webdav_request_context.go:51-196` and
`scheduler/process_files.go:70-211`. Define allowed state transitions in
`model/tenant/webdavresource`.

Keep WebAuthn ceremony, Gotenberg/S3 calls, worker batching, WebDAV protocol,
and object cleanup outside models. Use conditional updates for asynchronous
state transitions.

### Reuse active-membership policy for quota calculation

`model/tenant/filesystem/storage_quota.go:86-105` duplicates active membership
rules already represented by `TenantAccessService`
(`model/main/tenantaccess/tenant_access_service.go:28-42`). Divergence could
make access and billing/quota counts disagree.

Recommendation: expose a count operation from the existing access service or a
shared predicate builder accepted by the quota query. Do not introduce a new
repository abstraction solely for this reuse.

### Remove submitted values from validation logs

Generic form validation logs `FieldError.Value()`
(`action/util/functions.go:224-238`). Invalid email addresses and other PII can
therefore enter logs. Log field, rule, and request context, but never the
submitted value.

### Establish a progressive-enhancement failure baseline

The body starts hidden and becomes visible only after module JavaScript runs
(`core/ui/widget/base.gohtml:158-165,546-548`). CSS preload activation also
depends on JavaScript, and shared forms have no normal action or method
(`core/ui/widget/form.gohtml:3-11`). A blocked or failed script leaves even
sign-in blank.

Recommendation: never hide the body, add a stylesheet fallback, and give
authentication and other critical forms normal POST behavior before layering
HTMX. Full no-JavaScript support for every advanced screen is not required.

## P3: Opportunistic cleanup

### Reuse upload invariants already enforced by `S3FileSystem`

Upload handlers call `util/fileutil.EnsureFileDoesNotExist` before preparation,
while preparation already checks conflicts in
`model/tenant/filesystem/s3_file_system.go:285-347`. Make preparation APIs the
single authority after direct model tests exist.

### Move passkey rename persistence into `PasskeyService`

`action/auth/rename_passkey_cmd.go:69-79` validates and loads through the
service, then updates Ent directly. `RenameOwnCredential` can keep ownership,
normalization, and persistence together.

### Centralize temporary upload expiry

The same 15-minute lifetime appears in
`action/openfile/upload_files_cmd.go:141-153`,
`model/main/temporaryfile/upload_from_url_service.go:119-135`,
`scheduler/process_files.go:33,531-538`, and
`model/tenant/filesystem/s3_file_system.go:67,2104-2110`. Put one
`DefaultExpiresAt(now)` function in the existing temporary-file model package.

### Hide owner-only controls from non-owners

The command correctly authorizes creation, but non-owners still see Create User
controls (`action/managetenantusers/manage_users_of_tenant_page.go:39-52`,
`action/managetenantusers/user_list_partial.go:40-55`). Hide controls using the
same policy for clearer rendering; retain command authorization.

### Complete small browser cleanup items

**File handler.** Rejections become unhandled promises at
`core/ui/widget/base.gohtml:467-542`. Catch once and show the existing snackbar.

**Wake lock.** Listeners accumulate at `core/ui/widget/base.gohtml:103-153`.
Register one stable visibility listener.

**Search.** Inputs rely on placeholders at
`core/ui/widget/search.gohtml:32-51`. Add persistent accessible labels.

**Image preview.** Images have no alt text at
`core/ui/widget/file_preview.gohtml:7-11`. Add meaningful or intentionally
empty `alt`.

**Progress.** Indeterminate progress announces zero percent at
`core/ui/widget/progress_indicator.gohtml:9-16,33-35`. Omit `aria-valuenow` and
translate the label.

### Extend the browser test matrix selectively

Playwright currently runs Chromium only, with `retries: 0` and a trace setting
that activates only on retry (`playwright.config.ts:10-24`). Add Firefox and
WebKit smoke projects and use `retain-on-failure`. Prioritize the specific P0
and P1 scenarios in this review rather than building a broad new framework.

## Domain boundary guidance

The core domain is metadata-driven document filing and retrieval. The highest
model investment belongs in:

- Document Library: File lifecycle, versions, Inbox, folder tree, trash, and
  metadata assignments
- Classification: Tags, Document Types, Attributes, and typed properties

Supporting or generic areas should use narrower services and transaction
scripts:

- Identity, sessions, tenant membership, and Space access
- Ingestion, storage, OCR, preview conversion, and WebDAV intake
- Mail, configuration, provisioning, and migrations

HTMX is a delivery mechanism, not a bounded context. Browser events such as
`FileUpdated` are refresh contracts, not domain events.

## Boundaries to preserve

Do not move all action or scheduler code into models. Keep these concerns
outside:

- HTTP binding, body limits, multipart handling, and trust-boundary validation
- authorization context construction and HTMX redirects in the router
- page composition, query shaping, sorting, filtering, pagination, and empty states
- HTMX targets, swaps, URL headers, snackbars, and browser events
- Tika, Gotenberg, S3, SMTP, WebDAV protocol, and worker orchestration
- optimized Ent read queries used only to render a screen

Move business decisions, invariants, and lifecycle transitions into existing
model APIs. Continue using transaction scripts for simple CRUD and
infrastructure work. The current code does not justify repository-per-table
wrappers, CQRS, event sourcing, microservices, or a client-side state store.

## Suggested implementation order

1. Block active inline content and add the stored-XSS regression test.
2. Remove or lock down `X-Query-Endpoint`, including the setup-session test.
3. Enforce same-Space metadata relations and version-aware OCR writes.
4. Buffer write responses until commit and repair session/HTMX redirect paths.
5. Disable sensitive history snapshots and fix preview/form/filter races.
6. Make tenant User projection recovery idempotent.
7. Move File, filename, property, and Tag invariants into existing models.
8. Make morphing opt-in and establish action response contracts.
9. Address accessibility, browser-resource cleanup, and lower-risk model state
   machines while touching each flow.
10. Perform P3 cleanup only after the higher-risk behavior has focused tests.

## Open domain questions

1. Does "Mark as done" only leave Inbox, or must all required Document Type
   Attributes be complete?
2. Are cycles legal in Tag group and super/sub-tag relationships?
3. Are protected Document Types and Attributes immutable, undeletable, or only
   hidden from normal UI actions? The flags are currently dormant, so no model
   guard should be added until this is defined.
4. What exact lifecycle does `is_owning_tenant` imply when an Account belongs
   to more than one organization?
