# WebDAV Inbox Ingestion

Status: Proposed

Date: 2026-08-21

## Business Outcome

SimpleDMS must let users connect scanners, desktop file managers, mobile clients,
and scripts to a Space-specific WebDAV endpoint and drop files into the Space
Inbox without exposing the existing document tree over WebDAV.

The capability is ingestion, not remote file management. A WebDAV credential is
an account-owned intake key for exactly one Space. It may create new Inbox files
and narrowly rename a still-Inbox file uploaded by that same credential. It must
not list, read, overwrite, edit, delete, copy, or reorganize existing files.

Upload success is an integrity guarantee. It means the request stream was fully
consumed, bounded, hashed, compressed, encrypted, written to temporary S3
storage, verified against S3, and finalized in tenant data. Failed or interrupted
uploads must not expose partial logical files.

## Confirmed Decisions

- Use `golang.org/x/net/webdav`. It is the Go-maintained `x/net` package already
  required by this repository, not a Go standard-library package.
- Support common desktop, scanner/MFP, mobile, and CLI request patterns through
  automated protocol tests. Do not promise named products or maintain a manual
  compatibility matrix.
- Use one stable URL per Space:
  `/webdav/{tenantPublicID}/{spacePublicID}/`.
- Expose exactly one virtual child collection, `/Inbox/`.
- Structural discovery may reveal `/` and `/Inbox/`, but never user files.
- Reject folders and nested uploads. Accept files only at
  `/Inbox/{singleFilename}`.
- Allow `OPTIONS`, structural `PROPFIND`, `PUT`, `LOCK`, `UNLOCK`, and a narrow
  `MOVE` operation.
- Block reads, file metadata probes, deletes, copies, folder creation, property
  edits, and all other methods.
- A `LOCK` on a missing file path must not create an empty Inbox file.
- `MOVE` may only rename a file uploaded by the same credential while the file
  remains in Inbox. It must not change content or move the file out of Inbox.
- Scope every credential to exactly one Space. A user may create multiple named
  credentials for the same Space, for example one per device.
- Use HTTP Basic authentication over HTTPS only outside development.
- Generate the username and a high-entropy secret. Show the secret once and
  store only a salted hash.
- Credentials do not expire automatically.
- Retain revoked credential metadata until its account or tenant is deleted.
- Users manage their credentials in Account settings.
- Tenant owners inspect and revoke tenant credentials in tenant user management.
  They cannot create credentials for another user or see a secret.
- Enforce current account, tenant, and Space access for every request and again
  before upload finalization.
- Revocation or permission loss during an upload prevents finalization and
  triggers cleanup.
- Keep current tenant-owner implicit Space access behavior.
- Store one immutable source on each logical file. New versions and moves do not
  change it.
- Source values are `WebInterface`, `PWAOSOpen`, `URLImport`, `WebDAV`,
  `SystemExtraction`, and `UnknownLegacy`.
- Set all historical files to `UnknownLegacy`; do not infer their source.
- Show source in both Inbox layouts and in file details.
- Add a URL-backed multi-select source filter to the Inbox.
- Generate an extension-aware unique DMS filename when an Inbox filename is
  already used. Never overwrite.
- A repeated `PUT` by the same credential to the same active DAV path returns a
  conflict while the first linked file remains in Inbox.
- Reject zero-byte files.
- Accept chunked or otherwise unknown-length requests while enforcing the
  maximum size during streaming. Compare exact bytes when `Content-Length` is
  present.
- Use the existing upload-size limit and tenant storage quota.
- Stream through the application directly into temporary S3 storage. Do not
  spool a full local file or buffer a full file in memory.
- Return success only after raw-byte counting, plaintext SHA-256, gzip and age
  closure, backend-reported S3 size and full-object CRC32C verification, quota
  checks, and DB finalization.
- Fail closed when the S3-compatible backend omits or mismatches the required
  checksum or size.
- Harden the shared upload pipeline for every source, not only WebDAV.
- Clean known failures immediately. Reconcile uploads with no progress for one
  hour and orphan temporary objects through the existing scheduler.
- Preserve source through account-temporary PWA and URL-import staging.

## Ubiquitous Language

| Term | Meaning |
| --- | --- |
| WebDAV credential | An account-owned generated username and one-time secret scoped to one Space. |
| Credential label | A user-managed device name such as `Office scanner`. It is not used for authentication. |
| WebDAV endpoint | The Space URL `/webdav/{tenantPublicID}/{spacePublicID}/`. |
| Structural resource | The virtual `/` or `/Inbox/` collection needed for client discovery. It is not a DMS file. |
| DAV path | A client-visible path such as `/Inbox/scan.pdf`. It is not a durable read URL. |
| Active DAV resource | A temporary alias for a completed upload while its linked file remains in Inbox. |
| Lock placeholder | A hidden no-op file object used only when a client locks a missing path before upload. |
| Source | The immutable category describing how a logical file entered SimpleDMS. |
| Plaintext integrity | The raw uploaded byte count and SHA-256 before compression and encryption. |
| Stored integrity | The byte count, SHA-256, and full-object CRC32C of the transformed S3 object. |
| Finalization | The tenant transaction that records verified storage and makes the logical file visible. |
| Reconciliation | Scheduler recovery of stale upload state and unreferenced temporary objects. |

## Scope

### In Scope

- Main DB credential persistence and lifecycle.
- User credential creation and revocation in Account settings.
- Tenant-owner inspection and revocation in tenant user management.
- A Basic-authenticated WebDAV boundary separate from browser-session routing.
- Current account, tenant, Space, and credential authorization.
- Structural WebDAV discovery exposing `/Inbox/` only.
- Streaming `PUT` ingestion into Inbox.
- In-memory `LOCK` and `UNLOCK` compatibility.
- Same-credential rename of a still-Inbox WebDAV upload.
- Durable DAV alias tracking across process restarts.
- Immutable source tracking, display, and Inbox filtering.
- Shared upload and S3-copy integrity hardening.
- Scheduler reconciliation after process or network failure.
- Automated WebDAV request-sequence tests.

### Out Of Scope

- Listing existing or newly uploaded files through WebDAV.
- WebDAV downloads, previews, or file metadata access.
- WebDAV folder creation or nested upload paths.
- Content overwrite, delete, copy, or property editing.
- Moving a WebDAV upload out of Inbox.
- Full document-tree WebDAV access.
- Named-client compatibility guarantees or manual compatibility matrices.
- Persistent or distributed DAV locks.
- A separate WebDAV service, queue, or worker.
- Admin-created credentials.
- Automatic credential expiry.
- Per-credential quotas or permissions beyond current account and Space access.
- Inferring the source of historical files.

## Domain and Context Boundaries

### Subdomain Analysis

- **File storage and versioning: core domain.** It owns logical files, versions,
  Inbox state, tenant isolation, quota, source, and the invariant that visible
  files have verified bytes.
- **WebDAV ingestion: supporting subdomain.** It translates a narrow external
  protocol into the existing file-ingestion capability.
