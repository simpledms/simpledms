# WebDAV Inbox Ingestion Plan

Source specification: `docs/specs/20260821_webdav_inbox_ingestion.md`

Implementation checklist:
`docs/plans/20260821_webdav_inbox_ingestion_checklist.md`

Invariant reference: `docs/invariants/webdav_inbox_ingestion.md`

Status: Proposed

Date: 2026-08-21

## Business Domain And Outcome

SimpleDMS remains the system of record for tenant files. WebDAV is a supporting
ingestion boundary that lets a user send files into one Space Inbox without
turning SimpleDMS into a remotely browsable filesystem.

Primary actors:

- users who create one or more device credentials for Spaces they can access;
- scanners, desktop/mobile DAV clients, and scripts that upload files;
- tenant owners who inspect and revoke tenant credentials;
- scheduler and storage maintenance processes that recover interrupted uploads.

The feature is complete when an authorized client can discover the virtual
Inbox, stream a file into verified S3 storage, and receive success only after one
visible Inbox file is committed with its immutable source.

## Goals

- Add the Space endpoint
  `/webdav/{tenantPublicID}/{spacePublicID}/` using
  `golang.org/x/net/webdav` behind a restrictive adapter.
- Expose only the structural `/` and `/Inbox/` collections.
- Accept flat, non-empty file PUTs while rejecting all file reads, listings,
  overwrite, delete, copy, folder, and property-edit behavior.
- Add account-owned, Space-scoped, generated WebDAV credentials.
- Enforce current credential, tenant, user, and Space access on every request and
  immediately before finalization.
- Harden the shared upload path for every source, including storage checksum and
  final-copy verification.
- Prevent partial logical files from becoming visible.
- Preserve PWA and URL-import source and byte identity through main-to-tenant
  conversion.
- Show and filter immutable source in Inbox and show it in file details.
- Recover stalled state through the existing scheduler without deleting the only
  verified copy of a successful file.

## Non-Goals

- Full document-tree WebDAV.
- Reading, downloading, previewing, deleting, copying, or organizing DMS files
  through DAV.
- Nested DAV paths or DAV-created folders.
- Persistent/distributed DAV locks.
- A new service, worker, queue, CLI, upload format, or local spool directory.
- Account-password or browser-session authentication for DAV.
- Admin-created credentials or automatic credential expiry.
- Historical source inference.
- Named DAV-client compatibility guarantees.

## Current Repository Context

### Upload And Storage

- `action/browse/upload_file_cmd.go` and
  `action/browse/upload_file_version_cmd.go` coordinate normal uploads.
- `action/inbox/upload_file_cmd.go` is the legacy Inbox upload route.
- `action/openfile/upload_files_cmd.go` stages PWA/share/open-with files in the
  main DB before Space selection.
- `model/main/temporaryfile/upload_from_url_service.go` stages URL imports.
- `model/tenant/filesystem/s3_file_system.go` prepares rows, computes plaintext
  SHA-256, compresses, encrypts, and writes temporary S3 objects.
- `model/tenant/filesystem/prepared_upload.go` and
  `prepare_account_upload.go` carry the current preparation state.
- `scheduler/process_files.go` copies tenant temporary objects to final storage
  and deletes temporary objects later.
- `util/uploadx` contains existing request-failure cleanup helpers.

The current flow must change in four important ways:

1. `ParseMultipartForm`/`FormFile` may spill uploads locally and must not be used
   by streaming upload routes.
2. Router-created transactions currently surround handlers; request-body, URL,
   and S3 I/O must run outside transactions.
3. New logical `File` and `FileVersion` rows are prepared before bytes are
   verified; their visible links must move to finalization.
4. Expected byte count, transform close errors, backend checksum, and destination
   copy integrity are not fully enforced.

### Identity And Authorization

- Accounts, tenant assignments, and tenant records are in the main DB.
- Spaces and tenant users are in tenant DBs.
- `server/router.go` builds browser-session contexts and redirects unauthenticated
  requests; WebDAV needs a separate Basic-auth boundary.
- `ctxx.NewTenantContext` uses panic-style lookup and is not safe at an external
  credential boundary.
- `model/main/tenant.Tenant.HasAccount` does not enforce assignment expiry and
  must not authorize WebDAV.
- Tenant and Space public IDs currently use mutable public-ID mixins even though
  the DAV endpoint must remain stable.

### Inbox And Settings UI

- `action/inbox/files_list_partial.go` owns Inbox search, sort, pagination, and
  list/table state.
