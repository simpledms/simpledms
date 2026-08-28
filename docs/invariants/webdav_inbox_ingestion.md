# WebDAV Inbox Ingestion Invariants

Specification: `docs/specs/20260821_webdav_inbox_ingestion.md`

## Scope

These invariants apply to WebDAV credentials and protocol behavior, all shared
upload sources, immutable file source, account-temporary conversion, tenant
temporary-to-final storage, recovery, and Inbox source UI state.

WebDAV is an intake boundary. It is not a remotely browsable or editable view of
the SimpleDMS document model.

## Credential Identity And Scope

### One Owner And One Space

Rule: A WebDAV credential belongs to exactly one account, one tenant, and one
Space. It cannot authorize another account or Space.

Why: Device credentials must use the owning account's current permissions and
must not become tenant-wide bearer keys.

Enforced in:

- main `WebDAVCredential` immutable account/tenant/Space fields;
- credential creation authorization;
- endpoint-scope comparison before tenant DB access;
- current tenant and Space queries on every request.

Minimum regression tests:

- credential works only at its tenant/Space endpoint;
- valid credential at another endpoint returns not found;
- credential owner cannot inspect another account's credential.

### Stable Endpoint Identity

Rule: Tenant and Space public IDs used in WebDAV URLs are immutable.

Why: A user-managed connection must not silently stop working because a public
identifier changed.

Enforced in: Tenant and Space public-ID schema fields and generated mutation APIs.

Minimum regression tests: normal tenant and Space mutations cannot update the
public ID, and existing URLs remain unchanged after metadata edits.

### Generated Secret Is One-Time

Rule: WebDAV username and secret are generated. The secret is returned once,
sent only over HTTPS outside development, and stored only as a salted hash.

Why: Account passwords must not be reused by devices, and database access must
not reveal usable DAV secrets.

Enforced in:

- credential creation service;
- existing account password-hash utilities;
- `Cache-Control: no-store` creation response;
- settings renderers that never load sensitive fields.

Minimum regression tests:

- creation response contains the secret once;
- reload/list/admin views never contain it;
- logs never contain Authorization or the secret.

### Revocation Is Durable

Rule: Revocation disables authentication immediately but preserves credential
label, username, ownership, created/last-used/revoked times, and revoking account
until account or tenant deletion. A revoked username is never reused.

Why: Users and tenant owners need durable device history without retaining a
usable credential.

Enforced in: revoke commands, credential authentication, indexes, and settings
queries.

Minimum regression tests:

- revoked credential returns the same Basic challenge as invalid credentials;
- metadata remains visible according to owner/admin privacy;
- a new credential receives another username.

### Tenant Owner Authority Is Limited

Rule: A current tenant owner may inspect and revoke tenant credentials, but may
not create one for another user or view/recover its secret.

Why: Credentials are user-managed even when tenant owners need an incident
response control.

Enforced in: main Ent privacy and tenant-user-management commands.

Minimum regression tests: tenant-owner inspect/revoke succeeds; create/reveal
fails; non-owner tenant users cannot inspect another user's credentials.

## Authentication And Authorization

### Every DAV Method Is Authenticated

Rule: Every WebDAV method, including OPTIONS, PROPFIND, LOCK, and UNLOCK,
requires Basic authentication. WebDAV never redirects to browser sign-in and
never accepts a browser session as its credential.

Why: Discovery and lock requests reveal an endpoint and consume resources, while
browser redirects are not DAV protocol responses.

Enforced in: outer WebDAV handler mounted outside browser-session routing.

Minimum regression tests:

- unauthenticated OPTIONS/PROPFIND receive the Basic challenge;
- invalid and revoked credentials receive the same challenge;
- no DAV authentication failure redirects.

### HTTPS Before Secret Verification

Rule: Outside development, cleartext HTTP is rejected before parsing or verifying
the Basic secret. Only direct TLS or explicitly trusted proxy information can
establish HTTPS.