- **Credential management: supporting subdomain.** It owns generated
  credentials, revocation, and audit metadata.
- **WebDAV protocol and S3 storage: generic capabilities.** They remain behind
  adapters and do not become the document model.

### Bounded Contexts

The modular monolith keeps four logical boundaries:

- The **File context** owns `File`, `FileVersion`, source, unique filename
  generation, and Inbox lifecycle.
- The **Credential context** owns WebDAV credentials in the main DB because
  authentication occurs before a tenant context exists.
- The **WebDAV ingestion context** owns path validation, method gating, Basic
  authentication behavior, locks, DAV aliases, and protocol status mapping.
- The **Storage pipeline context** owns hashing, compression, encryption,
  temporary S3 writes, integrity verification, final copies, and cleanup.

These are package boundaries in the existing process. They are not new services.

### Context Map

- WebDAV ingestion is an anticorruption layer between DAV paths and the File
  context. A DAV path is only an ingestion alias and never a document path.
- WebDAV ingestion consumes credential authentication and current permission
  checks from the Credential context.
- WebDAV ingestion requests `create Inbox file from stream` and `rename active
  Inbox upload` behavior from the File context.
- Every ingestion source uses the same Storage pipeline. WebDAV does not add a
  second storage format or a weaker success definition.

## Protocol Contract

### Endpoint and Paths

Mount the endpoint at:

```text
/webdav/{tenantPublicID}/{spacePublicID}/
```

The only structural resources are:

```text
/
/Inbox/
```

The only writable file shape is:

```text
/Inbox/{singleFilename}
```

Path handling must:

1. Use URL `path` semantics, never OS `filepath` semantics.
2. Require exact `Inbox` casing.
3. Reject empty basenames and trailing-slash file paths.
4. Reject nested paths, traversal, NUL, control characters, decoded slash, and
   decoded backslash.
5. Detect encoded separators using `URL.EscapedPath` before normalizing.
6. Apply the existing `filenamex.IsAllowed` validation to the decoded basename.
7. Apply existing filename case-sensitivity rules when checking DMS conflicts.

Tenant and Space public IDs used by the endpoint must be immutable. Change their
schema mixins to immutable without changing existing values. A configured
WebDAV URL must not change because an endpoint identifier was edited.

An absolute `Destination` header for `MOVE` must use the current request host and
the same tenant and Space endpoint. Cross-host, cross-tenant, cross-Space, root,
and nested destinations are rejected.

### Authentication on Every Method

All methods, including `OPTIONS` and `PROPFIND`, require the Space credential.
An unauthenticated probe receives a Basic challenge and may retry with
credentials. WebDAV requests must never redirect to browser sign-in.

Credential failures return:

```text
401 Unauthorized
WWW-Authenticate: Basic realm="SimpleDMS WebDAV", charset="UTF-8"
```

Missing usernames, wrong secrets, and revoked credentials use the same response.
Do not reveal which part failed.

### Allowed Methods

| Method | `/` and `/Inbox/` | `/Inbox/{filename}` | Other paths |
| --- | --- | --- | --- |
| `OPTIONS` | Narrow capability response | Narrow capability response | `404` or `405` |
| `PROPFIND` | Structural `207` response | `404` | `404` |
| `HEAD` | `405` | `404` | `404` |
| `PUT` | `405` | Create one Inbox file | `409` or `404` |
| `LOCK` | Lock only | Lock or no-op placeholder | `409` or `404` |
| `UNLOCK` | Unlock only | Unlock only | `409` or `404` |
| `MOVE` | `405` | Narrow same-credential rename | `409` or `404` |
| `GET` | `405` | `404` | `404` |
| All other methods | `405` | `405` | `405` or `404` |

The outer endpoint handles `OPTIONS`, because the unfiltered
`webdav.Handler` advertises unsupported methods. Return:

```text
DAV: 1, 2
MS-Author-Via: DAV
Allow: OPTIONS, PROPFIND, PUT, LOCK, UNLOCK, MOVE
```

### Status Mapping

| Condition | Status |
| --- | --- |
| Missing, malformed, wrong, or revoked Basic credential | `401 Unauthorized` with challenge |
| Cleartext HTTP outside development | `403 Forbidden` before credential verification |
| Active credential but current tenant or Space access is gone | `403 Forbidden` |
| Credential does not match endpoint tenant or Space | `404 Not Found` |
| Unsupported method | `405 Method Not Allowed` with narrow `Allow` |
| Invalid write path, active alias conflict, or invalid MOVE | `409 Conflict` |
| Zero bytes, known-length mismatch, or malformed request body | `400 Bad Request` |
| Known request body larger than the upload limit | `413 Request Entity Too Large` |
| Stream grows beyond the upload limit | `413 Request Entity Too Large` |
| Tenant quota is insufficient | `507 Insufficient Storage` |
| Successful PUT | `201 Created` |
| Successful MOVE to a new alias | `201 Created` |
| Successful UNLOCK | `204 No Content` |
| Transform, S3, or integrity verification failure | `500 Internal Server Error` |

Response bodies must use protocol-safe generic text and must not expose internal
errors, object names, database IDs, or authorization details.

### Structural PROPFIND

`PROPFIND /` behaves as follows:

- Depth `0` returns only `/`.
- Depth `1` or `infinity` returns `/` and `/Inbox/`.

`PROPFIND /Inbox/` always returns only the collection itself, regardless of
depth. `PROPFIND /Inbox/{filename}` always returns `404`, including immediately
after a successful upload.

The structural responses may expose the smallest client-compatible property set:

- `resourcetype` as collection;
- `displayname`;
- a stable structural `getlastmodified` if required.

They must not expose user file names, counts, sizes, content types, content
hashes, file ETags, uploader identities, or modification times.

### LOCK and UNLOCK

Use `webdav.NewMemLS()` for protocol-compatible locking, wrapped by a bounded
server-owned lock system. Namespace every lock root with the credential public ID
so unrelated credentials cannot block one another. The wrapper, not a package
singleton, preserves a client's locks across requests.

The lock boundary must:

- accept at most 64 KiB of LOCK XML;
- allow at most 32 active locks per credential;
- cap requested and refreshed lock duration at one hour;
- reject excess locks without allocating another lock entry;
- expire and release lock accounting when the underlying lock expires.

The current SQLite deployment is a single application process, so persistent
locks are unnecessary. A restart may invalidate a lock token; the client can
acquire a new token. Lock loss must never affect stored bytes.

`x/net/webdav` opens and closes a missing resource during `LOCK`. The custom
filesystem must detect the request operation and return a no-op placeholder.
Closing that placeholder must not:

- reserve a DAV alias;
- create `File`, `FileVersion`, or `StoredFile` rows;
- reserve a DMS filename;
- write an S3 object;
- create a zero-byte Inbox item.

### PUT

The endpoint must authenticate and validate the complete route before reading
the body.

For a known `Content-Length`:

1. Reject `0` bytes.
2. Reject a value above the existing per-file maximum before streaming.
3. Pass the expected raw size to the shared upload pipeline.
4. Require the final raw byte count to match exactly.