- `action/inbox/file_list_item_partial.go` and `file_table.go` render the two
  Inbox layouts.
- `action/inbox/file_metadata_partial.go` and
  `action/browse/file_attributes_partial.go` render file details.
- `action/dashboard/account_page.go` and `account_cards_partial.go` are the
  account settings surface.
- `action/managetenantusers` owns tenant user management.

## Domain And Context Design

### Ubiquitous Language

- **WebDAV credential:** generated username and one-time secret owned by one
  account and scoped to one Space.
- **DAV path:** temporary client-facing ingestion alias, not a document path.
- **Active DAV resource:** durable alias used only for retry conflict and narrow
  MOVE while the linked file remains in Inbox.
- **Source:** immutable origin category on the logical File.
- **Plaintext integrity:** raw byte count and SHA-256 before transformation.
- **Stored integrity:** transformed byte count, SHA-256, and full-object CRC32C.
- **Finalization:** short transaction that links verified storage into a visible
  logical file/version.
- **Progress heartbeat:** persisted evidence that an upload or conversion remains
  active; stale means one hour without progress.

### Logical Contexts

- The **File context** owns logical identity, source, versions, unique names,
  Inbox state, quota, and visibility.
- The **Credential context** owns WebDAV credentials and revocation in the main
  DB.
- The **WebDAV ingestion context** translates DAV methods/paths into File-context
  operations and keeps DAV aliases out of the document model.
- The **Storage pipeline** owns streaming transforms, S3 integrity, final copy,
  and cleanup.

Use focused transaction-script/application-service code around existing Ent
models. A new aggregate framework, CQRS, event sourcing, or service boundary is
not justified.

## Target Data Model

### Shared File Source

Add a generated `GoType()` enum under the existing common model hierarchy:

```text
UnknownLegacy
WebInterface
PWAOSOpen
URLImport
WebDAV
SystemExtraction
```

Add immutable `source` fields to tenant `File` and main `TemporaryFile`.
Historical rows default to `UnknownLegacy`. New versions, moves, restores, and
Inbox state changes never update the field.

### Main WebDAV Credential

Add `db/entmain/schema/webdav_credential.go` with:

- immutable account and tenant edges;
- immutable Space public ID scalar;
- required label;
- unique generated username;
- sensitive salt/hash;
- `last_used_at`, `revoked_at`, and `revoked_by_account_id`;
- common/public-ID mixins and indexes for owner and tenant-owner lists.

The main DB owns the credential because authentication occurs before a tenant DB
can be selected. A narrow pre-context repository may query only by unique
username and return only authentication/context fields.

### Tenant DAV Resource

Add `db/enttenant/schema/webdav_resource.go` with:

- credential public ID scalar;
- Space edge;
- nullable File and StoredFile edges;
- canonical DAV path;
- `Uploading`, `Active`, and `CleanupPending` state;
- progress/finalization timestamps;
- a partial unique active path index scoped by credential and Space;
- stale-recovery and linked-row indexes.

The row is not a file or audit log. Delete it after the file leaves Inbox or
cleanup finishes.

### Upload And Conversion Fields

Extend upload status with `upload_last_progress_at`.

Add to main `TemporaryFile`:

- `content_sha256` and `storage_crc32c`;
- immutable source;
- conversion claim token, destination tenant/Space, and progress timestamp.

Add to tenant `StoredFile`:

- nullable `storage_crc32c` for new strict uploads;
- nullable unique source TemporaryFile public ID;
- conversion claim token.

Existing successful rows remain valid when `storage_crc32c` is null. Strict
CRC32C verification applies to uploads created after the schema change.

### Stable Endpoint IDs

Change Tenant and Space public-ID mixins to immutable while preserving every
existing value. Do not introduce a second endpoint-ID concept.

## Target Upload Architecture

### Transaction-Free Streaming Boundary

Add the smallest dedicated streaming action wrapper beside existing router
registration. It authenticates and resolves route identity in short read
transactions, closes them, streams request/outbound content, and opens short
preparation/finalization transactions only.

Browser/PWA multipart routes use `Request.MultipartReader`. Destination metadata
must arrive before the file part or through validated query/header data so a file
never waits in a local spool. URL import uses the same orchestration around its
outbound response stream.

### Shared Pipeline

All new upload sources use this order:

```text
raw reader
  -> limit/count
  -> plaintext SHA-256
  -> gzip
  -> age, when enabled
  -> transformed count/SHA-256/CRC32C
  -> checksum-enabled temporary S3 PUT
  -> checksum-enabled StatObject
```