Why: Basic credentials are reusable and must not be accepted over cleartext or
through spoofed forwarded headers.

Enforced in: outer WebDAV transport check.

Minimum regression tests: valid credentials over cleartext fail before hash
verification; arbitrary forwarded headers cannot bypass the check.

### Current Permission Is Authoritative

Rule: A successful credential hash is insufficient. Every request requires a
non-deleted account and tenant, unexpired tenant assignment, non-deleted tenant
User, and privacy-filtered current Space access.

Why: Credentials inherit current account permissions rather than preserving the
permissions present when they were created.

Enforced in:

- narrow main credential repository;
- error-returning WebDAV context builder;
- explicit assignment-expiry checks;
- tenant User and Space queries.

Do not enforce this rule through panic-based `OnlyX` constructors. Tenant access
checks must use the shared active-assignment predicates, including expiry.

Minimum regression tests: assignment expiry, account/tenant/user deletion, and
Space removal return controlled denial without panic.

### Finalization Rechecks And Serializes Permission

Rule: After body/S3 streaming, authorization is rechecked under short write locks
in fixed main-then-tenant order before tenant finalization commits.

Why: Revocation or permission removal during a long upload must serialize either
before finalization, causing failure, or after a valid committed upload.

Enforced in:

1. main conditional active-credential touch and authorization query;
2. tenant conditional DAV-reservation touch and User/Space query;
3. tenant finalization commit before main transaction release.

Browser, archive, and account-temporary finalization use the same main-then-tenant
lock order with fresh assignment, User, and Space checks.

No request, URL-fetch, or S3 I/O occurs inside these transactions.

Minimum regression tests: barrier-controlled credential revocation, assignment
change, tenant-user deletion, and Space removal races have deterministic results
and never expose partial files.

## DAV Namespace And Capabilities

### Structural Namespace Only

Rule: The endpoint exposes exactly `/` and `/Inbox/` as structural collections.
PROPFIND of Inbox never lists DMS files, newly uploaded files, or active aliases.

Why: WebDAV clients need a mountable structure, but the product capability is
consume-only.

Enforced in: custom synthetic filesystem and structural PROPFIND behavior.

Minimum regression tests:

- root depth 0 returns root only;
- root depth 1/infinity adds Inbox only;
- Inbox at every depth returns Inbox only;
- file PROPFIND returns no resource after successful PUT.

### Flat Safe Write Path

Rule: PUT is accepted only at exact-case `/Inbox/{singleFilename}` with a safe
decoded basename. Nested paths, traversal, empty names, trailing file slashes,
controls, NUL, and encoded/decoded separators are rejected.

Why: DAV paths must not become filesystem traversal or implicit DMS folder
semantics.

Enforced in: URL-path parser, `URL.EscapedPath` checks, and
`filenamex.IsAllowed`.

Minimum regression tests cover every rejected path class and valid filenames
with/without extensions.

### Narrow Method Set

Rule: The only accepted methods are OPTIONS, structural PROPFIND, PUT, LOCK,
UNLOCK, and narrow MOVE. GET, HEAD, DELETE, COPY, MKCOL, PROPPATCH, POST, and
unknown methods cannot reach file mutations or content.

Authenticated OPTIONS returns `DAV: 1, 2`, `MS-Author-Via: DAV`, and exactly
`Allow: OPTIONS, PROPFIND, PUT, LOCK, UNLOCK, MOVE`. Unauthenticated OPTIONS
returns the Basic challenge. HEAD and every blocked method are omitted from
Allow.

Why: Unfiltered `x/net/webdav` behavior is broader than the product contract.

Enforced in: outer method gate, direct OPTIONS response, custom filesystem, and
status mapper.

Minimum regression tests verify all three OPTIONS headers, rejected methods, and
that RemoveAll/Mkdir/read OpenFile are never used successfully.

### No File Metadata Resource

Rule: An uploaded DAV path is never a stable readable resource. File HEAD, GET,
and PROPFIND reveal no content, size, ETag, hash, or DMS metadata.