For chunked or unknown length:

1. Read at most `maximum + 1` raw bytes.
2. Reject on the first byte above the maximum.
3. Reject a final count of zero.
4. Treat a request-body read error or malformed chunk termination as failure.

A repeated `PUT` to the same canonical DAV path by the same credential returns
`409` before reading the body if its linked file still exists in Inbox. If that
file has left Inbox or was deleted, release the stale alias atomically and allow
a new reservation.

Different credentials may use the same DAV path. Their uploads remain separate
and DMS filename conflict handling applies.

After finalization, the uploaded path remains unreadable. Strip any content ETag
that the inner DAV handler would otherwise add to the PUT response; SimpleDMS
does not promise a stable remote resource.

### Unique DMS Filenames

Resolve Inbox filename conflicts in the final tenant write transaction. Preserve
the extension and add a numeric suffix before it:

```text
scan.pdf
scan (2).pdf
scan (3).pdf
```

For a name without an extension:

```text
scan
scan (2)
```

The transaction must check all non-deleted files in the same Space Inbox. SQLite
serializes tenant writers; integration tests must prove that concurrent uploads
cannot choose the same generated name. Do not overwrite an existing row or
object.

### Narrow MOVE

A MOVE is valid only when:

1. Source and destination are flat `/Inbox/{filename}` paths.
2. The source active DAV alias belongs to the authenticated credential.
3. The linked logical file belongs to the endpoint Space.
4. The linked file is non-directory, non-deleted, and still has
   `is_in_inbox=true`.
5. The destination DAV path is not active for the same credential.
6. The destination filename passes normal validation.
7. Current credential, account, tenant, and Space authorization still succeeds.

In one tenant write transaction:

1. Generate a unique DMS filename from the destination basename.
2. Rename `File.Name` to that generated value.
3. Replace the source DAV alias with the canonical destination alias.

MOVE must not alter content, versions, source, metadata, tags, properties, or
Inbox state. An `Overwrite: T` header never permits deletion or replacement. If
the file has left Inbox, deny MOVE and release the stale alias.

The outer handler must reject an active destination alias before dispatching
MOVE to `x/net/webdav`. `FileSystem.RemoveAll` always rejects, and protocol tests
must prove that `Overwrite: T` never calls it or changes either linked file.

## Credential Model and Lifecycle

### Main DB Schema

Add `db/entmain/schema/webdav_credential.go` with the repository's common and
public-ID mixins.

| Field | Type | Requirement |
| --- | --- | --- |
| `account_id` | int64 | Required immutable edge to the owning `Account`. |
| `tenant_id` | int64 | Required immutable edge to the main DB `Tenant`. |
| `space_public_id` | public-ID-compatible value | Immutable scalar because Space is in another DB. |
| `label` | string | Required trimmed user label. |
| `username` | string | Generated, unique, immutable, and ASCII-safe for Basic auth. |
| `secret_salt` | string/bytes | Sensitive. |
| `secret_hash` | string/bytes | Sensitive. Never returned after creation. |
| `last_used_at` | nullable time | Last successful current-permission authorization. |
| `revoked_at` | nullable time | Set once on revocation. |
| `revoked_by_account_id` | nullable int64 | Account that performed revocation. |

Indexes:

- unique `username`;
- `(account_id, tenant_id, space_public_id, revoked_at)` for self-service lists;
- `(tenant_id, account_id, revoked_at)` for tenant user management.

The main DB owns this record because Basic authentication must identify an
account and tenant before opening a tenant DB. The Space public ID is validated
against the tenant DB on every use; it is not a cross-database foreign key.

### Creation

Only the authenticated account owner may create a credential.

1. Select a tenant and Space currently accessible to the account.
2. Enter a non-empty device label.
3. Revalidate current tenant and Space access in the command.
4. Generate a globally unique username and high-entropy secret.
5. Hash the secret with the existing salted password-hash facilities. Do not add
   another cryptography dependency.
6. Persist the credential and return URL, username, and secret once.

The one-time response must use `Cache-Control: no-store`. Reloading or reopening
settings must never reveal the secret. Recovery means revoking the credential
and creating another one, not decrypting or resetting it.

### Revocation and Retention

The credential owner and a current tenant owner may revoke a credential. The
operation sets `revoked_at` and `revoked_by_account_id`. It does not delete the
record or allow the username to be reused.

Revoked metadata remains visible until account or tenant deletion. It includes
label, username, tenant, Space, created time, last used time, revoked time, and
revoked state. No UI or API reveals the stored hash or salt.

Space removal, tenant-assignment expiry, account deletion, tenant deletion, and
Space deletion deny access even if `revoked_at` is unset. Removing and later
restoring a permission does not rotate the credential; it becomes usable again
unless it was explicitly revoked.

### Basic Authentication and Context Construction

The WebDAV boundary must not use browser-session middleware, which redirects
unauthenticated requests.

For every request:

1. Reject cleartext HTTP outside development before parsing or verifying Basic
   credentials. Only trust TLS state or proxy information from configured
   trusted infrastructure, never an arbitrary forwarded header.
2. Parse Basic auth.
3. Look up the generated username through a narrowly scoped main DB repository.
4. Perform a constant-time hash verification. Use a fake hash path for missing
   usernames to reduce timing disclosure.
5. Reject missing, wrong, or revoked credentials with the same `401` challenge.
6. Compare credential tenant and Space scope with the public IDs in the URL.
7. Load the account and current tenant assignment, including assignment expiry.
8. Open the tenant DB and map the main account to its non-deleted tenant `User`.
9. Load the Space through existing privacy and assignment rules.
10. Build `MainContext`, `TenantContext`, and `SpaceContext` through a WebDAV-safe
    constructor that returns errors instead of using `OnlyX` or panicking.
11. Update `last_used_at` after successful authorization. Throttle this update so
    DAV probes do not write the main DB on every request.

Do not use `Tenant.HasAccount` for this boundary because it does not enforce
assignment expiry. Explicitly require a non-deleted account, tenant, tenant user,
unexpired tenant assignment, and privacy-filtered Space access.

After streaming, serialize authorization and finalization with short transactions
in a fixed main-then-tenant order:

1. Acquire a main write transaction and recheck credential revocation, account,
   tenant, and assignment expiry.
2. While that transaction prevents concurrent revocation, acquire a tenant write
   transaction and recheck tenant user and Space access.
3. Finalize and commit the tenant transaction.
4. Release the main transaction.

The transactions contain no request-body or S3 I/O. A revocation or permission
change is ordered either before these checks, in which case finalization fails,
or after the tenant commit, in which case the already authorized upload remains
valid. Add race tests with barriers around both permission stores.

Do not assume that a deferred SQLite transaction has acquired a write lock. The
main transaction must conditionally touch the still-active credential before its
authorization query, and the tenant transaction must conditionally touch the DAV
reservation before its Space query. A zero-row conditional update means access
changed and finalization fails.