Success requires:

- non-zero raw bytes;
- exact byte match when expected size is known;
- no source, cancellation, pipe, gzip-close, or age-close error;
- backend size equal to local transformed count;
- backend `FULL_OBJECT` CRC32C equal to local CRC32C;
- final quota check using plaintext bytes.

`UploadInfo.Size` is not the backend verification boundary, and multipart
SHA-256 is composite. Use stdlib `hash/crc32` Castagnoli locally and mandatory
checksum-enabled `StatObject` for the backend comparison. Keep local transformed
SHA-256 as the strong persisted storage hash.

### Visibility

Preparation creates hidden StoredFile intent only. New `File` and `FileVersion`
links are created during successful finalization. New-version preparation must
not rename or otherwise mutate the existing File; any requested rename happens
with the successful version link.

### Verified Final Copy

The scheduler verifies destination size and full-object CRC32C, then commits
`copied_to_final_destination_at`. Reads switch only after that commit. Temporary
deletion happens in a later safe step/pass. If server-side copy cannot
preserve/report the checksum, stream already transformed bytes into a
checksum-enabled PUT.

## Main-To-Tenant Conversion

PWA and URL imports require cross-database idempotency:

1. Claim the main TemporaryFile by compare-and-swap, recording destination and a
   random token.
2. Create/reuse one hidden tenant StoredFile keyed uniquely by the main temporary
   public ID. Store the current claim token as ownership metadata, but never make
   it part of the uniqueness boundary.
3. Verify the account object, decrypt while recomputing plaintext identity, and
   write a newly verified tenant temporary object.
4. Finalize with short main-then-tenant transactions and matching token checks.
5. Commit tenant success first, then mark main converted.
6. On retry, reuse an existing successful tenant result rather than create a
   duplicate.

A stale worker can clean only its token's random object and hidden row. Scheduler
takeover uses conditional token replacement and never cleans a newer or
successful result.

## WebDAV Boundary

### Authentication And Finalization Ordering

Every method requires Basic auth over HTTPS outside development. Invalid,
missing, and revoked credentials return the same challenge and never redirect.
Endpoint scope mismatch returns not found.

Use an error-returning context builder that explicitly checks non-deleted account,
tenant, tenant user, unexpired assignment, and privacy-filtered Space access.

Failed-auth limiter keys contain only normalized remote address and attempted
username. They never include Authorization, secret, stored hash, or raw DAV path.

After upload, finalization acquires locks in this order:

1. main write transaction and conditional active-credential touch;
2. current credential/account/tenant/assignment checks;
3. tenant write transaction and conditional DAV-reservation touch;
4. current tenant-user/Space checks;
5. tenant finalization commit;
6. main transaction release.

The conditional writes must acquire SQLite write locks before authorization
queries. No request-body or S3 I/O occurs inside either transaction.

### Protocol Adapter

An outer handler owns HTTPS/auth, body limits, method/path gating, direct
`OPTIONS`, context injection, response recording, status remapping, and log
redaction. Authenticated OPTIONS returns `DAV: 1, 2`, `MS-Author-Via: DAV`, and
exactly `Allow: OPTIONS, PROPFIND, PUT, LOCK, UNLOCK, MOVE`; HEAD is omitted.

The inner `webdav.Handler` uses a custom write-only filesystem:

- synthetic `/` and `/Inbox/` directories;
- root `Readdir` returns Inbox, Inbox returns no children;
- file Stat/Open remain hidden for PROPFIND/HEAD/GET;
- PUT OpenFile returns a streaming upload file;
- LOCK OpenFile returns a no-op placeholder;
- Mkdir and RemoveAll reject;
- Rename performs only the narrow MOVE transaction.

The pinned handler calls PUT File.Stat before File.Close and uses that FileInfo to
build an ETag. Close waits for storage verification and finalization. The outer
handler maps close failures away from the library's generic 405 and strips the
content ETag because the path is not readable/stable.

### Locks And Resource Bounds

Wrap one server-owned `webdav.NewMemLS()` with credential path namespacing and:

- 64 KiB PROPFIND/LOCK XML limits;
- at most 32 active locks per credential;
- at most one-hour lock duration;
- accounting release on expiry/unlock.

LOCK placeholders create no DMS rows, aliases, filename reservations, or objects.

### PUT, Retry, And MOVE

Reserve the credential/path alias before reading PUT. A repeated active path for
the same credential returns conflict before body consumption. Different
credentials may use the same DAV path independently.