Why: The DMS Inbox, not DAV, is the source of truth after ingestion.

Enforced in: context-gated Stat/OpenFile and PUT response ETag filtering.

Minimum regression tests: PUT followed by HEAD/GET/PROPFIND reveals no file.

## Logical File Visibility

### Preparation Is Hidden

Rule: Upload preparation creates hidden storage intent only. It does not create a
visible new File/FileVersion link or mutate an existing File.

Why: A process or storage failure before verification must not expose a partial
document or change visible metadata.

Enforced in:

- `PreparedUpload` creation;
- hidden StoredFile upload status;
- finalization-time File/FileVersion creation;
- new-version flow that delays rename/metadata changes.

Minimum regression tests:

- no new File/version is visible during upload;
- failed new upload leaves no logical file;
- failed version upload leaves current version, filename, source, and metadata
  unchanged.

### Visibility Starts At Finalization Commit

Rule: A new logical file/version becomes visible only in the transaction that
records verified storage success and commits all links.

Why: Visibility and verified bytes must have one authoritative boundary.

Enforced in: new-file, new-version, account-conversion, and WebDAV finalization
services.

Minimum regression tests: before-commit queries cannot see the file; after
successful commit exactly one File/version is visible.

### Network I/O Never Holds DB Transactions

Rule: Request-body reads, outbound URL downloads, encryption/compression streams,
S3 PUT/Stat/copy/delete, and DAV response delivery occur outside DB transactions.

Why: Slow clients and object storage must not hold SQLite locks or block unrelated
tenant work.

Enforced in: streaming action wrapper and short preparation/finalization helpers.

Minimum regression tests: browser, PWA, URL, and DAV test hooks observe no active
main/tenant transaction during network streaming.

## Source Immutability And UI Meaning

### Source Describes Initial Ingestion

Rule: `File.source` records how the logical file first entered SimpleDMS and is
immutable afterward.

Values are:

- `WebInterface` for normal browser uploads;
- `PWAOSOpen` for PWA/share/open-with staging;
- `URLImport` for URL staging;
- `WebDAV` for DAV PUT;
- `SystemExtraction` for extracted/internal generated children;
- `UnknownLegacy` for historical/unclassified rows.

Why: Source is provenance, not current location or latest-version transport.

Enforced in: immutable schema field and creation finalizers.

Minimum regression tests: version, move, Inbox processing, restore, and metadata
updates never change source.

### Historical Source Is Not Guessed

Rule: Existing rows migrate to `UnknownLegacy`; no heuristic reconstructs source.

Why: Inaccurate provenance is worse than explicit unknown provenance.

Enforced in: schema/migration default.

Minimum regression test: seeded legacy rows read as `UnknownLegacy`.

### Main Staging Preserves Source

Rule: PWA and URL source is stored on TemporaryFile and copied to the tenant File
after Space selection.

Why: The later conversion request cannot reliably infer the original ingress.

Enforced in: staging services and account-to-tenant finalization.

Minimum regression tests: PWA persists as `PWAOSOpen`, URL import as
`URLImport`, and destination selection does not reclassify either.

### Source Filter Has Stable Semantics

Rule: An absent source parameter means all sources. Repeated selected values OR
together and AND with existing Inbox predicates. Invalid values are rejected.
Source changes preserve search/sort and reset pagination.

Why: The server and URL, not browser-only state, remain the source of truth.

Enforced in: Inbox state binding, query builder, HTMX includes, and URL updates.

Minimum regression tests:

- single/multiple/absent/invalid source values;
- composition with search and sort;
- load-more/list-table/history preserve selection;
- list, table, and file details render the same immutable label.

## Upload Byte And Storage Integrity

### Raw Byte Completion

Rule: Every new upload contains at least one plaintext byte, never exceeds the
existing per-file limit, and exactly matches expected bytes when a trusted size is
known. Unknown/chunked streams are bounded while reading.

Why: Clean early EOF, malformed requests, and over-limit streams must not become
successful shorter files.