Failed Basic authentication must be rate-limited by remote address and attempted
username using a bounded limiter owned by server infrastructure. A process-local
limiter is sufficient for the current single-process deployment. Never place
credentials, Authorization headers, or raw filenames in rate-limit keys or logs.

### Ent Privacy

Browser-session credential queries must allow:

- an account to read and revoke only its own records;
- a current tenant owner to read and revoke records for that tenant;
- no tenant owner to create a record for another account.

The unauthenticated Basic lookup is the only pre-context query. Keep it inside a
small authentication repository that can query only by unique username and
returns only fields needed to verify and construct context.

## Active DAV Resource Model

Add a small tenant DB entity such as `WebDAVResource`. It exists only to reject
ambiguous retries and support post-PUT MOVE across restarts. It is not a file
listing or an audit log.

| Field | Type | Requirement |
| --- | --- | --- |
| `credential_public_id` | string/public ID | Main DB credential identity; no cross-DB FK. |
| `space_id` | int64 | Required tenant Space ownership. |
| `file_id` | nullable int64 | Linked logical file after successful finalization. |
| `stored_file_id` | nullable int64 | Prepared storage row used for cleanup correlation. |
| `dav_path` | string | Canonical flat path such as `/Inbox/scan.pdf`. |
| `state` | enum | `Uploading`, `Active`, or `CleanupPending`. |
| `last_progress_at` | time | Throttled upload heartbeat. |
| `finalized_at` | nullable time | Set with successful finalization. |

Use the normal common/public-ID and Space mixins where appropriate.

Indexes:

- unique active path on `(credential_public_id, space_id, dav_path)` while state
  is `Uploading` or `Active`;
- `(state, last_progress_at, id)` for recovery;
- indexes on `file_id` and `stored_file_id`.

Lifecycle:

1. Reserve `Uploading` before reading the body.
2. Attach the prepared `StoredFile` identity.
3. Refresh `last_progress_at` periodically while bytes are flowing, with bounded
   DB writes.
4. In finalization, attach the `File`, set `state=Active`, and set
   `finalized_at`.
5. On MOVE, update `dav_path` after validating the linked file.
6. Delete the alias after the linked file leaves Inbox or is deleted.
7. On known failure, clean storage and DB state immediately, then delete the
   alias. If cleanup itself fails, retain `CleanupPending` for scheduler retry.
8. Reconcile `Uploading` rows with no progress for one hour.

An active alias is visible only to PUT conflict detection and same-credential
MOVE. `PROPFIND`, GET, and file HEAD behave as though it does not exist.

## File Source

### Source Type

Add a generated enum following existing `GoType()` and `enumer` patterns. A
shared package under the existing common model hierarchy may be used by both
main and tenant schemas.

```text
UnknownLegacy
WebInterface
PWAOSOpen
URLImport
WebDAV
SystemExtraction
```

### Tenant File

Add an immutable `source` enum field to `db/enttenant/schema/file.go` with
`UnknownLegacy` as the migration-safe default.

Source assignment:

| Creation flow | Source |
| --- | --- |
| Browse or Inbox upload dialog | `WebInterface` |
| PWA share target or OS file handler | `PWAOSOpen` |
| Import from URL | `URLImport` |
| WebDAV PUT | `WebDAV` |
| Archive extraction or internal generated child | `SystemExtraction` |
| Existing row or unclassified legacy path | `UnknownLegacy` |

Uploading a new version never updates `File.source`. Moving a file into or out of
Inbox never updates it. Restoring a file never updates it.

Add Inbox query indexes that keep source-filtered newest/oldest and name sorts
bounded, based on the existing Inbox index shapes. At minimum they must begin
with Space, Inbox, deletion/directory state, and source before the selected sort
key. Verify the final index order with SQLite query plans and the existing file
listing benchmark.

### Main Temporary File

Add the same immutable source field to `db/entmain/schema/temporary_file.go`.
Also add the plaintext `content_sha256` and full-object `storage_crc32c` needed to
verify later account-to-tenant persistence. Existing `sha256` remains the local
full transformed SHA-256.

Add nullable conversion-coordination fields:

- `persistence_claim_token`;
- `persistence_tenant_id`;
- `persistence_last_progress_at`.

The claim token is a random compare-and-swap owner for one account-to-tenant
conversion attempt. It is not a credential and is never user-visible.

- PWA/share/open-with staging sets `PWAOSOpen`.
- URL import staging sets `URLImport`.
- Persisting an account temporary file copies its source to the new tenant
  `File`.

This value is required because Space selection happens after the bytes are
staged. Do not infer PWA versus URL import at the later Inbox request.

### Inbox and File Details UX

Show source as low-emphasis supporting text in both Inbox list and table layouts.
Do not turn every source into a high-emphasis decorative chip. Filename and
workflow state remain primary.

Show the same immutable source as read-only metadata in file details for files
inside or outside Inbox.

Suggested user labels:

| Value | English label |
| --- | --- |
| `WebInterface` | Web upload |
| `PWAOSOpen` | Open with |
| `URLImport` | URL import |
| `WebDAV` | WebDAV |
| `SystemExtraction` | System extraction |
| `UnknownLegacy` | Unknown |

Add German, French, and Italian translations through the normal message files.
Generated translations remain fuzzy and follow the repository translation rules.

### Source Filter

Add a source filter to the existing Inbox list state and partial.

1. Use a repeated URL parameter such as `source=WebDAV&source=PWAOSOpen`.
2. An absent parameter means all sources.
3. One or more values become `source IN (...)` and are ORed together.
4. AND the source predicate with Space, Inbox, search, metadata, and deletion
   predicates.
5. Reject invalid source enum values at the request boundary.
6. Preserve current sort and search when source changes.
7. Reset pagination when source changes.
8. Include source state in HTMX refresh, load-more, list/table switch, and URL
   replacement requests.
9. Use the existing filter menu/checklist or filter-chip widgets and
   server-rendered partials. Do not add a client-side source state model.
10. Browser back/forward and a shared URL must restore the selected sources.

## Shared Upload Integrity

The current S3 upload path is the foundation, but its integrity gaps must be
closed once for browser, PWA, URL import, file versions, extraction, and WebDAV.

### Visibility Boundary

For a new logical file, preparation creates only hidden storage intent and a
`StoredFile` upload row. It must not create a visible `File` or `FileVersion`.

After storage verification, one tenant write transaction:

1. Rechecks quota and operation-specific authorization.
2. Creates the logical `File` with immutable source.
3. Creates its `FileVersion` link.
4. Marks storage upload success with verified sizes and checksums.
5. Activates a WebDAV alias when the source is WebDAV.

For a new version of an existing file, preparation creates only the hidden
`StoredFile`; finalization creates the `FileVersion` link. The existing logical
file remains visible with its prior current version throughout. Preparation must
not rename the existing `File` or mutate any visible metadata. If a version flow
also requests a filename change, apply it atomically with the successful version
link during finalization.

This finalization-time link is mandatory. Do not rely on every list and detail
query independently remembering to hide an incomplete upload.

