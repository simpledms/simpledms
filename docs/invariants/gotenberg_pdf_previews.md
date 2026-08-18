# Gotenberg PDF Preview Conversion Invariants

## Scope

These invariants apply to the asynchronous conversion of eligible tenant file
versions into hidden PDF preview artifacts, the scheduler and storage lifecycle,
source-scoped preview/download routes, and the Preview/Original UI.

The canonical original remains owned by the existing File context. Preview
conversion is supporting functionality and must not change document identity,
version history, access control, or upload availability.

## Source Preservation

### Original Is Canonical

Rule: The uploaded original `StoredFile` and its `FileVersion` remain the
canonical document content.

Why: The PDF exists only to improve preview and must never cause data loss or
change what users downloaded as the original.

Enforced in: upload finalization, conversion storage, conversion failure paths,
and source-scoped download handlers.

### Conversion Never Blocks Upload

Rule: Upload completion must not wait for Gotenberg and must not fail because
Gotenberg is missing, unavailable, slow, or unable to read the source.

Why: An optional preview service must not become an upload dependency.

Enforced in: upload commands, scheduler discovery, and the Gotenberg client
boundary.

### Conversion Does Not Mutate Source Metadata

Rule: Conversion must not change the source filename, MIME type, content hash,
size, storage object, or version relationship.

Why: These values describe the original bytes and are not properties of the
derived preview.

Enforced in: derived-artifact persistence and conversion state transitions.

## Eligibility And Routing

### Explicit Eligibility

Rule: Only sources identified by the explicit supported MIME/extension allowlist
are converted.

Why: Sending arbitrary files to LibreOffice is unpredictable and makes preview
state unclear.

Enforced in: the pure eligibility classifier and its unit tests.

### MIME Or Extension Is Sufficient

Rule: A known supported extension or a known supported MIME type makes a source
eligible. MIME and extension matching is not required to agree.

Why: Uploaded files can have generic or inaccurate MIME detection, while a
trusted filename extension can still identify the intended converter family.

Enforced in: normalized classifier input and canonical MIME-only filename
mapping.

### Supported Families Are Bounded

Rule: The first implementation converts HTML, Markdown, and office-family
documents only. Plain text, images, graphics, PDFs, archives, directories, and
unknown formats remain original-only.

Why: The feature has a deliberate scope even where a Gotenberg module accepts a
broader set of formats.

Enforced in: allowlist, scheduler discovery, UI artifact lookup, and classifier
tests.

### Route Mapping Is Stable

Rule: HTML uses `/forms/chromium/convert/html`, Markdown uses
`/forms/chromium/convert/markdown`, and office-family documents use
`/forms/libreoffice/convert`.

Why: Route selection is part of the Preview Conversion boundary and must not be
duplicated across scheduler callers.

Enforced in: the concrete Gotenberg adapter and route construction tests.

### No Direct URL Conversion

Rule: SimpleDMS never calls Gotenberg's direct remote URL conversion route.

Why: Existing URL import already downloads and stores a response, and adding a
second remote fetch path would increase SSRF and lifecycle complexity.

Enforced in: adapter route set and URL-import integration tests.

### URL Imports Become Ordinary Sources

Rule: A file imported through the existing URL flow is converted only after its
downloaded response is stored, and only if that stored source is eligible.

Why: Conversion must operate on the same canonical bytes and storage lifecycle
regardless of upload origin.

Enforced in: scheduler discovery rather than URL-import handler coupling.

## Preview Artifact Identity And Visibility

### One Source Conversion Identity

Rule: The logical conversion identity is the source `StoredFile`. A source has
at most one conversion record and at most one active ready preview artifact.

Why: A unique source identity prevents duplicate scheduler work and duplicate
PDF objects after retries or process restarts.

Enforced in: unique source index, atomic claim, and idempotent discovery.

### Reused Stored Files May Share Previews

Rule: If multiple file versions refer to the same original `StoredFile`, they may
reuse that source's one preview artifact.

Why: The source bytes are identical, so re-converting them would waste storage
and external conversion capacity.

