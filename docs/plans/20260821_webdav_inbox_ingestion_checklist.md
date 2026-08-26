# WebDAV Inbox Ingestion Checklist

Specification: `docs/specs/20260821_webdav_inbox_ingestion.md`

Plan: `docs/plans/20260821_webdav_inbox_ingestion_plan.md`

Invariants: `docs/invariants/webdav_inbox_ingestion.md`

## Preparation

- [ ] Confirm the configured S3-compatible test backend supports
  `ChecksumFullObjectCRC32C` and checksum-enabled `StatObject`.
- [ ] Record baseline results for focused upload, quota, scheduler, and server
  tests before changing the shared upload path.
- [x] Identify every new-file, new-version, account-temporary, extraction, and
  final-copy caller of the shared filesystem methods.
- [x] Identify every permanent/temporary object cleanup path.
- [ ] Confirm migration generation and existing/new tenant initialization in a
  disposable environment.
- [x] Keep the invariant document in implementation and review scope.

## Schema And Generated Types

- [x] Add the file-source `GoType()` enum with `UnknownLegacy`,
  `WebInterface`, `PWAOSOpen`, `URLImport`, `WebDAV`, and
  `SystemExtraction`.
- [x] Add the DAV resource state enum with `Uploading`, `Active`, and
  `CleanupPending`.
- [x] Add `WebDAVCredential` main schema with account/tenant ownership, Space
  public ID, label, unique username, sensitive salt/hash, usage, and revocation
  fields.
- [x] Add credential owner/tenant indexes and public/common mixins.
- [x] Add browser-session credential privacy for owner and tenant-owner access.
- [x] Add `WebDAVResource` tenant schema with credential ID, Space, File,
  StoredFile, path, state, heartbeat, and finalization fields.
- [x] Add partial active-path uniqueness scoped by credential and Space.
- [x] Add DAV stale-cleanup and linked-row indexes.
- [x] Add immutable `File.source` with `UnknownLegacy` default.
- [x] Add source-filter indexes matching Inbox sort/query shapes.
- [x] Add immutable `TemporaryFile.source`.
- [x] Add `TemporaryFile.content_sha256` and `storage_crc32c`.
- [x] Add TemporaryFile conversion claim token, destination tenant/Space, and
  progress fields.
- [x] Add nullable `StoredFile.storage_crc32c`.
- [x] Add nullable unique `StoredFile.source_temporary_file_public_id`.
- [x] Add `StoredFile.source_conversion_claim_token`.
- [x] Add nullable `upload_last_progress_at` to the shared upload status mixin.
- [x] Make Tenant public ID immutable without changing stored values.
- [x] Make Space public ID immutable without changing stored values.
- [x] Run `gofmt` on manually changed Go files.
- [x] Run repository code generation.
- [x] Generate main and tenant migrations through `cmd/migrate`.
- [x] Review generated diffs without hand-editing Ent, enum, or migration output.
- [ ] Verify migration of existing databases and initialization of new tenants.

## Shared Streaming Integrity

- [x] Count and limit plaintext bytes before transformation.
- [x] Compute plaintext SHA-256 during the first read.
- [x] Preserve gzip-then-age transform order.
- [x] Compute transformed byte count, SHA-256, and stdlib Castagnoli CRC32C.
- [x] Configure MinIO PUT for full-object CRC32C.
- [x] Run checksum-enabled `StatObject` after PUT.
- [x] Compare backend object size with local transformed count.
- [x] Require backend checksum mode `FULL_OBJECT`.
- [x] Compare backend CRC32C with local transformed CRC32C.
- [x] Fail closed if size, checksum, or checksum mode is absent or mismatched.
- [x] Compare expected and actual plaintext bytes when expected size is known.
- [x] Bound unknown/chunked streams at maximum plus one byte.
- [x] Reject zero-byte uploads.
- [x] Propagate source-read and request-cancellation errors.
- [x] Propagate gzip, age, pipe, and S3 close/write errors.
- [x] Ensure every transform goroutine reports one terminal result and is joined.
- [x] Refresh upload progress at a bounded interval while bytes flow.
- [x] Recheck quota with actual plaintext bytes during finalization.
- [x] Remove failed temporary objects immediately when possible.
- [x] Retain retryable cleanup state when object removal fails.
- [x] Keep legacy successful rows with null CRC32C valid.