### Streaming Pipeline

Extend the shared upload-status mixin with nullable
`upload_last_progress_at`. New uploads set it at preparation and refresh it at a
throttled interval while raw bytes flow. Legacy rows without the field value fall
back to existing timestamps and are not treated as currently active uploads.

The pipeline must process raw content in this order:

```text
request/source reader
  -> raw byte limit and counter
  -> plaintext SHA-256
  -> gzip
  -> age encryption, when enabled
  -> transformed byte counter, SHA-256, and CRC32C
  -> S3 temporary object
```

Requirements:

1. Do not buffer or spool the complete raw or transformed file.
2. Propagate source-read, request-cancellation, pipe, gzip-close, age-close, S3,
   and context errors.
3. Close gzip, age, and pipes in the required order.
4. Treat any close error as upload failure even if S3 accepted an object.
5. Ensure every transform goroutine publishes exactly one terminal result so an
   S3 or transform failure cannot deadlock the request.
6. Cancel and join all upload goroutines before returning.
7. Record plaintext byte count and SHA-256.
8. Record transformed byte count, SHA-256, and CRC32C independently before S3
   receives those bytes.
9. Compare expected and actual plaintext bytes exactly when expected is known.
10. Enforce the existing maximum during streaming when expected size is unknown.
11. Reject zero plaintext bytes for every new upload source.

### S3 Temporary Object Verification

Unknown-size streams use multipart upload. Multipart SHA-256 is a checksum of
part checksums, not the full transformed SHA-256, and `UploadInfo.Size` is
client-counted. Therefore configure MinIO `PutObject` with
`ChecksumFullObjectCRC32C`, then issue `StatObject` with checksum retrieval after
upload.

Before finalization, require all of the following:

1. `PutObject` completed without error.
2. `StatObject` returned the backend's object size.
3. Backend size equals the locally counted transformed byte size.
4. `StatObject` returned CRC32C in `FULL_OBJECT` mode.
5. The normalized backend CRC32C equals the locally computed full-object CRC32C.

Persist local transformed SHA-256 as the canonical strong storage hash. The
backend CRC32C independently verifies transport and object persistence. A
multipart ETag or composite SHA-256 is not a substitute for full-object CRC32C.

If size or checksum is missing or mismatched:

1. Do not mark upload success.
2. Do not create or link a visible logical file.
3. Remove the temporary object immediately.
4. Leave `CleanupPending` state only if removal fails.
5. Return a generic storage-integrity failure.

Persist:

- plaintext count as `StoredFile.size`;
- plaintext SHA-256 as `StoredFile.content_sha256`;
- transformed count as `StoredFile.size_in_storage`;
- transformed SHA-256 as `StoredFile.sha256`;
- full-object CRC32C as a new optional `StoredFile.storage_crc32c` field.

Historical rows keep a null `storage_crc32c` and are not retroactively declared
corrupt. Strict CRC32C requirements apply to uploads started after this schema is
introduced. Existing successful rows already awaiting final copy retain their
legacy lifecycle so deployment cannot strand persisted data.

### Quota

Use plaintext bytes for tenant quota, matching current behavior.

- Preflight known sizes against file and tenant limits before streaming.
- Enforce the per-file maximum while streaming.
- Recheck quota with actual plaintext bytes in finalization to cover unknown
  lengths and concurrent uploads.
- A final quota failure deletes the temporary object and leaves no visible file.

### Account-Temporary Persistence

PWA and URL imports first write an account-temporary object and later re-encrypt
it into tenant storage. Both writes must use the same verified pipeline.

Cross-database persistence must be idempotent. Add a nullable, unique
`source_temporary_file_public_id` to tenant `StoredFile`. It records the main DB
TemporaryFile public ID and prevents a retry from creating a second tenant file.
Also store the current `source_conversion_claim_token` on that hidden StoredFile.
The durable main claim and progress time prevent expiry cleanup from removing an
object while conversion is active.

When persisting account temporary content:

1. In a short main transaction, create a random claim token with a conditional
   update that succeeds only when the file is unclaimed or its prior claim has
   made no progress for one hour. Record destination tenant/Space and suspend
   normal expiry.
2. In a short tenant transaction, query `StoredFile` by
   `source_temporary_file_public_id`.
3. If it is already successful, reuse its linked File and skip upload.
4. Otherwise, clean a stale row from an older claim and create one hidden
   StoredFile for the new claim before S3 I/O. Its random object name is unique
   to this claim.
5. Verify the account-temporary object's backend size and full-object CRC32C
   before reading it.
6. Recompute plaintext count and SHA-256 while decrypting it and require them to
   equal the account-temporary row's recorded plaintext values.
7. Stream into the claim's tenant temporary object, updating both progress
   heartbeats at a bounded interval, then verify that object.
8. Open short main-then-tenant finalization transactions and conditionally
   recheck the same claim token in both databases.
9. Reuse an already successful result or finalize that same hidden StoredFile,
   create the File and version once, and copy the immutable source.
10. Commit tenant finalization first, then mark the main TemporaryFile converted
    and clear its claim.

If the main commit fails after tenant commit, a retry finds the unique tenant
result and only completes the main marker. It must not create another file. Main
cleanup may remove the account object only after conversion is marked or the
idempotent tenant result is confirmed.

If a process with an obsolete claim resumes after takeover, its token checks fail
before finalization. It may clean only the random object and hidden row belonging
to its own token, never the replacement claim or a successful result.

### Temporary-to-Final Copy

The existing scheduler copies a successful temporary tenant object to final
storage. It must not trust an already existing destination object or a successful
copy call without integrity checks.

Before setting `copied_to_final_storage_at` or deleting the verified temporary
object:

1. Read destination size and full-object CRC32C through checksum-enabled
   `StatObject`.
2. Require both to equal the verified temporary object's stored size and
   `storage_crc32c`.
3. On mismatch or missing metadata, keep the temporary object, leave the file
   readable through it, and retry or report the copy failure.
4. Never switch reads to an unverified final object.

If server-side copy cannot preserve or report full-object CRC32C, stream the
already transformed temporary bytes through a checksum-enabled S3 PUT. Do not
decrypt, recompress, locally spool, or trust an unverified destination.

This rule applies to WebDAV and every existing upload source.

## Request and Transaction Flow

### PUT Phases

1. Validate HTTPS, Basic credentials, URL scope, current permissions, method, and
   DAV path.
2. Preflight known size, upload maximum, and current quota.
3. In a short tenant write transaction, release an obsolete alias if its file is
   no longer in Inbox and reserve `WebDAVResource(state=uploading)`.
4. Prepare hidden `StoredFile` upload state and attach it to the reservation.
5. Stream and verify the temporary S3 object outside a DB transaction.
6. Open the short main-then-tenant finalization transactions defined by the
   authentication boundary.
7. Recheck credential revocation, assignment expiry, tenant user, and Space
   access while the corresponding transactions serialize permission changes.