Enforced in: source-stored-file conversion lookup and version-scoped route
resolution.

### Preview Is Not A Document Version

Rule: A derived PDF must not be linked into `FileVersion`.

Why: Generated previews must not appear in document history, file listings,
duplicate detection, or normal document navigation.

Enforced in: derived `StoredFile` creation and conversion schema edges.

### PDF Artifact Properties

Rule: A ready artifact has MIME type `application/pdf`, a source-based filename
with `.pdf`, a recorded uncompressed byte size, and the existing tenant storage
protection.

Why: The artifact must be rendered and downloaded as a PDF while using the same
compression, encryption, hashing, and tenant isolation rules as originals.

Enforced in: derived storage methods and PDF response validation.

### Ready Means Final And Readable

Rule: A conversion cannot become `ready` until the derived object is at its
final storage location and can be opened/read through the tenant filesystem.

Why: A database row must never advertise a preview that is still temporary,
partial, or inaccessible after a scheduler restart.

Enforced in: final-copy coordination, ready transition, and integration tests.

### Partial Artifacts Are Not Visible

Rule: Failed or interrupted attempts must not expose a partial PDF and must clean
partial derived rows and storage objects.

Why: A corrupt or incomplete PDF is worse than the original-only fallback and
can leave unbounded orphan storage.

Enforced in: conversion failure handling, stale-claim recovery, and cleanup
tests.

## Conversion State And Scheduler

### State Values

Rule: A conversion record uses only these persisted states: `pending`,
`processing`, `ready`, and `failed`.

Why: These states cover work waiting, active work, usable output, and terminal
failure without persisting environment-specific configuration state.

Enforced in: the Ent enum and state-transition methods.

### Ineligible Sources Have No Conversion Record

Rule: An excluded or unknown source does not receive a conversion record or a
disabled fake artifact.

Why: Original-only behavior should remain simple and should not imply that a PDF
is expected for a file outside the feature scope.

Enforced in: classifier and scheduler discovery.

### Missing Configuration Is Runtime Disabled

Rule: An empty or invalid stored Gotenberg URL disables conversion and logs
a warning or error, but does not mark source uploads failed.

Why: Configuration availability is an application deployment condition, not a
document conversion result.

Enforced in: scheduler startup wiring and disabled preview rendering.

### Final Sources Only

Rule: The scheduler sends only completed source objects available at final
storage. It does not send unfinished or temporary-only uploads.

Why: Conversion must read stable canonical bytes and must not race upload
finalization or temporary-object cleanup.

Enforced in: discovery predicates and final-storage checks.

### Backfill Is The Normal Scan

Rule: The same scheduler scan discovers existing eligible historical versions,
new uploads, and new versions. There is no separate backfill command or admin
workflow.

Why: One idempotent path avoids divergent behavior and avoids a one-time command
that can be forgotten after deployment.

Enforced in: scheduler query and unique conversion record creation.

### Historical Versions Are Included

Rule: Discovery processes eligible historical versions, not only current file
versions.

Why: Version preview must work for the exact historical original selected by the
user.

Enforced in: source stored-file/version queries and version preview tests.

### Claims Are Atomic

Rule: A pending conversion is atomically claimed as `processing` before an
external request begins.

Why: Repeated scheduler passes or future concurrent workers must not convert the
same source simultaneously.

Enforced in: conditional Ent update and duplicate-conversion tests.

### Stale Claims Recover

Rule: A `processing` record older than the bounded recovery interval returns to
`pending` or is safely reconciled with an already stored artifact.

Why: Process termination must not permanently strand a source.

Enforced in: scheduler recovery pass and restart integration tests.

### Ready Transition Is Idempotent

Rule: Reprocessing or repeated finalization cannot create a second active ready
artifact for the same source.

Why: Network retries and process restarts can happen after an object write but
before the database update.

Enforced in: unique source relation, artifact reconciliation, and storage
cleanup.

### Single In-Flight Conversion Initially

Rule: One application process runs at most one conversion request at a time in
the initial implementation.