## Finalization-Time Visibility

- [x] Change new-file preparation to create hidden StoredFile intent only.
- [x] Create File and FileVersion only in successful finalization.
- [x] Set `WebInterface` in successful Browse/Inbox browser finalization.
- [x] Set `SystemExtraction` in successful extracted/generated child
  finalization.
- [x] Change new-version preparation to create hidden StoredFile intent only.
- [x] Create the new FileVersion link only in successful finalization.
- [x] Do not mutate File name/source/metadata during version preparation.
- [x] Apply any intended version-time rename in the finalization transaction.
- [x] Generate extension-aware unique Inbox names in the tenant write
  transaction.
- [x] Ensure concurrent writers cannot select the same generated name.
- [x] Keep all request-body and S3 I/O outside DB transactions.
- [x] Ensure finalization failure leaves no visible new file/version.

## Streaming HTTP Ingress

- [ ] Add the smallest dedicated streaming action wrapper beside the current
  router.
- [ ] Resolve authentication/route state in short transactions and close them
  before network I/O.
- [ ] Replace upload `ParseMultipartForm`/`FormFile` use with
  `Request.MultipartReader`.
- [ ] Ensure destination metadata precedes file content or arrives through
  validated query/header fields.
- [ ] Stream browser upload parts directly into the shared pipeline.
- [ ] Stream PWA multi-file parts sequentially with per-file guarantees.
- [ ] Run URL-import outbound HTTP outside router transactions.
- [ ] Verify no upload path creates a full local spool or full-file buffer.

## Account-Temporary Conversion

- [ ] Set `PWAOSOpen` when staging PWA/share/open-with files.
- [ ] Set `URLImport` when staging URL imports.
- [ ] Claim TemporaryFile conversion with a random compare-and-swap token.
- [ ] Record destination tenant/Space and suspend expiry while active.
- [ ] Permit takeover only after one hour without progress.
- [ ] Create/reuse one hidden tenant StoredFile keyed by TemporaryFile public ID.
- [ ] Bind the hidden StoredFile to the current conversion token.
- [ ] Clean stale hidden rows only when their token matches the stale claim.
- [ ] Verify account-object backend size and full-object CRC32C before reading.
- [ ] Recompute plaintext count/SHA-256 while decrypting.
- [ ] Require recomputed plaintext identity to match TemporaryFile metadata.
- [ ] Stream into a newly verified tenant temporary object.
- [ ] Recheck the same token in short main-then-tenant finalization transactions.
- [ ] Reuse an already successful tenant result rather than create another File.
- [ ] Copy TemporaryFile source into the tenant File during finalization.
- [ ] Commit tenant success before marking main converted.
- [ ] Clear claim only after main conversion marker succeeds.
- [ ] Ensure an obsolete worker can clean only its own random object/hidden row.

## Verified Final Storage

- [ ] Verify final destination with checksum-enabled `StatObject`.
- [ ] Require destination size and full-object CRC32C to match the temporary row.
- [ ] Set and commit `copied_to_final_destination_at` only after verification.
- [ ] Keep reads on temporary storage until that copy-state commit succeeds.
- [ ] If server copy cannot preserve/report CRC32C, PUT already transformed bytes
  through the checksum-enabled path.
- [ ] Never decrypt, recompress, or locally spool during final copy.
- [ ] Never trust a pre-existing destination object without verification.
- [ ] Delete temporary object only in a later safe step/pass after copy-state
  commit.

## Credential Domain And UI