8. In the tenant transaction:
   - verify the reservation still belongs to the credential and Space;
   - recheck quota with actual bytes;
   - generate the unique Inbox filename;
   - create `File(source=webdav, is_in_inbox=true)`;
   - create `FileVersion`;
   - record verified `StoredFile` success;
   - set the DAV resource to `Active` and link the File.
9. Commit the tenant transaction, release the main transaction, and return `201`.

No DB transaction remains open while reading a request body or calling S3.

### Failure Rules

- Before object creation: release the reservation.
- During streaming: cancel pipes and S3 work, remove any partial object, mark
  hidden upload state failed, and release the reservation.
- After S3 success but before finalization: remove the verified temporary object
  if finalization fails.
- If object removal fails: retain cleanup state for scheduler retry.
- After finalization but before the response reaches the client: keep the valid
  file and active alias. A client retry receives `409`, preventing duplication.
- A failure response must never cause deletion of a previously finalized file.

### Existing HTTP Ingress

Shared hardening includes the HTTP ingress, not only `S3FileSystem`. The current
browser form helper calls `ParseMultipartForm` and can spool content to disk, and
the normal router keeps read transactions open for the complete handler. Upload
and URL-import routes must stop using those paths.

Add the smallest dedicated streaming-action wrapper that:

1. Authenticates and snapshots route identity through short read transactions.
2. Closes those transactions before request-body or outbound network I/O.
3. Uses `Request.MultipartReader` instead of `ParseMultipartForm` or `FormFile`.
4. Streams each file part directly into the shared upload pipeline.
5. Opens only short preparation and finalization transactions.

Place destination control fields before the file part, or move them to validated
query/header metadata controlled by the existing upload widget, so no file part
must be spooled while waiting for form fields. PWA multi-file requests process
parts sequentially with the same per-file guarantees. URL import uses the same
transaction-free orchestration around its outbound HTTP stream.

## `x/net/webdav` Integration

Use the library for DAV parsing, dispatch, properties, conditional locks, and
lock tokens, but do not expose its defaults without an outer boundary.

### Outer Endpoint

An outer `http.Handler` owns:

- route extraction;
- HTTPS and Basic authentication;
- current context construction;
- method and path gating;
- request-size preflight;
- `OPTIONS`;
- 64 KiB limits for PROPFIND and LOCK XML bodies;
- sanitized logging;
- request operation values passed through `context.Context`;
- final status mapping and response-header filtering.

The normal browser router and its unauthenticated redirects are not used.

### Inner Handler and Filesystem

The inner `webdav.Handler` uses:

- a custom write-only `webdav.FileSystem`;
- the credential-namespaced view of the bounded server lock system;
- a logger that never includes Authorization or file content.

The custom filesystem behaves as follows:

- `Stat` returns synthetic directory info for `/` and `/Inbox/`.
- `Stat` hides file paths from PROPFIND, GET, and HEAD.
- Request-context-gated PUT and MOVE internals may see synthetic active-resource
  info required by `x/net/webdav`.
- Structural `OpenFile` returns a synthetic directory. Root `Readdir` returns
  only Inbox; Inbox `Readdir` returns no children.
- PUT `OpenFile` returns a streaming file backed by the shared upload pipeline.
- LOCK `OpenFile` returns the no-op placeholder.
- Read `OpenFile` for a file returns `os.ErrNotExist`.
- `Mkdir` and `RemoveAll` always reject.
- `Rename` implements only the narrow MOVE transaction.

The pinned library calls PUT `File.Stat` before `File.Close`, then uses that
pre-close `FileInfo` to build an ETag after close. It does not call filesystem
`Stat` again. The streaming file must therefore:

1. Report only synthetic in-request size before close.
2. Perform transform completion, S3 verification, permission recheck, and DB
   finalization in `Close`.
3. Return finalization errors through a request operation result.
4. Let the outer handler replace the library's generic `405` with the defined
   `400`, `403`, `409`, `413`, `507`, or `500` response.

Use a bounded recording response writer for the small DAV response when status
remapping is required. It must never buffer upload content.

Do not revive the obsolete, commented `webdav/dir.go`; it predates current
contexts, storage, authorization, and transaction boundaries.

## Scheduler Reconciliation

Extend the existing scheduler rather than adding a worker service.

An upload is stale after one hour without a persisted progress heartbeat, not
merely one hour after it started. The streaming path updates heartbeat at a
throttled interval so a slow but progressing upload is not reclaimed.

### Stale Upload State

In bounded batches, find hidden `StoredFile` uploads and WebDAV resources whose
last progress is older than one hour and which have no success marker.

For each stale upload:

1. Confirm it has not become successful since selection.
2. Claim cleanup atomically so concurrent scheduler passes cannot both clean it.
3. Abort an incomplete multipart upload when supported.
4. Remove its temporary object if present.
5. Mark or delete failed hidden storage state according to existing cleanup
   conventions.
6. Delete its DAV resource after cleanup succeeds.
7. Retain `CleanupPending` and retry later if S3 cleanup fails.

### Stale Account-to-Tenant Claims

In bounded batches, find main `TemporaryFile` conversion claims with no progress
for one hour.

1. Replace the stale claim token with a scheduler cleanup token through a main DB
   compare-and-swap, then commit.
2. Open the recorded tenant DB and query `StoredFile` by the TemporaryFile public
   ID.
3. If the tenant result is successful, mark the main TemporaryFile converted and
   clear the claim. Never delete the tenant object or File.
4. If an unfinished tenant row belongs to the stale token, claim its cleanup in a
   short tenant transaction, remove its random object outside all transactions,
   then delete/mark the hidden row and clear the main claim with short
   transactions.
5. If no tenant row exists, clear the main claim and restore normal expiry.
6. If S3 cleanup fails, retain the scheduler token and retry later.

All token changes are conditional. A scheduler pass must not clean a newer claim,
and no main or tenant transaction remains open during S3 calls.

### Active Alias Release

Delete active DAV resources whose linked file is deleted, missing, in another
Space, or no longer in Inbox. MOVE also checks the linked file synchronously, so
delayed scheduler cleanup never restores rename permission.

### Orphan Temporary Objects

In bounded S3 listing batches, inspect tenant and account temporary prefixes for
objects older than one hour. Delete an object only when no current row references
it as:

- an unfinished upload with recent progress;
- a successful upload awaiting final copy;
- an account `TemporaryFile` still in its normal lifecycle;
- a preview/conversion or other existing temporary workflow artifact.

Never apply orphan cleanup to the final storage prefix. Never delete a successful
file's only verified readable object.

### Crash Windows

| Crash point | Recovery |
| --- | --- |
| Before reservation | No upload state exists. |
| After DAV reservation | Stale reservation is removed after one hour without progress. |
| After hidden storage prepare | Hidden row and reservation are reconciled. |
| During streaming | Partial object or multipart upload is aborted/removed. |
| After S3 verification, before finalization | Unlinked temporary object and hidden row are removed. |
| During finalization | Transaction rollback leaves only hidden recoverable state. |
| After finalization, before HTTP response | Valid file remains; retry receives `409`. |
| During account-to-tenant conversion | Claim-token recovery reuses success or cleans only its hidden attempt. |
| After tenant conversion commit, before main commit | Retry finds the unique successful tenant result and marks main converted. |
| During final S3 copy | Verified temporary object remains readable until destination verification. |
| After verified final copy, before temp deletion | A later pass removes the duplicate temporary object. |