Enforced in: raw limit/counter before hashing/transformation.

Minimum regression tests: zero, known truncation, known excess, chunked success,
chunked overflow, malformed body, and cancellation.

### Plaintext And Stored Identity Are Separate

Rule: Uploads persist plaintext count/SHA-256 and transformed count/SHA-256.
Full-object CRC32C is retained as diagnostic metadata only.

Why: Plaintext hash supports document identity; transformed values verify the
actual encrypted/compressed storage object.

Enforced in: shared stream, upload result, TemporaryFile, and StoredFile.

Minimum regression tests compare every value with independently calculated test
fixtures in encryption-enabled and disabled modes.

### Backend Verification Is Mandatory For New Uploads

Rule: Before finalization, the stored object is read back. Its size and SHA-256
must equal the locally calculated transformed values.

Why: MinIO UploadInfo size is client-counted and S3-compatible providers differ in
their checksum metadata. Application-calculated SHA-256 verifies the exact stored
bytes independently of provider checksum behavior.

Enforced in: direct stored-object size and SHA-256 verification after PUT.

Minimum regression tests: wrong size, unreliable backend checksum metadata,
stored SHA-256 mismatch, multipart stream, and backend failure leave no visible
file.

Historical successful rows without stored SHA-256 are backfilled by hashing their
temporary object before final persistence or account conversion.

### Every Transform Error Fails Upload

Rule: Source read, cancellation, gzip close, age close, pipe, goroutine, S3 PUT,
and verification read errors all prevent finalization.

Why: An accepted object can still be incomplete when transform closure failed.

Enforced in: ordered close/error propagation and exactly one joined terminal
result per goroutine.

Minimum regression tests inject each failure and prove no deadlock, visible file,
or untracked temporary object.

### Quota Uses Actual Plaintext Bytes

Rule: Tenant quota uses plaintext size and is rechecked with actual bytes during
finalization.

Why: Unknown lengths and concurrent uploads can invalidate preflight quota.

Enforced in: preflight, streaming limit, and final tenant transaction.

Minimum regression tests: known over-quota preflight and finalization quota race
return 507 and clean temporary state.

## Account-Temporary Conversion

### One Active Claim

Rule: A TemporaryFile conversion has one compare-and-swap claim token and one
recorded destination at a time. Takeover is allowed only after one hour without
progress.

Once a tenant result commits, the claim remains pinned to that tenant until the
main conversion marker is repaired. A retry through another tenant cannot take
over the staged file.

Why: Concurrent Space-selection requests and restarts must not convert the same
staged object twice.

Enforced in: main conversion claim fields and heartbeat updates.

Minimum regression tests: active claim blocks another worker; stale claim can be
taken over; old worker token no longer finalizes.

### One Successful Tenant Result

Rule: A main TemporaryFile public ID maps to at most one successful tenant
StoredFile/File result in a tenant DB.

Why: Tenant commit may succeed before the main converted marker commits.

Enforced in: unique nullable source TemporaryFile ID and idempotent finalization.

Minimum regression tests: retry after tenant commit/main failure reuses success
and creates no duplicate File/version/object.

### Conversion Preserves Plaintext Identity

Rule: Before tenant finalization, account object size/SHA-256 is verified and
decryption recomputes plaintext count/SHA-256 equal to TemporaryFile metadata.

Why: Re-encryption must not silently truncate or alter the staged content.

Enforced in: direct account-object hashing and streaming tenant persistence.

Minimum regression tests: account checksum, plaintext count, or plaintext hash
mismatch prevents tenant File creation.

### Stale Workers Cannot Clean New Work

Rule: Cleanup and finalization are conditional on the same claim token. An old
worker may clean only its own random object and hidden row.

Why: Claim takeover must not let the old process destroy a replacement or a
successful result.

Enforced in: main/tenant token fields, conditional updates, and scheduler claim
takeover.

Minimum regression tests: resume obsolete worker after takeover and after
success; replacement and success remain intact.

## Final Storage Safety