- [x] Implement generated ASCII-safe username and high-entropy secret.
- [x] Hash secret with existing salted account utilities.
- [x] Use constant-time verification and a fake-hash path for missing usernames.
- [x] Add owner credential creation for currently accessible Spaces.
- [x] Return URL, username, and secret once with `Cache-Control: no-store`.
- [x] Never render the secret after the creation response.
- [x] Add owner revoke without deleting metadata or reusing username.
- [x] Record revocation time and revoking account.
- [x] Add Account settings list with label, tenant, Space, username, created,
  last-used, and revoked state.
- [x] Handle inaccessible/deleted destinations without leaking protected names.
- [x] Add tenant-owner inspection/revoke under tenant user management.
- [x] Prevent tenant owners from creating a user credential or seeing a secret.
- [x] Prevent credential owners from inspecting another account's records.
- [x] Use existing HTMX snackbar/event and focused partial refresh patterns.
- [x] Preserve focus and add explicit accessible labels for copy/create/revoke.

## WebDAV Authentication And Authorization

- [x] Register the WebDAV boundary separately from browser-session redirects.
- [x] Reject cleartext HTTP outside development before Basic verification.
- [ ] Trust only direct TLS or configured trusted proxy information.
- [x] Return the specified Basic challenge for missing, wrong, or revoked
  credentials.
- [x] Add the narrow pre-context credential lookup by unique username.
- [x] Add bounded failed-auth rate limiting owned by server infrastructure.
- [x] Key failed-auth limiting only by normalized remote address and attempted
  username; exclude Authorization, secret, stored hash, and raw DAV path.
- [x] Compare credential tenant/Space scope with endpoint public IDs.
- [x] Return not found for endpoint scope mismatch.
- [x] Explicitly require non-deleted account and tenant.
- [x] Explicitly require unexpired tenant assignment.
- [x] Explicitly require non-deleted tenant User.
- [x] Load Space through existing privacy/assignment rules.
- [x] Add error-returning WebDAV context construction; do not use `OnlyX`.
- [x] Do not use `Tenant.HasAccount` for WebDAV.
- [x] Throttle successful `last_used_at` updates.
- [ ] Before finalization, acquire main write lock through conditional credential
  touch and recheck main authorization.
- [ ] Then acquire tenant write lock through conditional DAV reservation touch
  and recheck tenant User/Space access.
- [ ] Commit tenant finalization before releasing the main transaction.
- [ ] Keep request/S3 I/O outside both transactions.

## WebDAV Protocol Boundary

- [x] Mount `/webdav/{tenantPublicID}/{spacePublicID}/` before the root fallback.
- [x] Require Basic authentication for every method, including discovery.
- [x] Implement authenticated direct `OPTIONS` with `DAV: 1, 2`.
- [x] Add `MS-Author-Via: DAV`.
- [x] Return exactly `Allow: OPTIONS, PROPFIND, PUT, LOCK, UNLOCK, MOVE`; omit
  HEAD and every blocked method.
- [x] Expose only structural `/` and `/Inbox/` via PROPFIND.
- [x] Return no file children from `/Inbox/` at every depth.
- [x] Reject file PROPFIND, HEAD, and GET without revealing a resource.
- [x] Use URL `path` semantics and exact `Inbox` casing.
- [x] Reject traversal, nested paths, empty basename, trailing file slash,
  controls/NUL, decoded separators, and encoded separators.
- [x] Apply `filenamex.IsAllowed` to decoded basenames.
- [x] Validate absolute MOVE Destination host and same tenant/Space endpoint.
- [x] Limit PROPFIND and LOCK XML bodies to 64 KiB.
- [x] Add bounded response recording for status remapping only.
- [x] Never buffer upload content in the response recorder.
- [x] Return generic protocol-safe response bodies.
- [x] Redact Authorization, secrets, content, hashes, and raw user paths from logs.

## DAV Filesystem And Locks