Why: LibreOffice conversion is serialized and the existing scheduler does not
need speculative concurrency machinery.

Enforced in: scheduler loop structure and worker tests.

## Retry Invariants

### Retry Count Is Durable

Rule: Attempt count and next eligible attempt time are persisted in the tenant
database.

Why: A scheduler restart must not reset retry limits or hammer Gotenberg.

Enforced in: conversion record updates and scheduler restart tests.

### Automatic Retry Limit

Rule: A conversion gets the initial attempt plus at most three automatic retries
for transient failures.

Why: Temporary service failure should recover, but permanently unreadable files
must not consume scheduler capacity forever.

Enforced in: retry classifier, persisted count, and retry tests.

### Retry Classification

Rule: Network errors, timeouts, service-unavailable responses, and Gotenberg 5xx
responses are retryable. Document/content failures and relevant 4xx responses
are terminal for the automatic cycle.

Why: Retrying a malformed source does not fix it, while service failures may
recover.

Enforced in: Gotenberg adapter response classification.

### Manual Retry Is Viewer Authorized

Rule: After automatic retries are exhausted, any user who can view the source
file may reset its failed conversion to `pending`.

Why: Retry is a preview operation, and the source file's access policy is the
correct authorization boundary.

Enforced in: Retry command source lookup and HTTP integration tests.

### Retry Does Not Mutate The Original

Rule: Manual or automatic retry changes only conversion state and derived
artifact work. It never changes the original or creates a visible version.

Why: Retry must be safe to expose to viewers.

Enforced in: command handler, conversion service, and source preservation tests.

## Storage And Quota

### Derived Bytes Count

Rule: A ready derived PDF's `StoredFile.size` contributes to tenant storage
usage.

Why: The PDF is durable tenant data even though it is hidden from document
navigation.

Enforced in: existing storage quota aggregation and quota integration tests.

### Conversion May Create Overage

Rule: Conversion does not reject or undo an already successful original upload
because storing the PDF would exceed quota.

Why: Optional preview generation must not cause data loss after upload.

Enforced in: derived persistence path, not the normal upload quota guard.

### Later Uploads Observe Overage

Rule: Normal future uploads continue to use existing tenant quota checks and may
be blocked while derived bytes leave the tenant over quota.

Why: Counting derived storage must remain consistent with existing quota policy.

Enforced in: storage quota tests covering original and derived rows.

### Same Storage Protection

Rule: Derived PDFs use the same tenant bucket/path isolation, encryption,
compression, hashes, temporary storage, and cleanup model as originals.

Why: Hidden does not mean less protected.

Enforced in: S3 filesystem methods and cross-tenant storage tests.

## Access And Security

### Source-Scoped Authorization

Rule: Preview status, inline PDF, PDF download, original source view, and Retry
all authorize through the source file, tenant, Space, and version context.

Why: The derived artifact has no independent user-facing identity and must not
become an access-control bypass.

Enforced in: route handlers and source-scoped repository lookups.

### No Artifact-ID Access

Rule: Public URLs and form data contain public file identifiers and optional
version numbers only. Internal preview artifact IDs are never a client security
boundary.

Why: A leaked internal artifact identifier must not provide cross-space access.

Enforced in: route construction, request binding, and authorization tests.

### Safe HTML Original Inspection

Rule: The HTML Original tab displays escaped source text with a text content type
and no-sniff behavior. Uploaded HTML must not execute in the SimpleDMS origin.

Why: Raw HTML is untrusted active content.

Enforced in: dedicated source-inspection route, response headers, and browser
security tests.

### Converter Isolation

Rule: SimpleDMS does not pass user cookies, credentials, or application
authorization headers to Gotenberg. Gotenberg Chromium and LibreOffice deny
private/link-local outbound destinations.

Why: Uploaded content and its external references must not reach internal
services or carry user secrets.

Enforced in: adapter request construction, Compose configuration, and security
configuration tests.

### Error Detail Boundary