### Final Destination Is Verified Before Use

Rule: Reads switch from temporary to final storage only after the final destination
is read back and its size and stored-object SHA-256 match the verified temporary
row.

Why: A successful copy call or pre-existing destination does not prove byte
identity.

Enforced in: final-copy scheduler and `copied_to_final_destination_at` update.

Minimum regression tests: missing/mismatched checksum, wrong size, and corrupt
pre-existing destination keep reads on temporary storage.

### Temporary Is Deleted Last

Rule: The verified temporary object remains readable until the final object is
verified and the copy state commits. Temporary deletion happens afterward.

Why: Copy or scheduler failure must not remove the only verified bytes.

Enforced in: StoredFile open selection and scheduler copy/delete order.

Minimum regression tests: copy failure and restart preserve readability; repeated
delete is idempotent.

### Object Deletion Includes Retained Versions

Rule: Deleting an object removes its current value, retained versions, and delete
markers. New application-created buckets do not enable Object Lock implicitly.

Why: A delete marker alone neither releases storage nor removes sensitive bytes.

Enforced in: shared object deletion used by upload cleanup and scheduler cleanup.

Minimum regression tests: deleting from a versioned bucket removes every version
and delete marker; missing objects remain an idempotent success.

## DAV Aliases, Names, Retries, And MOVE

### Alias Is Not A File

Rule: A WebDAVResource is visible only to same-credential PUT conflict and MOVE.
It is never exposed by DAV discovery/read operations.

Why: The alias supports client protocol patterns without creating a remote file
namespace.

Enforced in: alias queries and context-gated custom filesystem.

Minimum regression tests: active alias is hidden from PROPFIND/HEAD/GET.

### Active Same-Path Retry Conflicts

Rule: A repeated PUT from the same credential to the same active DAV path returns
409 before body read while its linked file remains in Inbox. It never replaces or
duplicates the file.

Why: A client may retry after finalization when the success response was lost.

Enforced in: partial unique alias index and reservation transaction.

Minimum regression tests: response-loss retry conflicts and the original File
remains unchanged.

Different credentials may use the same DAV path independently.

### DMS Filename Conflicts Never Overwrite

Rule: DMS filename conflicts generate an extension-aware unique name under the
tenant write transaction using the repository's existing Inbox filename
case-sensitivity semantics. No existing row or object is overwritten.

Why: DAV paths are intake names and cannot safely identify existing DMS files.

Enforced in: finalization and MOVE unique-name service.

Minimum regression tests: concurrent and case-variant same-name uploads/MOVEs
follow existing comparison semantics, select distinct names when conflicting,
and preserve extensions.

### MOVE Is A Narrow Rename

Rule: MOVE can change only File.Name and the alias for the same credential's
non-deleted, non-directory file while it remains in the same Space Inbox. It
cannot change content, version, source, metadata, tags, properties, or Inbox
state.

Why: Temporary-name upload clients need final naming, not general editing.

Enforced in: Destination validation, alias ownership query, and tenant rename
transaction.

Minimum regression tests:

- MOVE after Inbox processing is denied and alias released;
- cross-host/tenant/Space/nested Destination is denied;
- destination conflict generates a unique DMS name;
- `Overwrite: T` never calls RemoveAll or replaces content.

## Lock Resource Bounds

### Lock Is Compatibility State Only

Rule: DAV locks are process-local protocol state. Losing locks on restart never
changes stored bytes; clients may reacquire.

Why: Persistent distributed locks are unnecessary for consume-only behavior and
the current single-process SQLite deployment.

Enforced in: server-owned wrapper around `webdav.NewMemLS()`.

Minimum regression test: lock loss/reacquisition has no DMS or S3 side effect.

### Locks Are Bounded And Namespaced

Rule: Lock roots are namespaced by credential, LOCK/PROPFIND XML is limited to
64 KiB, each credential has at most 32 active locks, and requested/refreshed
duration is capped at one hour.