- [x] Add synthetic directory FileInfo/File implementations for `/` and Inbox.
- [x] Return only Inbox from root Readdir and no children from Inbox Readdir.
- [x] Hide active aliases and DMS files from read/list Stat/OpenFile calls.
- [x] Return streaming upload File from PUT OpenFile.
- [x] Return no-op placeholder from LOCK OpenFile.
- [x] Ensure placeholder Close creates no File, version, StoredFile, alias,
  filename reservation, or S3 object.
- [x] Reject Mkdir and RemoveAll unconditionally.
- [x] Implement Rename as narrow MOVE only.
- [x] Wrap server-owned `webdav.NewMemLS()` with credential path namespacing.
- [x] Limit each credential to 32 active locks.
- [x] Cap requested/refreshed lock duration at one hour.
- [x] Release lock accounting on expiry and unlock.
- [x] Strip content ETag from successful PUT response.
- [x] Map inner close/finalization errors to specified HTTP/WebDAV statuses.

## WebDAV PUT, Aliases, And MOVE

- [ ] Authenticate, authorize, validate path, and preflight known size/quota
  before reading PUT body.
- [ ] Release stale alias atomically if its linked file left Inbox or disappeared.
- [ ] Reserve `WebDAVResource(uploading)` before reading body.
- [ ] Return `409` before body read for same-credential active path conflict.
- [ ] Allow different credentials to reserve the same DAV path independently.
- [ ] Attach hidden StoredFile and refresh alias progress during streaming.
- [ ] Verify storage outside transactions.
- [ ] Recheck credential/current permissions with fixed finalization lock order.
- [ ] Generate unique DMS name in final tenant transaction.
- [ ] Use existing Inbox filename case-sensitivity semantics for PUT and MOVE
  conflict checks.
- [ ] Create one `File(source=webdav, is_in_inbox=true)` and one version.
- [ ] Mark StoredFile success and activate alias in the same transaction.
- [ ] Return `201` only after commit.
- [ ] Keep valid File/alias if response delivery fails; retry returns `409`.
- [ ] Cleanup reservation/object/hidden row on known failure.
- [ ] Retain `CleanupPending` only when cleanup itself fails.
- [ ] Permit MOVE only for same credential, same Space, non-deleted,
  non-directory, still-Inbox linked file.
- [ ] Reject an active destination alias before inner DAV dispatch.
- [ ] Generate unique DMS destination name without overwrite.
- [ ] Update only File.Name and DAV alias.
- [ ] Ensure `Overwrite: T` never invokes RemoveAll or replaces content.
- [ ] Release/deny alias once the file leaves Inbox.

## Inbox Source UI And Translations

- [ ] Ensure version upload, move, Inbox processing, restore, and metadata updates
  never change File source.
- [ ] Verify historical rows read as `UnknownLegacy` without inference.
- [ ] Add repeated `source=` values to Inbox URL state.
- [ ] Treat absent source as all sources.
- [ ] Reject invalid source values.
- [ ] OR selected sources and AND them with existing predicates.
- [ ] Preserve search/sort and reset pagination on source changes.
- [ ] Include source values in HTMX refresh, load-more, list/table, and URL state.
- [ ] Render low-emphasis source in Inbox list layout.
- [ ] Render source in Inbox table layout.
- [ ] Render read-only source in Inbox and Browse file details.
- [ ] Restore source selections through shared URL and browser back/forward.
- [ ] Add German, French, and Italian translations with fuzzy flag and required
  translator comments.
- [ ] Keep user-provided labels and filenames untranslated.
- [ ] Verify filter controls, credential controls, and source text are accessible
  on compact and expanded layouts.

## Scheduler Recovery