## UX Requirements

### WebDAV Credentials Page

Add a dedicated WebDAV credentials page. Its navigation-rail item appears
between Account and System. List the current account's credentials across
tenants and Spaces in a list/table view rather than cards.

Each row/card shows:

- label;
- tenant and Space;
- username;
- created time;
- last used time, when available;
- revoked time/state;
- revoke action for active credentials.

Expose the create action as a FAB and as the first list item in every accessible
Space tab. The tab action preselects its Space. The form asks for an accessible
Space and device label; each Space option identifies its tenant. The success
dialog shows the complete URL, username, and secret once, with clear copy
actions and a warning that the secret cannot be recovered.

When no credentials exist, show an empty state. Otherwise, group credentials by
Space in tabs and show the complete WebDAV URL once in a docked toolbar per
active Space tab. Only the URL text is a copy link. The link explains its copy
action on hover and confirms the copy through a snackbar. Put revocation in each
active credential row's overflow menu.

Add a filter icon to the main app bar. It opens the standard filter side sheet
with Active and Revoked checkbox chips. Show only active credentials by default;
allow either status or both statuses to be selected.

If a referenced Space or tenant is no longer accessible, retain the credential
record, show the destination as unavailable without leaking protected names, and
allow the owner to revoke it.

### Tenant User Management

For a selected tenant user, tenant owners can inspect:

- label;
- Space;
- username;
- created, last-used, and revoked times;
- revoked state.

They may revoke an active credential. They may not create one, reveal a secret,
or replace a secret.

### States and Accessibility

- Use the dedicated WebDAV credentials destination, existing lists, and dialogs.
- Keep the primary create FAB and one Space-scoped add row per tab.
- Confirm revocation and report success through existing snackbar/event flows.
- Preserve keyboard focus when HTMX refreshes a list.
- Give URL, username, secret, create, copy, and revoke controls explicit labels.
- Do not encode source or credential state by color alone.

## Security and Reliability

- Require HTTPS before accepting Basic credentials outside development.
- Authenticate every DAV method.
- Never accept a browser session cookie in place of a WebDAV credential.
- Do not use client-provided tenant or Space IDs without matching credential
  scope and current privacy-filtered queries.
- Recheck authorization before finalization.
- Use generated high-entropy credentials, salted hashes, constant-time
  comparison, and bounded failed-auth rate limiting.
- Never log Authorization, secrets, request bodies, file content, or plaintext
  hashes at normal log level.
- Sanitize user-controlled paths in logs.
- Bound body size during streaming even without `Content-Length`.
- Respect cancellation and server shutdown without finalizing partial content.
- Keep compression and age encryption behavior identical across upload sources.
- Fail closed on missing or mismatched storage integrity metadata.
- Keep the verified temporary object until final storage is verified.
- Do not add a distributed lock, local spool directory, or new storage format.

## Implementation Guidance

Likely additions and focused changes:

- `db/entmain/schema/webdav_credential.go`
- `db/enttenant/schema/webdav_resource.go`
- `db/entmain/schema/temporary_file.go`
- `db/enttenant/schema/stored_file.go`
- Tenant and Space schemas to make endpoint public IDs immutable
- a shared file-source enum under the existing common model hierarchy
- a WebDAV endpoint and custom filesystem under `server` plus one focused model
  package
- a transaction-free streaming-action router path for existing upload ingress
- `model/tenant/filesystem/s3_file_system.go`
- `model/tenant/filesystem/prepared_upload.go`
- `db/entx/upload_status_mixin.go` or equivalent upload-heartbeat state
- existing browse, Inbox, version, PWA, URL import, and extraction commands
- `action/inbox/files_list_partial.go` and file details rendering
- account settings and tenant user management actions/partials
- existing scheduler file-processing and cleanup loops

Reuse existing Ent privacy rules, `filenamex`, upload limits, quota helpers,
`PreparedUpload`, and `uploadx` cleanup behavior where they still satisfy this
specification. Do not reuse panic-based context constructors or account checks
that ignore assignment expiry at the WebDAV trust boundary.

Do not:

- edit generated Ent code or migration files manually;
- hold a DB transaction across network I/O;
- introduce an interface with only one speculative implementation;
- add another S3 client or upload format;
- create a package-level lock or credential registry;
- add a user-facing CLI for v1.

## Testing Requirements

### Unit Tests

- DAV path parsing, exact Inbox casing, traversal, encoded separators, nested
  paths, and filename validation.
- Method gate and narrow `Allow` header.
- Basic challenge, secret verification, fake-hash path, revoked credentials, and
  endpoint-scope mismatch.
- Source enum parsing, default, labels, and repeated URL parameters.
- Extension-aware unique filename generation.
- DAV resource reservation, conflict, activation, MOVE, release, heartbeat, and
  cleanup states.
- Known-size exact match and mismatch.
- Unknown-size bound, overflow, and zero-byte rejection.
- Source read, cancellation, gzip-close, age-close, pipe, and S3 errors.
- Missing/mismatched backend CRC32C, checksum mode, and size.
- Local plaintext SHA-256 plus transformed SHA-256, CRC32C, and counts.
- Lock XML/body limit, per-credential count, timeout cap, expiry, and namespacing.

### Protocol Tests

Use `httptest` with raw DAV XML and headers. Cover representative request
sequences without naming client products:

1. Unauthenticated `OPTIONS`, challenge, then authenticated retry.
2. `PROPFIND /` with Depth 0 and 1.
3. `PROPFIND /Inbox/` with Depth 1 and infinity, returning no files.
4. Direct `PUT /Inbox/a.pdf`.
5. `LOCK`, PUT with lock token, then `UNLOCK`.
6. PUT to a temporary name followed by MOVE to a final name.
7. Chunked PUT without `Content-Length`.
8. PUT followed by file PROPFIND, HEAD, and GET, all revealing no resource.
9. Repeated PUT of an active DAV path returning `409` before body consumption.
10. MOVE after Inbox processing returning conflict/forbidden.
11. Rejected `MKCOL`, nested PUT, DELETE, COPY, and PROPPATCH.
12. Rejected cross-host and cross-Space MOVE destination.
13. Process lock loss followed by successful lock reacquisition.
14. Structural HEAD rejected and omitted from `Allow`.
15. MOVE with `Overwrite: T` never calls `RemoveAll` or changes a destination.
16. Oversized LOCK and PROPFIND XML rejected before DAV parsing.

### Integration Tests

- A successful WebDAV upload creates exactly one Inbox File, one version, one
  verified StoredFile, `source=webdav`, and one active DAV alias.
- No logical file is visible before finalization.
- Lost permission or revocation between streaming and finalization fails and
  removes staged storage.