Finalization generates an extension-aware unique DMS filename in the tenant
transaction using the repository's existing Inbox filename case-sensitivity
semantics, creates one source=`webdav` Inbox File/version, and activates the
alias. If the HTTP response is lost, the committed file stays and retry conflicts.

MOVE is permitted only for the same credential's linked non-deleted file while
it remains in the same Space Inbox. It changes only File.Name and the alias.
Validate Destination host/endpoint/path before dispatch. Reject active
destinations even when Overwrite is true; RemoveAll is never used.

## UI And HTMX

### Credential Management

The dedicated WebDAV credentials page shows owned credentials across tenant
Spaces as a list/table. Its navigation-rail item sits between Account and
System. An empty state is shown before credentials exist. Otherwise credentials
are grouped by Space in tabs, with the complete WebDAV URL shown once in a
docked toolbar for the active Space. Only the URL text is a copy link, with a
tooltip and snackbar feedback. Creation is exposed as a FAB and as a
Space-preselected first list item in each tab. Revocation is in each credential
row's overflow menu. A main-area filter icon opens a side sheet with Active and
Revoked checkbox chips; Active is the default. The one-time secret response
still uses
`Cache-Control: no-store`. Tenant user management shows credential metadata and
revoke action to tenant owners only. Commands return existing snackbar/event
responses and refresh the smallest stable partial.

### Source

Add low-emphasis source text to Inbox list/table and read-only source to file
details. Add a repeated `source=` multi-select URL filter. Selected sources OR
together and AND with existing search/filter predicates. Source changes reset
pagination and survive all HTMX refresh, list/table, load-more, sort, and browser
history flows.

Use existing widgets and server state. Do not add client-side source state.

## Scheduler Recovery

Extend existing scheduler loops in bounded batches:

- reclaim hidden uploads and DAV resources after one hour without heartbeat;
- release aliases whose files left Inbox or disappeared;
- claim and recover stale main-to-tenant conversion tokens;
- complete a missing main conversion marker when tenant success exists;
- clean only unfinished rows/objects belonging to the stale token;
- abort incomplete multipart uploads where supported;
- remove unreferenced account/tenant temporary objects older than the grace
  period;
- retry `CleanupPending` state after S3 failures.

Object deletion happens outside DB transactions. Never scan/delete the final
prefix and never delete the only verified readable object of a successful file.

## Delivery Phases

### Phase 1: Schema And Generated Types

Add source/resource enums, credential/resource schemas, upload/conversion fields,
indexes/privacy, immutable endpoint IDs, generated Ent code, and generated
migrations. Verify existing and new tenant initialization before behavior changes.

### Phase 2: Shared Upload Integrity And Visibility

Implement the common counters/hashes/CRC32C, close/error propagation, backend
StatObject verification, heartbeat, final quota check, finalization-time links,
and transaction-free multipart/URL ingress. Assign `WebInterface` and
`SystemExtraction` in the browser and extraction finalizers changed in this
phase. Keep existing upload tests green before adding WebDAV.

### Phase 3: Staging And Final Storage

Make account-to-tenant conversion claim-based and idempotent. Verify plaintext
identity during re-encryption. Harden temporary-to-final copy and preserve legacy
rows with no new CRC field. Assign and propagate `PWAOSOpen` and `URLImport`
at their staging/finalization boundaries.

### Phase 4: Credential Domain And Authorization

Implement credential generation/hash/privacy, owner and tenant-owner commands,
the narrow Basic repository, HTTPS/rate limits, non-panicking context creation,
and fixed finalization lock ordering.

### Phase 5: WebDAV Protocol And Ingestion

Implement endpoint registration, outer gate, synthetic filesystem, bounded lock
wrapper, PUT streaming file, status mapping, durable aliases, retry conflict,
unique name finalization with source=`webdav`, and narrow MOVE.

### Phase 6: Source Presentation And Credential UI

Render the already assigned source and URL filter, add account credential
management and tenant-owner inspection/revoke, and add translations/accessibility
behavior.

### Phase 7: Recovery And Security Hardening

Add scheduler recovery, orphan cleanup, claim takeover, log redaction, race
coverage, and all corruption/failure tests.

### Phase 8: Release Verification

Run focused and full Go tests/build/vet, generated-code and migration review,
SQLite query-plan checks, automated protocol request sequences, and browser E2E.

## Testing Strategy

### Unit

- source/path parsing and unique filenames;
- credential verification and limiter/lock bounds;
- upload counts, hashes, CRC32C, close/error propagation;
- DAV resource and conversion claim transitions;
- source-filter URL state.