- [ ] Query stale hidden uploads by one hour without persisted progress.
- [ ] Query stale DAV resources by one hour without persisted progress.
- [ ] Atomically claim cleanup before object/multipart deletion.
- [ ] Abort incomplete multipart upload when supported.
- [ ] Run S3 cleanup outside DB transactions.
- [ ] Delete DAV resource after successful cleanup.
- [ ] Retry `CleanupPending` after S3 failure.
- [ ] Release active aliases whose linked file is missing/deleted/not Inbox.
- [ ] Claim stale main conversion token by compare-and-swap.
- [ ] Mark main converted when a successful tenant result already exists.
- [ ] Clean only unfinished tenant state belonging to the stale token.
- [ ] Clear claim/restore expiry when no tenant result exists.
- [ ] Scan temporary prefixes in bounded batches for unreferenced old objects.
- [ ] Preserve current upload, successful-copy-pending, account temporary,
  preview, and other referenced workflow objects.
- [ ] Never orphan-clean the final prefix.
- [ ] Never delete a successful file's only verified readable object.

## Tests

- [ ] Add path, method, status, source, credential, lock, and unique-name unit
  tests.
- [ ] Add raw DAV protocol tests for all approved client request sequences.
- [ ] Add tests for zero/oversize/truncated/chunked/canceled request bodies.
- [ ] Add gzip/age/pipe/S3 failure and no-deadlock tests.
- [ ] Add missing/mismatched backend size/CRC32C tests.
- [ ] Add invisible-before-finalization and failed-version metadata tests.
- [ ] Add concurrent same-alias and same-DMS-name tests.
- [ ] Add revocation, assignment, tenant-user, and Space barrier-race tests.
- [ ] Add context error tests proving no panic/redirect.
- [ ] Add account conversion crash-window, takeover, and stale-worker tests.
- [ ] Add final-copy corruption/pre-existing destination tests.
- [ ] Add heartbeat and one-hour no-progress recovery tests.
- [ ] Add orphan-cleanup protection tests for every referenced workflow.
- [ ] Add source assignment, staging propagation, filter, and query-plan tests.
- [ ] Add Account settings and tenant-owner browser tests.
- [ ] Add source list/table/details and history browser tests.
- [ ] Verify logs contain no Authorization, secret, content, or raw body data.

## Verification

- [x] Run `gofmt -w <changed-go-files>`.
- [x] Run `go generate ./...`.
- [x] Run migration generation through
  `CGO_ENABLED=1 go run ./cmd/migrate/main.go webdav_inbox_ingestion`.
- [x] Review generated migration and Ent diffs; do not hand-edit them.
- [x] Run focused model/filesystem tests.
- [x] Run focused server/protocol tests.
- [x] Run focused scheduler/recovery tests.
- [x] Run focused action/UI tests.
- [ ] Run directly relevant Toxiproxy/bad-network tests.
- [x] Run `go test ./...`.
- [x] Run `go build ./...`.
- [x] Run the tagged CGO build.
- [x] Run `go vet ./...`.
- [ ] Run `npm run test:e2e`.
- [x] Run `git diff --check`.
- [x] Document any unrelated existing failure with focused passing evidence.

## Completion Criteria

- [ ] Users can create multiple generated, one-Space credentials and see a secret
  once.
- [ ] Tenant owners can inspect/revoke but cannot create/reveal credentials.
- [ ] Current permissions and revocation are enforced on request and finalization.
- [ ] DAV discovery exposes Inbox structure and no user file.
- [ ] PUT creates one visible verified Inbox file and no partial file.
- [ ] Same active path conflicts; DMS name conflicts generate unique names.
- [ ] LOCK creates no file; MOVE performs only the permitted rename.
- [ ] Every ingestion source records the correct immutable source.
- [ ] Source appears and filters correctly in Inbox and file details.
- [ ] Account conversion is idempotent across every crash boundary.
- [ ] Temporary-to-final copy is verified before read switching/deletion.
- [ ] One-hour no-progress recovery is idempotent and preserves successful data.
- [ ] All required generation, migration, test, build, vet, E2E, and diff checks
  pass.