Why: `x/net/webdav` lock state is not hardened against unbounded untrusted input.

Enforced in: outer body limits and bounded LockSystem wrapper.

Minimum regression tests: oversized body and excess lock fail before allocation;
expiry/unlock release accounting; one credential cannot block another.

### LOCK Never Creates An Inbox File

Rule: Locking a missing path creates only a no-op placeholder required by the DAV
handler. It creates no File, FileVersion, StoredFile, DAV alias, filename
reservation, or S3 object.

Why: `x/net/webdav` opens and closes missing lock resources, and zero-byte DMS
files are forbidden.

Enforced in: request operation context and placeholder File.Close.

Minimum regression test: LOCK then UNLOCK without PUT leaves every persistence
store unchanged.

## Cleanup And Recovery

### Stale Means No Progress

Rule: An upload or conversion becomes stale after one hour without a persisted
progress heartbeat, not one hour after start.

Why: Slow but progressing uploads must not be reclaimed.

Enforced in: upload, DAV resource, and conversion heartbeat fields and scheduler
queries.

Minimum regression tests: progressing operation older than one hour survives;
no-progress operation is reclaimed.

### Cleanup Is Claimed And Idempotent

Rule: Scheduler cleanup uses conditional state/token claims. Repeated passes may
retry cleanup but cannot clean another worker's state or a successful result.

Why: Scheduler overlap, restart, and S3 failure are normal recovery conditions.

Enforced in: upload state, DAV `CleanupPending`, conversion tokens, and conditional
updates.

Minimum regression tests: repeated passes, cleanup failure/retry, stale worker,
and concurrent scheduler passes preserve correct state.

### Successful Bytes Are Never Orphan-Cleaned

Rule: Orphan scanning applies only to temporary prefixes and deletes an old
object only after proving no current upload, successful copy-pending StoredFile,
TemporaryFile, preview/conversion, or other workflow row references it. Final
prefix objects are never orphan-scanned.

Why: Object age alone cannot distinguish an orphan from the only verified copy
of a valid file.

Enforced in: bounded scheduler object listing and reference checks.

Minimum regression tests: every referenced workflow survives cleanup; an
unreferenced temporary object is removed; final-prefix objects are untouched.

### Main Conversion Recovery Prefers Success

Rule: When a stale main conversion claim has a successful tenant result, recovery
marks main converted. It never deletes the tenant File/object.

Why: Tenant success can commit before the main marker.

Enforced in: stale account-conversion reconciliation order.

Minimum regression test: missing main marker is repaired without duplicate or
deletion.

## Security And Observability

Rule: Normal logs never contain Authorization headers, generated secrets, request
bodies, file content, or plaintext hashes. User-controlled paths are sanitized.

Why: Upload and authentication diagnostics must not become a credential/content
exfiltration path.

Enforced in: outer WebDAV logger, storage errors, failed-auth limiter, and
scheduler logs.

Minimum regression tests capture logs for authentication, invalid path, S3
failure, and cleanup and assert sensitive values are absent.

Rule: Failed Basic attempts and DAV XML/lock state consume bounded process
resources.

Failed-auth limiter keys contain only normalized remote address and attempted
username. They never contain Authorization, secret, stored hash, credential
material, or raw DAV filename/path.

Why: The endpoint is internet-facing and performs expensive secret verification
and stateful lock operations.

Enforced in: bounded failed-auth limiter, body limits, and bounded lock wrapper.

Minimum regression tests verify key composition, eviction/expiry, and bounded
state under repeated invalid input.

## Review Checklist

An implementation review must confirm:

- credentials never outlive current authorization;
- DAV never reveals a DMS file resource;
- no visible file/version exists before verified finalization;
- every new upload has verified plaintext and stored identity;
- main-to-tenant conversion is idempotent across every commit boundary;
- final copy is verified before read switching or temporary deletion;
- aliases and locks remain protocol-only bounded state;
- stale cleanup cannot delete active or successful content;
- source remains immutable and URL-backed filter behavior stays server-owned.