### Protocol

Use raw `httptest` requests and DAV XML for OPTIONS/PROPFIND, direct PUT,
LOCK-PUT-UNLOCK, temp PUT-MOVE, chunked PUT, retry conflict, hidden reads,
rejected methods/nesting, cross-endpoint Destination, lock limits, and
Overwrite=true safety.

### Integration And Storage

- invisible-before-finalization and exactly-one-file success;
- known truncation, zero/oversize, cancellation, quota race, S3/checksum errors;
- revocation/permission barrier races;
- source assignment and staging propagation;
- conversion retry after every main/tenant crash boundary;
- final-copy verification and no successful-content cleanup;
- concurrency for same alias and same DMS filename.

### Browser

- create and revoke owner credential, secret shown once;
- tenant-owner inspect/revoke without secret/create access;
- source in both Inbox layouts and details;
- multi-source URL state and back/forward restoration.

## Likely Files To Touch

### Schema And Model

- `db/entmain/schema/webdav_credential.go`
- `db/entmain/schema/temporary_file.go`
- `db/entmain/schema/tenant.go`
- `db/enttenant/schema/webdav_resource.go`
- `db/enttenant/schema/file.go`
- `db/enttenant/schema/stored_file.go`
- `db/enttenant/schema/space.go`
- `db/entx/upload_status_mixin.go`
- source/resource enum packages under existing model conventions
- focused credential and DAV resource model files

### Upload, Server, And Scheduler

- `model/tenant/filesystem/s3_file_system.go`
- `model/tenant/filesystem/prepared_upload.go`
- `model/tenant/filesystem/prepare_account_upload.go`
- `util/uploadx/*`
- `server/router.go` or a focused streaming-action wrapper file
- `server/server.go`
- new focused WebDAV handler/auth/filesystem/lock files
- existing browse, Inbox, open-file, and URL-import handlers
- `scheduler/process_files.go` and focused cleanup files if needed

### UI

- `action/dashboard/account_page.go`
- `action/dashboard/account_cards_partial.go`
- `action/managetenantusers/*`
- `action/inbox/files_list_partial.go`
- `action/inbox/file_list_item_partial.go`
- `action/inbox/file_table.go`
- `action/inbox/file_metadata_partial.go`
- `action/browse/file_attributes_partial.go`
- route/event files only where existing surfaces need them
- source translation entries in German, French, and Italian message files

Do not edit generated Ent files, generated enum files, generated migration
contents, `catalog.gen.go`, or `out.gotext.json` manually.

## Verification Commands

Run focused packages after each phase, then the complete set:

```bash
gofmt -w <changed-go-files>
go generate ./...
CGO_ENABLED=1 go run ./cmd/migrate/main.go webdav_inbox_ingestion
go test ./model/main/... ./model/tenant/... ./util/...
go test ./server ./scheduler ./action/...
go test ./...
go build ./...
CGO_ENABLED=1 go build -tags "sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_icu" ./...
go vet ./...
npm run test:e2e
git diff --check
```

Use `docker compose up -d` for integration tests requiring the configured
S3-compatible backend. Slow bad-network/Toxiproxy tests run only when validating
the changed upload/recovery paths.

## Risks And Mitigations

- **DAV defaults expose too much:** gate methods/paths outside the library and
  test every blocked method.
- **Shared hardening causes broad regression:** complete it before DAV and keep
  focused browser/PWA/version/URL tests green.
- **Cross-DB duplicate after crash:** unique source TemporaryFile key plus token
  compare-and-swap and tenant-first success commit.
- **Revocation races finalization:** fixed main-then-tenant write-lock ordering and
  barrier tests.
- **Slow active upload is reclaimed:** persist throttled progress and define stale
  as one hour without progress.
- **S3 backend omits integrity metadata:** fail closed for new uploads and verify
  backend support before release.
- **Cleanup removes good content:** claim cleanup, retain verified temporary data
  until final verification, and test every crash boundary.
- **Lock/auth state consumes process memory:** bounded limiter, XML, lock count,
  duration, and expiry accounting.

## Open Questions

No blocking product questions remain.

Engineering choices left during implementation:

- Basic-safe generated username/secret format and length.
- Failed-auth limiter interval and bounded storage details.
- Minimal structural DAV property set needed by protocol tests.
- Exact focused package/file split.
- Whether to inspect PutObject response fields in addition to mandatory
  checksum-enabled StatObject verification.
