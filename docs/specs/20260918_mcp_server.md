# MCP server and shared HTMX commands

Date: 2026-09-18  
Status: proposed  
Scope: supporting architecture; slice 01 implementation present, verification pending.

The [feature specification](../features/2026.09-mcp_server/spec.md) defines the revised first
release: upload, classify with Tags/properties/document types, and file documents. Notes and
standalone rename are deferred. Its recorded assumptions and contracts take precedence over
the original exploratory recommendations.

## Recommendation

Add MCP to the existing SimpleDMS Go process using the official
[`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).
Expose a Streamable HTTP endpoint at `/mcp`. Make the existing model methods the
shared execution boundary for browser commands and MCP tools.

The smallest useful design has:

1. Two thin adapters: existing `action/*` handlers and a new `server/mcp` package.
2. Existing model operations, with remaining business decisions extracted from handlers as needed.
3. Shared authorization/context and transaction execution, extracted from the existing router.
4. Explicit, typed MCP tool registration using the SDK.

This requires neither a command bus nor a second service. Reuse existing methods directly when
they already represent a complete operation; do not create a forwarding command class for each
method. The small amount of form/JSON and HTML/MCP adaptation is useful transport-specific code.

Related documents:

- [Useful tools and current command mappings](20260918_mcp_tool_catalog.md).
- [Architecture decision record](../adrs/2026.09-shared_commands_for_htmx_and_mcp.md).
- [Implementation plan](../features/2026.09-mcp_server/plan.md).
- [Source review and execution record](#source-review-and-execution-record).

## What exists today

### HTTP actions are broader than business commands

[`action.Actions`](../../action/actions.go) composes the feature action groups.
[`Router.RegisterActions` and `RegisterAction`](../../server/router.go) discover `Actionable`
fields and register their handlers and optional forms. This collection includes pages, partials,
dialogs, browser-state changes, and mutations. It is not an appropriate automatic MCP allowlist.

`Actionable.Handler` receives `httpx.ResponseWriter`, `*httpx.Request`, and `ctxx.Context`.
[`FormData`](../../action/util/functions.go) decodes and validates form submissions. Browser
tenant/Space selection comes from route parameters or `HX-Current-URL` in `Router.context`.
`wrapCommand` renders queued widgets or dispatches a client-specified `X-Query-Endpoint` partial.
These are browser concerns, not a reusable tool execution interface.

### Some operations already have the right boundary

- [`DocumentNotes`](../../model/tenant/file/document_notes.go) owns note authorization and
  lifecycle. [`DocumentNoteCmd`](../../action/browse/document_note_cmd.go) mostly binds input,
  calls that model, and creates browser feedback.
- [`TagService`](../../model/tenant/tagging/tag_service.go) and
  [`PropertyService`](../../model/tenant/property/property_service.go) are already called by
  their corresponding management commands.
- [`RenameFileCmd`](../../action/browse/rename_file_cmd.go) loads a file and calls
  `infra.FileSystem().Rename`. The same applies to several filesystem operations.
- [`FileVersionFromInboxService`](../../model/tenant/file/file_version_from_inbox_service.go)
  implements the version merge; the command adds form-specific coordination and feedback.

Other operations still make business decisions in handlers: marking an Inbox file done,
trashing/restoring files, selecting a document type, and setting file property values. Extract
these decisions into the relevant models before exposing them through a second transport.

### Reads need extraction too

The most useful agent operations are often queries rather than `*Cmd` types. Browse search is
in [`ListDirFileQueryService`](../../action/browse/list_dir_file_query_service.go), which still
depends on partial-state types and a property-filter callback. Inbox search is in
[`FilesListPartial`](../../action/inbox/files_list_partial.go).

Extract reusable query inputs and data results from these implementations. Preserve the existing
FTS, ordering, resolved-Tag, property-filter, and Inbox/filed semantics instead of implementing a
second search algorithm. The MCP adapter must not parse a rendered partial to recover data.

### Transaction and identity details matter

- `Router.wrapTx` opens main and tenant transactions, constructs `ctxx` contexts, catches legacy
  panics, and commits or rolls back. Handlers can currently write a response before commit.
- Uploads and archive extraction use `wrapManualTx` and
  [`txx` helpers](../../util/txx/main_tenant_tx_helpers.go) to separate storage I/O from short
  transactions. A transaction must not span an MCP connection or a whole upload stream.
- [`Config.IsReadOnly`](../../util/actionx/common.go) also returns true for manually managed
  operations that write. It cannot determine an MCP tool's `readOnlyHint` or permission scope.
- Ent privacy depends on the correctly constructed tenant/Space context. In particular,
  [`SpaceMixin`](../../db/enttenant/schema/space_mixin.go) filters by the current Space.
- Files and notes have public identifiers. Tags, properties, document types, and document-type
  attributes currently use internal numeric IDs; see [public identifiers](#public-identifiers).
- [`WebDAV`](../../server/webdav/handler.go) is a useful precedent for an in-process protocol
  adapter with non-browser authentication. Its credentials deliberately have different powers
  from the proposed MCP credentials.

## Architecture

```text
HTMX request                               MCP tools/call
    |                                            |
browser Session + form binding             bearer authentication + JSON schema
    |                                            |
action/<feature>/*Cmd.Handler              server/mcp typed tool handler
    |                                            |
    +----- shared context/transaction execution --+
                         |
                existing model operation
                or extracted model/query method
                         |
                Ent / filesystem / storage
                         |
               committed data result or error
                    /                 \
      snackbar + HX events        structured MCP result
      partial reads return HTML   tool/protocol error mapping
```

### Responsibilities and package placement

| Location | Responsibility |
| --- | --- |
| `action/<feature>/` | Browser forms, presentation state, widgets, events, and redirects. |
| `server/mcp/` (new) | SDK endpoint, typed tool schemas, registration, and MCP result mapping. |
| `common/execution/` (proposed) | Concrete executor for authorized context and transaction lifetime. |
| `model/tenant/file/` | File lifecycle, notes, and file-specific rules currently left in handlers. |
| `model/tenant/filesystem/` | Filesystem operations and orchestration that needs the filesystem. |
| Existing metadata model packages | Tag, property, and document-type behavior. |
| `model/main/mcpcredential/` (proposed) | Initial MCP credential lifecycle and authentication. |

These locations describe responsibilities, not a requirement to create empty packages or a class
per tool. For example, `DocumentNotes.Create` already provides the shared operation.

Keep model dependencies directed inward. A model must not import `action`, `server/mcp`, widgets,
or `common.Infra`. Filesystem already imports the file model, so put orchestration requiring both
in the filesystem package, not in a file-model type that imports filesystem back again. Query
extraction needing `FileTree` can live there as well; retain the existing Ent query helpers.

Construct and inject the executor, model dependencies, and MCP server at startup, following the
existing `Infra`/constructor pattern. Keep request actors and transactions out of shared server
fields. Register a small, explicit list with `mcp.AddTool`; do not build a second reflection-based
registry or expose a generic `execute_action(endpoint, data)` tool.

### Shared operation shape

Use ordinary Go methods that accept an authorized `ctxx.Context` or `*ctxx.SpaceContext`, typed
values, and return data plus an error. Existing positional parameters are fine for small methods.
Introduce a request/result struct only when the operation benefits from it.

For example, the existing Tag service operation is:

```go
func (qq *TagService) AssignToFile(
	ctx ctxx.Context, fileID, tagID, spaceID int64,
) (*enttenant.Tag, error)
```

The two adapters then do the following:

| Step | HTMX | MCP |
| --- | --- | --- |
| Decode | Existing assignment form data | `file_id` and `tag_id` JSON |
| Resolve | Scoped file/Tag lookup | Public IDs resolved within the credential's Space |
| Execute | Shared desired-assignment model method | The same method |
| Project | Data needed for feedback/partial | Explicit public assignment data |
| Complete | Commit, then existing browser feedback | Commit, then typed SDK output |

Strengthen this existing model operation with the specification's desired-state semantics and
scoped validation; the current repository call inserts an assignment without checking for an
existing one. Its numeric parameters stay internal. Map Ent objects to explicit response types
inside the transaction, while all needed data is available. Never serialize an Ent entity directly
or perform lazy edge queries after its transaction closes. Add error-returning scoped lookups
alongside panic-based `GetX` as needed; a few coordination lines do not require a new service layer.

### Context, authorization, and transactions

Keep `Router.RegisterAction`, `wrapTx`, and `wrapManualTx` as the browser entry points. Extract
their reusable execution machinery and have these wrappers delegate to it; MCP calls the same
machinery. This is an extraction of the existing guarantees, not an unwrapped route around them.

The shared executor receives the authenticated actor, explicit tenant/Space scope, transaction
mode, and operation callback. Its responsibilities are:

1. Resolve the account and scope, check active tenant membership using
   [`TenantAccessService`](../../model/main/tenantaccess/tenant_access_service.go), and load the
   current tenant user and permitted Space. Preserve tenant-owner implicit Space access.
2. Construct `VisitorContext -> MainContext -> TenantContext -> SpaceContext` using the request
   context. Locale/timezone may use account settings; HTMX flags are false for MCP.
3. Use read-only database connections for queries and the required write connection for
   mutations. Transaction mode is server configuration, never a tool argument.
4. Invoke the model operation with Ent privacy active. Operation-specific permissions remain
   in the model, including note authorship/ownership checks.
5. Roll back on errors, cancellation before completion, and recovered legacy panics. Convert
   expected failures to returned errors; log unexpected failures without sending internals.
6. Commit before releasing successful results or browser success events.

Browser authentication and redirects remain in the HTTP adapter; MCP authentication failure
never redirects to sign-in. MCP credential restrictions are additional to current account and
Space access, including for tenant owners. Revalidate them per call, not just when a client first
connects. Respect application lock/maintenance and unavailable-tenant states at this boundary.
Enforce the credential's read/write mode when executing a tool, independently of annotations or
tool discovery. A single static catalog is sufficient; never capture one client's actor in a
shared SDK handler or treat a legacy MCP session identifier as authentication.

Two existing implementation constraints need explicit treatment:

- `httpx.ResponseWriter` does not buffer writes. For migrated browser mutations, collect the
  result/snackbar/events until commit, or buffer the bounded browser response while preserving
  current partial composition. Leave streaming paths on their dedicated lifecycle.
- Main and tenant SQLite transactions commit separately; there is no distributed atomic commit.
  Prefer a read-only main authorization transaction and one tenant mutation transaction for
  ordinary document commands. Preserve stronger locking/revalidation where an existing workflow
  needs it. Do not represent a cross-database change as automatically atomic.

For manual storage workflows, reuse `txx` and the current prepare/upload/finalize lifecycle. An
adapter must not wrap these operations in another encompassing write transaction.

### Browser behavior

Retain `FormHelper`, existing `*Dialog`/`*Partial` types, snackbar widgets, and `ui/uix/event`.
Form decoration and `validate` tags are not an MCP schema. Bind each transport's representation
to the same model inputs, with business validation in the model.

Current handlers are mixed: rename and notes already use snackbar/events, while commands such
as move, make-directory, and Tag assignment render fragments. Preserve their browser contract
during extraction. Where converted to separated commands, return snackbar feedback and
`HX-Trigger` events and let existing partial endpoints render refreshed HTML. Move UI composition
only with its consumers, rather than changing all command responses as an MCP prerequisite.

`HX-Current-URL`, `HX-Target`, `X-Query-Endpoint`, selected files, open dialogs, and expanded
groups stay in the browser adapter. An MCP call does not send HTMX headers or trigger an event in
an unrelated open browser tab; subsequent browser reads see committed changes normally.

## MCP boundary

### Transport and protocol

Use `mcp.NewStreamableHTTPHandler` in the existing HTTP mux. Prefer stateless request execution
and ordinary JSON responses for the initial bounded tools. Let the SDK implement protocol
discovery/version handling, JSON-RPC, schemas, and legacy initialization compatibility.

As of this review, the official SDK documents support for MCP `2026-07-28` starting with v1.7.0,
including older protocol versions. The 2026 revision removed protocol-level HTTP sessions and
the standalone GET stream. Pin a compatible released SDK when implementing; do not implement a
custom handshake or assume every supported client uses the same revision.

Mount the SDK handler directly under MCP-specific middleware. Do not wrap the whole protocol
handler in a database transaction or feed it through `wrapCommand`. No database transaction is
needed just to list static tool definitions. Let the SDK dispatch a tool, then execute its
authorized operation through the shared executor.

Validate a supplied `Origin` against allowed origins, retain trusted-proxy handling, and require
HTTPS outside local development. Keep request/result size limits. For any later MCP resources,
use the same access checks and private cache policy for account-dependent data. Start with tools;
resource templates and prompts are unnecessary for the first document workflow.

### Initial authentication recommendation

For the smallest self-hosted integration, use a separate, revocable, account-owned MCP bearer
credential, scoped to exactly one tenant and Space. Clients connect to `/mcp` with
`Authorization: Bearer ...`. The credential supplies scope, so initial tool arguments need no
tenant selector, Space selector, or mutable "current Space" state. `get_space` identifies it.

Follow the existing WebDAV credential lifecycle pattern: generate a high-entropy secret, show it
once, store its hash plus owner/scope/label/revocation metadata, and manage it from a full browser
Session. Offer read-only and read/write modes; read-only is the default. Token management is new
work, including an Ent schema and generated migration. Never accept existing WebDAV credentials
for MCP: their ingestion-only authority must not become document-reading authority.

This initial choice assumes a client that can configure bearer headers. A personal token is not
the standard MCP OAuth authorization flow and will not provide automatic sign-in for every MCP
client. If general remote-client interoperability is required for the first release, use an OAuth
2.1 authorization server and make SimpleDMS its resource server instead. That requires protected
resource metadata, authorization-server discovery, PKCE for public clients, resource/audience
validation, and mapping the authorized subject/grant to the same SimpleDMS actor and Space.
The SDK's auth helpers are not a complete authorization server.

The feature specification records configurable bearer headers as the initial client assumption;
planning is not blocked on OAuth. A future multi-Space grant can introduce explicit per-call
scope and `list_spaces`; it does not belong in a Space-bound first version.

### Schemas, results, and errors

- Use stable, descriptive names such as `assign_tag` and `file_inbox_document`.
- Define MCP JSON inputs explicitly and use SDK schema inference where sufficient. Add enum,
  required-field, and size constraints explicitly where Go type inference cannot express them.
- Shared model validation must still execute after schema/form validation. Handle missing values
  separately from `false`, `0`, and empty strings, especially for properties.
- Return explicit data in `structuredContent`, with the SDK's JSON text-content fallback for
  clients that consume text. Avoid localized snackbar prose as the machine-readable result.
- Map expected operation failures to `CallToolResult.IsError`, with a stable code and safe message
  in its error content. Initially map existing `e.HTTPError.StatusCode()`/`Message()` into
  `invalid_input`, `forbidden`, `not_found`, `conflict`, or `limit_exceeded`. Unexpected failures
  become `internal_error`. Do not pass arbitrary Go errors directly to the SDK, which may expose
  their messages.
- Authentication failures use HTTP 401/403. Malformed JSON-RPC, unsupported methods, and schema
  failures follow the SDK/protocol's error handling; they are distinct from business failures.
- Existing `e.HTTPError` is coupled to HTTP/snackbars. A compatibility mapper is sufficient for
  this proposal; replacing the repository's complete error model is not a prerequisite.

Set annotations explicitly according to actual effects. Queries use `readOnlyHint: true`.
Mutations use false; trashing, metadata removal, and merging must describe their effects and use
appropriate `destructiveHint`. Keep `idempotentHint` false unless the operation's semantics justify
it, and use `openWorldHint` for an eventual URL-fetching tool. Annotations are client hints, not
authorization enforcement.

An MCP/JSON-RPC request ID is not an idempotency key. Explicit metadata assignments have the
desired-state behaviour specified for the feature; uploads and filing retain their own retry
semantics. Do not claim exactly-once execution. After an uncertain write outcome, clients should
query state before retrying. Durable deduplication is a separate capability.

### Public identifiers

All external entity references use `PublicID`, including IDs inside returned objects. Keep
internal numeric IDs only for joins and model calls after a scoped lookup. A version number
qualified by a file public ID is a valid domain selector, not a database row identifier.
The catalog also identifies the existing document-scoped legacy-note selector; it must remain
explicitly distinguishable from a public ID.

These source schemas currently lack active public-ID mixins:

- [`Tag`](../../db/enttenant/schema/tag.go).
- [`Property`](../../db/enttenant/schema/property.go).
- [`DocumentType`](../../db/enttenant/schema/document_type.go).
- [`Attribute`](../../db/enttenant/schema/attribute.go).

Before publishing the corresponding tools, add immutable public IDs, backfill existing records,
and enforce uniqueness using the repository's generated migration workflow. Follow the
[additive SQLite migration guidance](../../AGENTS.md#additive-sqlite-migrations), including
separate field/index additions where required. Do not expose numeric IDs temporarily or use
mutable names as identifiers. Resolve all referenced objects inside the authorized Space.

Metadata public IDs are a first-release prerequisite, delivered with classification rather than
as a standalone database slice. Before that slice, `get_file` can return display names/values
without internal identifiers. Attribute IDs are unnecessary for inline type descriptions and
remain deferred until independently addressable attribute operations are introduced.

### Bounded reads and binary data

Proposed initial list contract: `limit` defaults to 50, maximum 100, and a nonnegative `offset`.
Return `has_more` and `next_offset`; use deterministic ordering with an ID tie-breaker. Offset
pagination matches current query code and is not a snapshot across concurrent edits. Apply limits
in the query, including when adapting currently unbounded notes/version methods.

For OCR text, use Unicode-character offsets and a bounded length, for example 12,000 characters
by default and at most 50,000. Return the file/version reference, OCR availability, and truncation
or continuation information. Missing OCR is an explicit state, not a fabricated empty document.
Existing OCR content is file-level; this does not promise historical-version OCR.

Binary upload is in the first release. The [feature upload contract][feature-upload] chooses a
bounded base64 `upload_file` tool reusing the existing storage pipeline. This replaces the original
suggestion to defer all binary transfers. It trades a documented size/memory ceiling for a complete
native MCP path. Large-file streaming and binary downloads remain extensions; a client-local path
is not readable by the server, and a browser-cookie URL is not an MCP download API.

URL import also needs orchestration: the current command stages an account-temporary upload and
redirects to Space selection. A tool must target the bound Space, finalize persistence, and return
the durable file ID. Preserve URL validation and storage integrity, quota, cleanup, source, and
finalization authorization from the existing pipeline. Choose an explicit source classification
for any new MCP binary-ingestion path rather than accidentally recording a browser upload.

## Suggested scope and extraction order

The [plan](../features/2026.09-mcp_server/plan.md) delivers one first release across four observable
journeys: connect/inspect, upload, classify, and file. Uploads and classification are core scope;
notes and standalone rename are later candidates. Metadata IDs arrive in their consuming slice.
Read-only credentials can execute only read tools.

Before calling an implemented slice complete, demonstrate:

- Equivalent model effects through HTMX and MCP, including browser events/partial refreshes.
- Rejection of cross-scope references, revoked credentials, and current permission loss.
- Rollback and absence of success output on model or commit failure.
- Typed, bounded tool results and correct protocol/business error separation.

Use the existing action/model integration-test patterns and the SDK's in-memory client/server
transport for focused checks. Test actual Streamable HTTP authentication separately. These are
future acceptance expectations, not checks performed for this documentation change.

## Source review and execution record

2026-09-18, documentation-only work:

- Inspected action composition, representative commands/queries, router wrappers, `ctxx`, Ent
  source schemas/privacy, transaction helpers, filesystem, notes, duplicate detection, metadata
  models, WebDAV, and upload paths. Source links above and in the catalog identify the findings.
- Used `gopls` symbol navigation and these API-inspection commands:
  - `go doc github.com/simpledms/simpledms/model/tenant/file`
  - `go doc github.com/simpledms/simpledms/model/tenant/filesystem.S3FileSystem`
  - `go doc github.com/simpledms/simpledms/model/tenant/tagging.TagService`
- Consulted the local HTMX `HX-Trigger` reference and the upstream references below.
- Produced this proposal, the tool catalog, and a proposed ADR. No runtime code, dependencies,
  schemas, generated files, or migrations were changed. No build, tests, browser journey, or MCP
  interoperability checks were run. Implementation effort and client compatibility remain
  unverified; the priority labels are recommendations, not completion statuses.

Planning follow-up, 2026-09-18:

- No MCP feature `spec.md` existed. Created the canonical feature specification from this proposal
  and the user's revised priorities, then linked a four-slice plan and durable-rule index.
- Inspected only the relevant credential UI/tests, upload/finalization paths and storage rules,
  metadata/query code, public-ID/data-migration mechanisms, filing regressions, and browser helpers.
- Recorded bearer-client, existing-definition, and bounded-upload assumptions. Used direct
  planning; no planner/architect agents were needed. No implementation or verification was run.
- Repository inspection used `git status --short`,
  `git status --short --untracked-files=all -- docs`, and `gopls` symbol navigation, with
  source/document reads. The scoped status showed only the MCP documentation files. Existing
  changes in i18n and URL-import code were not part of this work.
- Future slice checklists are unperformed. Record actual implementation commands/results here
  when they occur rather than copying an execution log into every slice.

Slice 01 implementation, 2026-09-18:

- Added the owned credential schema/model and generated main migration
  `20260918225936_mcp_credentials`, plus generated Ent bindings. The migration creates only the
  MCP table/indexes. Credential references cascade on physical account/tenant deletion.
- Pinned the official SDK at v1.8.0. `/mcp` uses stateless Streamable HTTP, JSON responses, current
  bearer authorization on each request and tool execution, same-origin checks when Origin is
  supplied, trusted-proxy HTTPS handling, and the specified request-size bound. The locked app's
  maintenance listener returns 503 for this endpoint.
- Shared `execution.ScopeResolver` and `file.InboxQuery` with browser code; added scoped
  `FileReader` lookup and character-window projection. Protocol reads own short read-only
  transactions in `CredentialService.Read`; the existing browser wrappers retain ownership.
- Added Account → MCP credentials, create/list/revoke, translations, and one-time token display.
  New credential writes opt into commit-buffered responses. A small native `SelectField` supplies
  keyboard-accessible dynamic Space options without changing existing list-radio consumers.
- Added SDK/HTTP action tests for scope isolation, pagination/OCR windows, errors, revocation,
  membership expiry, read-only execution, commit failure, document states, and form handling;
  added fresh/populated migration tests and desktop/mobile Playwright `@connect` coverage.
- Dependency/generation commands executed successfully:
  - `go list -m -versions github.com/modelcontextprotocol/go-sdk`
  - `go get github.com/modelcontextprotocol/go-sdk@v1.8.0`
  - `go get github.com/modelcontextprotocol/go-sdk/mcp@v1.8.0`
  - `go generate .` (Ent generation, repeated after adding delete cascades).
  - `go run ./cmd/migrate mcp_credentials` initially also generated a pre-existing WebDAV unique
    index drift. Regenerated using a temporary Atlas `WithDiffHook` limited to `AddTable` for
    `mcp_credentials`, with generator-written checksums. The provisional files were replaced by
    generation, not hand-edited into a different migration. No WebDAV index change ships.
  - `go run /tmp/opencode/generate_mcp_migration.go` performed that scoped generation; the
    temporary helper is not an application dependency or repository artifact.
  - `go generate ./i18n`, repeated after source translations; existing unrelated missing-entry
    diagnostics remain. Output from the last run is `/tmp/opencode/mcp-i18n-generation.log`.
  - `gofmt -w` on the touched application, schema, and test Go files.
- **Verification boundary:** no tests, build, vet, Playwright, live server/client session, or
  manual workflow were run. No implementation-blocking test was needed. Source/API inspection,
  required generation, formatting, and diff inspection are not verification of the slice.
  Implementation checklist items are complete; verification items and the plan checkbox remain
  pending for the requested review phase. No unresolved document conflict or implementation
  blocker is known.

Slice 01 token-dialog fix, 2026-09-28:

- A user reported that the one-time token was not displayed. The credential handler returned the
  token dialog, but `Router.wrapCommand` performed an empty second render. On a commit-buffered
  response, that set `HX-Reswap: none` before headers were sent, so HTMX discarded the dialog.
- `go test ./server -run '^TestMCPConnectionCreateResponseShowsTokenDialog$' -count=1` first
  failed on that header. After skipping the empty render when the handler already wrote a body,
  the same focused test passed; it checks both the swap header and token dialog content.
- `gofmt -w server/router.go server/mcp_connection_test.go` completed. No broader Go suite,
  Playwright/browser check, build, or manual client journey was performed. Slice 01 verification
  checkboxes and the plan checkbox remain pending.

Slice 01 document URLs, 2026-09-28:

- `list_inbox` document summaries and `get_file` metadata now expose the canonical browser `url`.
  The URL uses the configured public origin when available and otherwise the active MCP request
  origin. Inbox listing eager-loads parent public IDs so URLs contain no internal identifiers.
- `go test ./server -run '^TestMCPConnectionScopesReadsAndRevocation$' -count=1` passed, including
  URL assertions for both tools. Broader slice verification remains pending.

Slice 02 implementation, 2026-09-29:

- Added the bounded `upload_file` tool with padded base64 input, a 10 MiB decoded ceiling further
  restricted by configured upload limits, the existing 16 MiB transport ceiling, exact-basename
  validation, and a committed result containing public file ID, browser URL, filename, size, and
  Inbox state. Read-only credentials fail before ingestion.
- Extracted Browse/Inbox prepare plus shared new-file/version expected-byte upload, finalization,
  MIME follow-up, and failure coordination into `filesystem.FileIngestionService`. The existing
  prepared-upload, quota, checksum, encryption, and scheduler paths remain authoritative.
  Finalization errors retain the verified temporary bytes and unfinished row for reconciliation
  because commit results can be ambiguous.
- Added an optional fresh-main authorization callback to the established main-before-tenant
  finalization helper. MCP uses it to recheck the same credential's owner, tenant, Space, writable
  mode, and revocation state before the tenant commit.
- Appended and regenerated the integer-backed `filesource.MCP` value without changing prior enum
  values. Existing source filters discover it through `filesource.Values`; UI labels display the
  untranslated protocol name.
- Added `server/mcp_upload_test.go` for bounded input, read-only denial, configured limits,
  encryption-on/off byte persistence, source/URL projection, and revocation before finalization.
  Added the desktop/mobile `@upload` Playwright journey while retaining the ordinary browser
  upload in its setup.
- Implementation commands: `go generate ./model/main/common/filesource` and `gofmt` over touched
  Go files. The implementation-blocking
  `go test ./server -run '^TestMCPUploadRejectsInvalidInputAndReadOnlyCredential$' -count=1`
  passed after correcting the version-upload caller found by its initial compile failure. No S3
  checks, browser checks, build, vet, or manual MCP client journey were run. Slice 02 verification
  items and the plan checkbox remain pending.

Slice 03 implementation, 2026-09-29:

- Added optional, immutable, application-defaulted public IDs for Tag, Property, and DocumentType.
  Generated migration `20260929123641_metadata_public_ids` contains only three additive nullable
  columns; generated migration `20260929123746_metadata_public_id_indexes` contains only their
  three unique indexes. Neither migration rebuilds, copies, or drops an existing table, and no
  attribute public ID was added. MCP projections reject empty identifiers even after ledger
  completion. The rollout mixin remains nullable only for additive SQLite compatibility and is
  explicitly scheduled for replacement by the required mixin in a future non-rebuilding schema
  tightening.
- Added three bounded tenant data migrations with independent cursors and ledger completion. They
  initialize only missing IDs in batches of 100, preserve IDs created during an upgrade, resume
  after interruption, and run both for existing and newly initialized tenants. Metadata tools are
  unavailable until all three ledger entries complete.
- Strengthened Tag assignment/unassignment as desired-state operations, rejecting Group Tags and
  preserving direct versus resolved results. Added explicit document-type set/clear operations and
  retained the browser's toggle as an adapter. Extracted typed field upsert/removal into the
  property model; MCP validates exactly one matching value, safe integers, minor money units, and
  ISO dates while browser decimal/date-presence handling stays at its adapter.
- Registered the ten classification tools: `list_tags`, `list_properties`, `list_document_types`,
  `get_document_type`, `assign_tag`, `unassign_tag`, `set_file_property`,
  `remove_file_property`, `set_document_type`, and `clear_document_type`. Definition and file
  projections contain public IDs only; `get_file` now reports document type, direct/resolved Tags,
  and typed assigned values.
- Existing Browse property handlers and document-type selection now call the shared model
  operations. Added SQLite-backed migration and MCP cross-transport test coverage plus the shared
  desktop/mobile `@classify` Playwright journey. The journey imports Invoice metadata through the
  create-Space UI, creates Number/Checkbox fields through browser controls, uploads/classifies over
  MCP, corrects values in the browser, and reads the corrections back over MCP.
- Generation/formatting commands executed: the repository's `go generate .` invocation failed
  under the active Go 1.27 toolchain with Ent's `package "context" without types` loader error.
  The pinned tool succeeded through `go tool ent generate ./db/enttenant/schema/ --target
  ./db/enttenant --feature intercept,entql,privacy,schema/snapshot,sql/versioned-migration,sql/modifier,sql/execquery
  --template ./db/enttmpl`; `go run ./cmd/migrate metadata_public_ids` and
  `go run ./cmd/migrate metadata_public_id_indexes` generated the staged migrations. The command's
  unrelated pre-existing main WebDAV index drift was excluded. `gofmt` ran over all changed Go
  files.
- Two compile-only diagnostics passed: `go test ./server ./model/tenant/... ./scheduler -run '^$'`
  and `go test ./server ./model/tenant/tenantdatamigration ./action/browse
  ./model/tenant/property ./model/tenant/documenttype ./model/tenant/tagging -run '^$'`. These did
  not execute runtime tests. No focused classification test, migration execution test, Playwright,
  build, vet, live client, or manual desktop/mobile journey was run. Verification checklist items
  and the plan checkbox remain pending.

Slice 04 implementation, 2026-09-29:

- Added `filing.FilingService` as the shared lifecycle owner for folder filing and completion
  without a move. Inbox `MoveFileCmd`, `AssignFileCmd`, and `MarkAsDoneCmd` now use it. Only live
  Inbox documents can complete; folder mode, scoped destinations, directory shape, filename,
  same-parent root filing, conflicts, and child-directory creation are validated inside the same
  tenant transaction. Existing file identity and related metadata rows are not rewritten.
- Extracted `file.FiledQuery` from Browse's filed-query behavior. Browse and MCP now share FTS,
  resolved-Tag AND filtering, document-type filtering, sorting, deterministic ordering, and filed
  Space scope. Browser property-filter state remains in the adapter and outside the first MCP
  contract.
- Registered `list_directory`, `create_directory`, `file_inbox_document`,
  `mark_inbox_file_done`, and `search_files`. Directory and file references are scoped public IDs;
  lists default to 50, cap at 100, and expose deterministic offset pagination. Filed search excludes
  Inbox documents, directories, deleted rows, and other Spaces.
- Ordinary MCP reads retain read-only tenant transactions and ordinary mutations use the
  read/write connection. Upload remains a write-annotated tool but keeps a read-only outer scope
  because its established prepare/I/O/finalize workflow owns separate fresh write transactions.
  This fixes classification and filing writes without weakening storage transaction boundaries.
- Added `server/mcp_filing_test.go` and the desktop/mobile `@file` Playwright journey. The focused
  `go test ./server -run '^TestMCP(Filing|MarkInbox)' -count=1 -v` passed, covering directory
  discovery/creation, failed-operation state preservation, folder and same-parent filing,
  classification-filtered filed search, pagination, non-folder completion, read-only denial, and
  foreign-Space rejection. The final compile-only check passed for `model/main/mcpcredential`,
  `model/main/tenant`, `model/tenant/file`, `model/tenant/filing`, `model/tenant/filesystem`,
  `action/browse`, `action/inbox`, `server`, and `server/mcp`.
- No Playwright run, broad Go suite, build, vet, concurrent-completion check, or manual MCP client
  journey was performed. Slice verification checkboxes and every plan delivery checkbox remain
  pending.

All-slice review and fixes, 2026-09-29:

- Reviewed all four slices after `2d6fd77`, expanding from filing/search into their shared browser,
  credential, ingestion, metadata, and migration callers. No applied migration SQL was changed.
- Fixed ordinary browser upload regressions: manual upload wrappers close authorization
  transactions before handlers run, so parent lookup and post-upload Inbox rendering now use
  fresh read transactions instead of the closed request transaction.
- Corrected MCP write context construction to carry the actual transaction mode. Upload retains
  its read-only outer execution scope and independent prepare/finalize writes; its revocation test
  now exercises that real lifetime. Refreshed initialized tenant fixtures for encrypted byte reads.
- Enabled the existing committed-response option for the shared browser filing, property, type,
  and direct Tag assignment commands. Protecting HTTP tests inject tenant commit failures and
  verify no success body/event escapes and no mutation persists. Filing coverage now also checks
  concurrent completion, conflicts/repeats, and rollback of a newly created child directory.
- Fixed completed data migrations reporting incomplete on subsequent calls. Added fresh and
  populated production SQL replay/backfill tests preserving grouped/composed Tags, type attributes,
  money minor units, and file assignments, with stable IDs on repeated runs. Fixed pre-existing
  feature-test setup errors (duplicate names, stale tenant identity, wrong browser page/context).
- Rejected dot/noncanonical directory names and dot filenames at the shared filesystem/filing
  boundary. Added public SDK tests for those failures and missing/null/wrong/out-of-range typed
  values while asserting prior state remains intact.
- Updated browser checks to the actual navigation, unnamed Details dialog, Info/Fields tabs,
  visible source-filter labels, download control, and debounced property saves. The tests await
  exact persisted-operation responses rather than an unrelated stale snackbar. No new UI was
  introduced.
- Final verification passed:
  - `go test ./server -run '^TestMCP' -count=1 -timeout 120s`, including production main/tenant
    migrations, encryption-on/off byte round trips, revocation, classification, filing, and browser
    commit-failure tests.
  - `go test -count=1 ./model/tenant/tenantdatamigration`; affected main-tenant, Tag, property,
    document-type, and `util/txx` packages also passed scoped tests during review.
  - `go test ./server -run '^(TestMarkAsDoneCmd|TestDocumentNotesHTTPInboxRootFilingPreservesHistory|TestDocumentNotesHTTPMissingParentRestoreAndNonFolderFiling|TestBrowseListDirPartial|TestFileVersionFromInboxCmd|TestUploadFileCmd|TestUploadFilesCmd|TestBrowserUploadFinalizationRejectsRemovedTenantAssignment)' -count=1`.
    Concurrent ordinary upload and failed-version-upload regression checks passed separately.
  - `E2E_BASE_URL=https://localhost:7202 E2E_LOGIN_EMAIL=dev+admin@simpledms.app E2E_LOGIN_PASSWORD=12345678 npm run test:e2e -- e2e/mcp_workflows.spec.ts e2e/browse_upload_filters.spec.ts`:
    **11/11 passed** (all four MCP journeys on desktop/mobile plus three Browse regressions).
    The disposable `.e2e-mcp-meta` instance ran current source; the initial inherited credential
    mismatch was resolved by explicitly selecting its test account. The user's `.testdata` was
    not modified by this review.
  - `go build ./...`; `go vet ./server ./action/browse ./action/inbox ./model/main/mcpcredential
    ./model/tenant/filesystem ./model/tenant/filing ./model/tenant/tenantdatamigration`.
  - Touched Go files were formatted; `git diff --check` passed.
- Remaining checklist evidence: no separate manual external-client journey, no full repository
  test suite, and no slow network-fault suite was run. Upload-specific cancellation/uncertain-commit
  injection and full exact-limit/quota coverage, concurrent classification/commit-error cases,
  legacy-connection restart/cancelled-read cases, and browser filing through both adapters in
  non-folder/multiple-selection states remain unverified by this review. Automated coverage does
  not complete those multi-case checklist entries. Completed items are checked in their slices;
  all four plan delivery boxes remain open.

Same-tenant cross-Space regression, 2026-09-29:

- Added `TestMCPBearerCredentialCannotCrossSpaces` in
  `server/mcp_space_isolation_test.go`. One account has access to both Spaces in one tenant;
  a B-scoped credential provides a positive control while the A-scoped credential must reject
  B's file, metadata, and destination IDs. Listings/search expose A's documents and definitions
  only, and rejected reads/writes leave both Spaces' document and directory state unchanged.
- `go test ./server -run '^TestMCPBearerCredentialCannotCrossSpaces$' -count=1 -timeout 60s`
  passed. No production code changed, and broader checks were not repeated.

Slice 05 metadata management implementation, 2026-09-29:

- Added 20 tools through existing UI model/services: nine definition CRUD tools; Tag grouping,
  composition, and atomic create-and-assign; five typed attribute operations; and template
  discovery/import. The catalog has the complete names and the feature specification owns their
  input contracts. Field type editing stays unavailable, as in the UI.
- New mutations use the existing credential write executor, require completed metadata-ID
  readiness, and resolve all public references within the bound Space. Referenced-definition
  deletions and duplicate constraints fail as business errors without clearing other metadata.
- Attribute edits/removal use the existing unique document-type/Tag or document-type/field pair;
  no attribute ID or new schema migration is needed. The decision is recorded in
  [the attribute-reference ADR](../adrs/2026.09-mcp_metadata_attribute_references.md).
- Centralized atomic Tag creation/assignment for browser and MCP, with committed browser feedback.
  Shared model validation now enforces the UI's group/composition/attribute choices instead of
  relying on hidden picker options. Library imports reject unknown keys before making changes.
- Added real SDK/HTTP CRUD, relationship, immutable-type, referenced-deletion, read-only, and
  same-account/same-tenant cross-Space coverage. Updated Go/browser discovery expectations for the
  40-tool set. No new browser view or bundled asset was introduced.
- Necessary implementation checkpoints passed:
  - `go test ./server -run '^TestMCPMetadataManagementCRUD$' -count=1 -timeout 60s`
  - `go test ./server -run '^TestMCPMetadataManagementRelations$' -count=1 -timeout 60s`
  - `go test ./server -run '^TestMCPMetadataManagementRejectsReadonlyAndInvalidTypes$|^TestMCPBearerCredentialCannotCrossSpaces$' -count=1 -timeout 90s`
  Initial test failures were test-adapter assumptions: SDK `GetError` is server-only and empty
  units/attribute names are omitted by existing projections. Assertions now use client error text
  and the actual optional-field contract. Cross-Space management checks assert business-level
  `not_found` errors with valid schemas, rather than passing on a missing edit boolean.
- Updated source catalogs for four shared-model validation messages in German, French,
  and Italian, marked fuzzy with the Codex translation comment. `go generate ./i18n` succeeded;
  existing missing-entry diagnostics remain. Concurrent user cleanup edits appeared during this
  work and were preserved, including the updated translation extractor. The narrow CRUD and
  cross-Space checkpoint passed again against those current context/UI APIs after generation.
- Touched Go files were formatted. Complete Go/model/browser verification, build/vet, and manual
  management-client journeys were not performed for this extension. Slice 05 and its plan box
  remain pending verification; prior review results do not verify newly added behavior.

Extensions 06–09 implementation, 2026-09-30:

- Implemented the requested priority order: original-byte downloads, typed field filters, note
  history/management, and standalone filed-entry rename/move. The registered tool set is now 49;
  the catalog and contracts describe inputs and bounds. No schema migration is required.
- `download_file` reads bounded, version-pinned plaintext ranges through the canonical storage
  reader and returns base64/metadata, without browser authentication or internal object references.
  The [byte-range ADR](../adrs/2026.09-mcp_original_byte_ranges.md) records bounds and the extra
  prefix I/O cost of compressed/encrypted storage. Read-only credentials retain read access.
- Extracted typed property predicates into `property.FilePropertyFilter` for Browse and MCP.
  Added numeric/money/date ranges and strict MCP binding; preserved browser normalization and
  unconfigured/open-range states. Fixed the existing unchecked-Checkbox predicate to include
  documents missing that property's assignment even when another field is assigned. Stale browser
  property IDs no longer index a missing definition.
- Note tools call `DocumentNotes` for current access, author/owner permissions, legacy handling,
  replacement, deletion, and retained history. The model's shared query now permits pagination;
  MCP previews/body windows and new text inputs are bounded. Added `conflict` mapping for historical
  mutation conflicts. No attribution or timestamps are invented for legacy entries.
- Added `FileOrganizationService` as a scoped filed-entry adapter to existing filesystem methods.
  Rename validation now rejects invalid canonical names and returns persistence errors; optional
  new-child movement from the current parent is permitted as an actual move. Collisions/cycles and
  failed child creation preserve transaction state; classification, source, versions, and history
  remain intact. Root/Inbox entries use their existing lifecycle rather than standalone organization.
- Added SDK/HTTP byte, filter, note, and organization coverage. Necessary focused implementation
  checkpoints passed in order:
  - `go test ./server -run '^TestMCPDownloadRoundTrip$' -count=1 -timeout 90s`
  - `go test ./server -run '^TestMCPFieldFilters$' -count=1 -timeout 60s`
  - `go test ./server -run '^TestMCPNotesHistory$' -count=1 -timeout 90s`
  - `go test ./server -run '^TestMCPOrganization$' -count=1 -timeout 90s`
  - `go test ./server -run '^TestMCPDownloadRejectsForeignSpaceFile$|^TestMCPNotesHistory$|^TestMCPOrganizationAuthorization$' -count=1 -timeout 90s`
  Initial failures came from fixture privacy contexts, a tokenizer-sensitive test word, and
  error-message/status assumptions. Corrections retain the real API/model semantics.
- Discovery expectations and source documents were updated; Go files were formatted. Complete
  model/browser/regression suites, build/vet, fault-injected storage/cancellation cases, and manual
  client workflows remain pending. Later coverage additions have not been broadly verified; slice
  delivery boxes remain open. Prior review evidence does not cover these new changes.

## Operating implemented slices

After deploying the generated migration and rebuilt application, open Account → MCP credentials.
Create a label/Space-scoped credential, copy its token once, and configure a bearer-capable client
with the displayed `/mcp` URL and `Authorization: Bearer <token>`. HTTPS is required outside
development; no OAuth discovery flow is provided by this slice.

The registered tools currently cover scoped Space/Inbox inspection, bounded OCR, Inbox upload,
existing-metadata classification, destination discovery/creation, folder/non-folder filing, and
filed search with typed field conditions, metadata definition/attribute management, library import,
original-byte download chunks, note/history operations, and filed rename/move. `get_file` includes
typed classification once the tenant's metadata public-ID backfill
is complete. Inbox, directory, and filed-search pages default to 50 items, cap at 100, and reject
offsets above 1,000,000; OCR windows default to 12,000 and cap at 50,000 Unicode characters.
Revocation from Account settings takes effect on subsequent requests, including existing SDK
clients. Losing current account/tenant/Space access also denies reads. All mutation tools require a
read/write credential.

These are implementation instructions, not evidence that the deployment/client journey passed.

## Protocol references

Reviewed on 2026-09-18:

- [Official Go SDK and version compatibility](https://github.com/modelcontextprotocol/go-sdk).
- [SDK tools and structured output][sdk-tools].
- [MCP Streamable HTTP, 2026-07-28][mcp-http].
- [MCP authorization, 2026-07-28][mcp-auth].
- [HTMX response events](https://htmx.org/headers/hx-trigger/).

[mcp-http]: https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http
[mcp-auth]: https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization
[sdk-tools]: https://github.com/modelcontextprotocol/go-sdk/blob/main/docs/server.md
[feature-upload]: ../features/2026.09-mcp_server/spec.md#upload-contract