- Barrier-controlled revocation and Space-removal races serialize against
  finalization in the defined main-then-tenant order.
- Missing/deleted tenant users and expired assignments return controlled errors,
  not panics.
- Same credential/path concurrency creates one file and one conflict.
- Same DMS filename concurrency generates unique names without overwrite.
- Different credentials can ingest the same DAV path independently.
- MOVE renames only the same credential's still-Inbox file.
- MOVE destination name conflict generates a unique DMS name.
- Browser, PWA, URL import, WebDAV, and extraction assign the expected source.
- PWA and URL source survive account-temporary persistence.
- Account-temporary retry after tenant commit/main commit failure reuses the
  idempotent tenant result and creates no duplicate.
- A failed new-version upload changes neither source, name, nor visible metadata.
- Historical rows default to `UnknownLegacy`.
- Source filter ORs selected values and ANDs search/other predicates.
- List, table, and file details render source.
- Query plans remain bounded for source-filtered Inbox sorts.

### Storage and Recovery Tests

- Known-length truncation fails.
- Request cancellation fails and cleans up.
- Transform closure failure cannot finalize.
- Missing, mismatched, malformed, or non-full-object S3 CRC32C fails closed.
- Backend `StatObject` size mismatch fails closed.
- Quota race at finalization returns `507` and cleans up.
- S3 outage and slow/disconnected upload leave no visible file.
- A progressing upload older than one hour is not reclaimed.
- An upload with no progress for one hour is reclaimed.
- Stale multipart upload and temporary object cleanup are idempotent.
- Orphan cleanup does not delete referenced account temporary, tenant temporary,
  preview, or successful objects.
- Account-to-tenant re-encryption verifies plaintext identity.
- Stale account-to-tenant claims are taken over by compare-and-swap, and an old
  worker cannot finalize or clean a replacement claim.
- A successful tenant conversion with a missing main marker is reused and marks
  main converted without another File.
- Temp-to-final copy verifies destination size/full-object CRC32C before
  switching reads.
- A corrupt existing final destination is never trusted.
- A successful finalized file survives every cleanup path.
- Browser and PWA multipart ingress does not call `ParseMultipartForm`, create a
  local spool file, or hold a DB transaction while streaming.

### Security Tests

- WebDAV failures never redirect to sign-in.
- Cleartext requests are rejected before secret verification.
- Invalid and revoked credentials produce the same Basic challenge.
- A valid credential cannot access another tenant or Space URL.
- Tenant-assignment and Space-assignment loss apply on the next request.
- Tenant owners can revoke but cannot reveal or create user credentials.
- Credential owners cannot inspect another account's records.
- Failed-auth rate limiting is bounded and does not log secrets.
- Logs exclude Authorization and uploaded content.

### Browser Tests

- Account settings list credentials across Spaces.
- Creation shows the secret once and uses a no-store response.
- Reloading never shows the secret.
- The user can revoke an owned credential.
- Tenant user management permits owner inspection/revocation only.
- Source appears in Inbox list, table, and file details.
- Multi-source filtering updates the URL and server-rendered partial.
- Browser back/forward restores source selection.

## Acceptance Criteria

The feature is complete when all of the following are true:

1. A user can create multiple named, one-Space WebDAV credentials for Spaces the
   account currently accesses.
2. Each generated secret is shown once, sent only over HTTPS outside development,
   and stored only as a salted hash.
3. Users and tenant owners can revoke credentials according to the defined
   ownership rules; revoked metadata remains but cannot authenticate.
4. WebDAV uses a Basic challenge and never redirects to browser sign-in.
5. Current tenant and Space permissions are enforced on every request and before
   upload finalization.
6. Structural discovery exposes `/Inbox/` and no user files.
7. Existing and uploaded files cannot be listed, read, overwritten, deleted,
   copied, nested, or edited through WebDAV.
8. A valid PUT streams without full-file disk or memory buffering.
9. Zero-byte, oversized, truncated, canceled, transform-failed, quota-failed, and
   S3-integrity-failed uploads leave no visible logical file.
10. Backend S3 temporary size and full-object CRC32C match local transformed
    count and CRC32C before finalization; local full SHA-256 is persisted.
11. A successful PUT creates one Inbox file with source `webdav` only after
    verification and commit.
12. A repeated same-credential PUT to the active DAV path returns `409` without a
    duplicate or overwrite.
13. Inbox filename conflicts generate concurrency-safe unique names.
14. LOCK before PUT never creates a zero-byte file.
15. MOVE only renames the same credential's linked file while it remains in
    Inbox.
16. Browser, PWA/open-with, URL import, WebDAV, extraction, and historical source
    values are assigned as specified.
17. Source is visible in both Inbox layouts and file details, and the URL-backed
    multi-select filter composes with existing Inbox state.
18. Account-temporary persistence and tenant temp-to-final copy retain verified
    byte identity.
19. Immediate cleanup and one-hour no-progress reconciliation are idempotent and
    never delete a successful file's only verified object.
20. Automated protocol tests cover common desktop, scanner/MFP, mobile, and CLI
    request patterns without publishing named-client guarantees.

## Tradeoffs and Rejected Alternatives

- **Consume-only WebDAV instead of full file management:** matches the ingestion
  goal and avoids exposing the document tree.
- **Generated Basic credentials instead of account passwords or sessions:** works
  with non-browser devices, supports per-device revocation, and keeps account
  login factors separate.
- **Main DB credential plus Space public ID instead of a cross-DB edge:** allows
  authentication before a tenant connection exists.
- **Temporary DAV alias instead of a readable DAV file:** supports client retry
  conflicts and final-name MOVE without pretending WebDAV is durable storage.
- **In-memory locks instead of persistent locks:** sufficient for the current
  single-process SQLite deployment and irrelevant to stored-byte durability.
- **`x/net/webdav` behind a restrictive adapter instead of hand-written DAV:**
  reuses Go-maintained parsing and lock behavior without exposing unsafe default
  methods.
- **Shared integrity hardening instead of WebDAV-only checks:** fixes corruption
  boundaries once for every ingestion caller.
- **Finalization-time logical file creation instead of query-time hiding:** makes
  partial-file invisibility one invariant rather than a convention in every
  query.
- **No full readback after PUT:** local plaintext/transformed hashing plus
  checksum-enabled backend StatObject size and full-object CRC32C verifies the
  stream without doubling transfer and latency.
- **No new service or CLI:** the monolith, existing scheduler, browser settings,
  and standard WebDAV clients cover v1.

## Open Questions

No blocking product questions remain.

Engineering choices left to implementation are:

1. Exact generated username and secret format and length, provided the secret has
   high entropy and both are Basic-auth-safe.
2. Exact failed-auth rate-limit window and bounded storage strategy.
3. Exact structural DAV property subset needed by the automated request patterns.
4. Exact package names for the endpoint, filesystem adapter, and DAV resource
   model.
5. Whether the `PutObject` response is checked in addition to mandatory
   checksum-enabled `StatObject`; backend size and full-object CRC32C must match
   local values in either case.