Rule: Users see stable friendly conversion messages. Detailed Gotenberg response
bodies and traces stay in logs and are not stored as user-facing document data.

Why: Converter errors can contain implementation details or source-derived
content.

Enforced in: failure-category mapping, UI rendering, and logging tests.

## Preview UI And HTMX

### Exact Source Version

Rule: Original and Preview tabs in the main preview and version dialog refer to the
same exact source version.

Why: A historical version must never display the current version's preview.

Enforced in: version-aware source lookup, route parameters, and browser tests.

### Preview Is Default

Rule: An eligible source selects `Preview` by default when no explicit
preview-tab state exists, including while conversion is pending or failed.

Why: Preview is the primary reading view and owns its status, while explicit
Original selection must persist.

Enforced in: preview state parsing and tab composition.

### Original Remains Usable

Rule: Pending, disabled, failed, and unavailable conversion states never disable
or hide the Original tab or original download.

Why: The original is the guaranteed fallback.

Enforced in: preview composition and failure-state tests.

### Preview Status Content

Rule: Pending, processing, unavailable, and failed states are rendered inside
the selectable Preview tab content. The Original tab contains only the source.

Why: Users need to understand that conversion is asynchronous without waiting
for it to finish before using the file.

Enforced in: shared preview composition and status fragment rendering.

### Failed PDF Tab

Rule: A terminal failed conversion keeps the Preview tab visible with a friendly
failure state and viewer-authorized Retry action.

Why: Users need recovery without losing access to the original.

Enforced in: status fragment, Retry command, and browser tests.

### Polling Is Fragment-Scoped

Rule: HTMX polling refreshes only the stable preview status/content fragment and
stops at ready or terminal failure.

Why: Polling must not reset side-sheet tabs, preview selection, or unrelated file
details.

Enforced in: partial target/swap design and browser polling tests.

### Preview State Is Separate

Rule: Main Original/Preview selection uses a separate URL state field from existing
metadata side-sheet tabs.

Why: Reusing `tab` would make unrelated tab state collide and would cause stale
state to select the wrong content.

Enforced in: state structs, route generation, and back/forward tests.

## Lifecycle

### New Versions Have Independent Conversion Lifecycles

Rule: Uploading a new version starts a new conversion lifecycle and cannot
overwrite an earlier version's original or preview.

Why: Version history must remain immutable and independently previewable.

Enforced in: source stored-file identity, version-aware lookup, and integration
tests.

### Rename Does Not Reconvert

Rule: Renaming a file does not change conversion identity or source bytes. The
derived filename is based on the source version's original filename.

Why: A name change is not a content change.

Enforced in: source `StoredFile` identity and download filename construction.

### Soft Delete Supports Restore

Rule: Soft deletion must retain enough source and preview state for restore.

Why: Restoring a file should not require unnecessary reconversion.

Enforced in: soft-delete queries and trash restore behavior.

### Permanent Cleanup Is Complete

Rule: Permanent source cleanup removes the conversion record, derived
`StoredFile` row, and derived storage object together.

Why: No inaccessible preview may remain as an orphan or continue consuming
quota.

Enforced in: file cleanup workflow and object/row cleanup tests.

## Deployment And Availability

### Optional Configuration

Rule: An empty or invalid stored Gotenberg URL logs a warning/error and
disables conversion without application startup failure.

Why: Deployments may intentionally run without the optional service.

Enforced in: server scheduler wiring and Compose/env tests.

### Compose Network Boundary

Rule: Development and sample Compose include Gotenberg, but the sample stack
does not publish Gotenberg externally.

Why: SimpleDMS should reach the converter over the internal service network only
in deployments.

Enforced in: both Compose files and deployment smoke checks.

### Converter Outage Degrades Gracefully

Rule: Gotenberg outage, timeout, or terminal conversion failure degrades to
original-only preview and does not stop the scheduler's other loops.

Why: Mail, cleanup, upload processing, OCR, and normal file access must not be
coupled to the optional converter.

Enforced in: recoverable scheduler loop, retry handling, and outage integration
tests.
