# Command/query UI audit

Date: 2026-10-02. Initial references below identify the pre-refactoring source.
Execution and final findings: [plan/record](../plans/20261002_command_query_ui_plan.md).
Policy: [AGENTS.md](../../AGENTS.md#command--query-ui-flow).

## Assessment and method

**Mixed alignment.** Existing partials and capability events provide most of the
needed infrastructure, but mutation responses still own presentation in several
important journeys. No percentage is inferred from search counts.

Inspected registration, request/context binding, automatic/manual transactions,
response buffering, rendering/feedback, HTMX inheritance and navigation scripts,
and controls/handlers/subscriptions across every registered action capability.
Traced creation/edit/deletion of Spaces, tags, fields, document types and attributes;
file rename/move/delete/restore, Inbox filing/completion/upload, list preferences,
metadata assignment, credentials, account and system settings, and staged-upload
Space selection. Browser execution and failure checks are recorded separately.

## Capability assessment (initial)

| Capability | Assessment | Traced chain / gap |
| --- | --- | --- |
| Browse/files | Mixed | Rename emits `FileUpdated`; lists listen. Move/make-directory render `ListDirPartial`; metadata/type assignment and view preferences use `QueryHeader`. The folder chooser creation prototype is unregistered; the active move form creates its destination inline. |
| Inbox | Not aligned | Mark-done control sends `X-Query-Endpoint` to `InboxPage`; upload renders a new Inbox view; GET pages consume `upload_token`. Filing emits `FileMoved`, but selection/filters are reset. |
| Trash | Mixed | Restore emits `FileRestored` and list listens; selected-record cleanup and swap-none controls require review. |
| Tagging | Not aligned | Assign/unassign/create-and-assign return assignment HTML; edit chooser has no refresh subscription. Both subtag assignment and unassignment return a row. Toggle already returns only feedback/events. |
| Manage tags | Mixed | CRUD notifications refresh `tagList`; read-only expand/collapse is a view query despite its `Cmd` name. |
| Fields/properties | Aligned at capability level | CRUD emits property events, `propertyList` POST query listens including empty state. Shared commit barrier and form-target gaps still apply. |
| Document types | Mixed | CRUD/attribute events exist; list reload loses selected ID; detail heading has no rename listener and selected deletion needs query-owned cleanup. |
| Spaces | Mixed | CRUD events reach `spaceCards`; navigation rail/dashboard context needs independent invalidation. |
| Tenant users | Aligned at capability level | Create/delete feedback/events reach `userListPartial`; model role checks and Ent privacy govern writes/reads. Shared barrier applies. Mail-sending creation must not be exercised against real users. |
| Space users | Mixed | Assignment events reach `usersOfSpaceListPartial`; custom form still depends on caller replacement target. |
| Dashboard/account/settings | Mixed | Account/system islands listen; credential issuance also queries/renders business overview OOB. Distinct credential notifications avoid unrelated account reloads. |
| Auth | Intentional exception / mixed | Sign-in/out and recovery navigate; passkey JSON adapter emits account notification. Password/account edits use events. Standalone completion and manual commit paths require checking. |
| Admin | Mixed | Passphrase/upload-limit commands notify system island; initialization/unlock/maintenance are explicit transitions. |
| Open/import files | Not aligned | Staging is a command, but Space selection links to a mutating Inbox GET. Must consume through an authorized command. |
| Download/static/common | Intentional exception | Download streams and static pages are queries; shared form/move helpers must follow command policy. No document share-link issuance capability exists in this checkout; one-time MCP/WebDAV tokens and passkey recovery codes are the equivalent result flows. |

## Infrastructure versus adoption

### Infrastructure gaps (priority 0)

1. `server/router.go:187-193,324-340,427-442`: commit buffering is opt-in;
   ordinary commands can write success headers/body before either commit.
   `wrapManualTx:447-509` closes authorization snapshots before dispatch; each
   manual command must finish its own write transactions before success.
2. `server/router.go:213-293`: legacy dispatcher rewrites the request body and
   directly calls another handler in the command's transaction. Actual consumers
   include `action/inbox/file_metadata_partial.go:159-170`,
   `action/browse/list_dir_partial.go:688-701`, and
   `action/browse/file_attributes_partial.go:231-239`; it is not dead code.
3. `ui/renderer.go:30-31` appends queued renderables on every invocation;
   `wrapCommand:257-261` avoids a second render only when bytes were written.
   Explicit exceptional responses and error rendering need single-consumption feedback.
4. `action/util/form_helper.go:96-98,115-130` recommends legacy query headers and
   chooses replacement swaps from caller targets. A missing target blocks a
   request even with swap none. Custom forms repeat this pattern.
5. Request payload decoding is **runtime generic reflection**, not generated, in
   `action/util/functions.go:139-265` (`FormDataX`) and `:277-355` (`State`).
   Both command and POST query payloads must be tested. No generated command/query
   decoder registry was found; Ent/enum generation is unrelated.

### Incomplete adoption (priority 1)

- `action/tagging/assign_tag_cmd.go:65-73`,
  `create_and_assign_tag_cmd.go:85-122`, `unassign_tag_cmd.go:62-78`, and
  `assign_sub_tag_cmd.go:58-66`: notification beside replacement HTML.
  `edit_assigned_tags_partial.go:158-208` lacks the required listener.
- `action/browse/move_file_cmd.go:80-101`: presentation query after mutation.
  `set_file_property_cmd.go:91-98`: business OOB date suggestions.
- `action/inbox/inbox_page.go:121-228`: query consumes staged uploads using fresh
  write transactions. `action/openfile/select_space_page.go:111-122` initiates it
  through GET. `action/inbox/upload_file_cmd.go:93-108` renders after ingestion.
- `action/dashboard/create_mcp_credential_cmd.go:83-120` and the matching WebDAV
  implementation: one-time secret exception unnecessarily includes a queried
  business overview. Keep only the secret; notify the list with destination state.
- `action/documenttype/document_types_list_partial.go:43-53` resets selection;
  `details_partial.go:62-74` has no heading invalidation.
- `action/browse/select_dir_make_dir_cmd.go:69-79`: unfinished create/select
  prototype; registration and reference inspection established it has no consumers.
  It can be removed instead of inventing a new nested interaction.

### View state and navigation (priority 1)

- `core/ui/widget/htmx_attrs.go:158-164` morphs explicit GET attributes, but raw
  boosted `href` links use the body default (`base.gohtml:160-164`), which does not
  specify a morph swap. Rail restoration occurs after processing/swapping
  (`ui/uix/web/assets/navigation_rail.js:198-202`), potentially after collapse.
- `action/util/functions.go:300-326` uses current URL for POST state. GET navigation
  can drop those parameters while controls remain visible. Refresh requests must
  include the controls or retain the URL state.
- Inbox parent listens to move/delete while child lists listen to other file
  changes; keep those responsibilities distinct when adding completion events.

## Priorities

1. Default commit barrier, migrate live legacy consumers, then delete dispatcher.
2. Remove command business rendering, add missing islands/result listeners, make
   Inbox pages genuinely read-only, preserve query context/state.
3. Narrow rail morph/state rule; inspect actual rendered navigation and transient sizes.
4. Focused server/browser checks and second complete-chain audit; document actual
   exceptions and blockers rather than assuming event emission means compliance.

## Second source audit — 2026-10-03

Re-read the changed handlers and their controls, events, listeners and queries,
including empty roots and helpers. Verification evidence and limitations live only
in the [execution record](../plans/20261002_command_query_ui_plan.md#execution--verification-record).
The source migration and subsequent requested defect/browser follow-up are delivered.
The execution record distinguishes actual control flows from synthetic state checks
and the remaining narrow coverage limitations.

| Capability | Final source assessment | Command → event → query ownership |
| --- | --- | --- |
| Infrastructure | Aligned | `Router.RegisterAction` buffers ordinary writes; manual storage responses are also buffered. `wrapCommand` renders feedback only. `Renderer` takes the feedback queue once. No live `QueryHeader` or client-selected query dispatcher remains. |
| Browse | Aligned | Make-directory → `directoryCreated` → `fileList`; preferences/move/delete/merge → one parent list query with selection; metadata → file-list and metadata queries. Renaming refreshes list and detail. Navigation guards and empty `#details` prevent outgoing listeners from applying to a different capability. |
| Inbox | Aligned | Complete → `inboxChanged`; move/delete/merge → parent `InboxPage` query; upload → file-list query. Query chooses the next filtered item only when the old selection leaves the queue. GET and POST rendering no longer consume staged uploads. |
| Open/import | Intentional exception | `ConsumeUploadsCmd` authorizes explicit public tenant/Space route IDs, finishes conversion, then navigates. URL and native shared-file staging keep their context-selection responses. |
| Trash | Aligned | Restore returns `fileRestored` plus feedback; `TrashListPartial` queries the full list/detail region, retains an unaffected selection, and clears a restored selection/URL. |
| Tagging / manage tags | Aligned | All assignments and CRUD return Tag notifications/feedback. Assignment and subtag chooser roots reload themselves, including empty states. The read-only group-toggle remains a view query. The unused create-directory chooser prototype was removed. |
| Fields | Aligned | Property CRUD and file-property mutations retain their event/query chains. Date suggestions are rendered by the metadata query. Filter fields have stable IDs. |
| Document types | Aligned | Type CRUD refreshes one list/detail query that recovers selection from current URL and clears deleted selections; attribute changes refresh only the attribute island. |
| Spaces | Aligned | CRUD refreshes Space cards/dashboard cards and the separate rail. Rail query uses the actual current page, retaining tenant/Space context. |
| Tenant / Space users | Aligned | Existing create/delete/assignment events reach their read-only list queries. Custom assignment forms no longer require a replacement target. Existing model/Ent authorization is retained. |
| Credentials | Intentional exception | Create renders only the one-time secret and emits a capability event carrying public destination IDs. Separate list POST maps that destination and retains the status filter. Edit/revoke are ordinary feedback/event commands. Event-specific evaluated values are guarded during ordinary submissions. |
| Account / auth | Aligned with exceptions | Account/password/passkey changes notify account islands; auth and language transitions navigate/reload. WebAuthn JSON and the one-time recovery-code adapter retain their protocol results. |
| Admin / settings | Aligned with exceptions | System events refresh system cards; organization settings query targets its own region. Initialization/unlock/native maintenance responses retain explicit transition behavior. |
| Download / static / shared composition | Intentional exception | Streaming downloads and static pages remain queries. FormHelper submits ordinary commands with swap-none; command forms no longer depend on list/detail targets. |

### Specific second-pass corrections

- Reused tag-list queries now recreate their toolbar target instead of inheriting
  an empty target after the first refresh.
- File-version merging emits one `fileVersionMerged` notification instead of
  invalidating both parent and children through three overlapping file events.
  Version content is refreshed by its detail owner.
- Credential `hx-vals` uses a JavaScript object with a guarded event-dependent
  field; the initial bare expression was invalid HTMX syntax and was fixed after
  browser reproduction.
- `CloseDetailsHeader` preserves URL state; empty detail responses replace the
  root and retain a harmless trigger so HTMX removes old body subscriptions.
- The shared side-sheet script recognizes an already-initialized retained node;
  morph updates server content without reopening a user-closed sheet.
- List and detail refreshes ignore late notifications after leaving their route
  context. The follow-up browser run verified Browse-to-Inbox navigation and the
  settled Inbox listener without stale Browse requests.
- Existing MCP design guidance and navigation reference pointers now defer to
  AGENTS.md. HTTP request decoding remains runtime generic decoding; no generated
  decoder registration or DB schema generation was required.

### Follow-up outcome

Both independently reproduced domain defects were fixed in the existing model/
repository methods. The repeat audit also corrected the Manage Tags Delete swap
and retry-preview target omissions. Real command/query browser checks, repeated
chooser interactions, focus/caret, nonzero scroll, URL state and open/closed sheets
passed in the isolated follow-up. See the single execution record for commands,
results, the scoped limitations and cleanup; these defects are not architecture
exceptions.
