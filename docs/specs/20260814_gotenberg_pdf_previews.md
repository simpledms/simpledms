# Gotenberg PDF Preview Conversion

Status: Proposed

Date: 2026-08-14

## Business Outcome

SimpleDMS must keep the uploaded file as the canonical original while creating a
PDF representation for easier, more consistent previewing. Conversion must not
slow down or block upload completion. Users must be able to inspect the original
when the browser can display it and use the generated PDF when it is the better
reading format.

The feature is supporting functionality around the existing file and version
model. It does not create a second user-visible document version.

## Confirmed Decisions

- Convert newly completed uploads, newly uploaded versions, and existing
  eligible historical versions.
- Include files created through the existing Import URL flow after the response
  has been downloaded and stored. Do not call Gotenberg's direct URL route.
- Keep one hidden, durable PDF preview artifact for each source file version.
- Keep the original file unchanged and use it for the normal original download.
- Count generated PDF bytes in tenant storage usage, but allow conversion to
  create quota overage after the original upload has succeeded.
- Use Preview as the default tab for every eligible source state.
- Add Preview/Original tabs to the main preview and historical version preview.
- Allow downloading both the original and the generated PDF.
- Keep pending and failed conversion state visible while keeping the original
  usable.
- Retry transient failures automatically three times, then provide a Retry
  action to any user who can view the source file.
- Show friendly user-facing errors and keep detailed converter errors in logs.
- If the stored Gotenberg URL is empty, log a warning and disable conversion
  without preventing normal application use.
- Add Gotenberg to both development and sample Docker Compose files.

## Ubiquitous Language

| Term | Meaning |
| --- | --- |
| Original | The uploaded bytes represented by the existing `StoredFile`. |
| Source version | A `FileVersion` and its linked original `StoredFile`. |
| Preview artifact | A derived PDF stored with the existing tenant storage and encryption rules. It is not a `FileVersion` and is not listed as a document. |
| Conversion | The asynchronous operation that sends one original to a Gotenberg route and stores the returned PDF. |
| Eligible source | An original whose trusted filename extension or detected MIME type maps to a supported conversion family. |
| URL import | The existing flow that downloads a URL response and stores it as an ordinary upload. It is not a remote-page conversion request. |

## Scope

### In Scope

- HTML files converted through Chromium HTML conversion.
- Markdown files converted through Chromium Markdown conversion.
- Office-family documents converted through the LibreOffice conversion route.
- Automatic discovery of existing and newly completed eligible source versions.
- Durable conversion status, retries, and user-triggered retry.
- Tenant-authorized preview and download of the derived PDF.
- Safe original HTML source inspection.
- Compose service and environment configuration.

### Out Of Scope

- Gotenberg's direct `/forms/chromium/convert/url` route.
- Converting a URL string into a stored page snapshot.
- Multi-file HTML bundles or a new asset-upload model. A single uploaded HTML
  file is sent as the conversion input.
- A separate plain-text-to-PDF implementation. `.txt` remains original-only,
  even though some LibreOffice builds accept plain text.
- Image, graphics, PDF, archive, or arbitrary unknown-file conversion.
- Replacing the original with the PDF.
- Creating a visible file version for the PDF.
- Synchronous conversion during upload.
- A second conversion queue, worker service, or separate deployment. The
  existing scheduler owns the work.

## Domain and Context Boundaries

### Subdomain Analysis

- **File storage and versioning: core domain.** It owns the canonical original,
  tenant isolation, versions, access, and lifecycle.
- **Preview conversion: supporting subdomain.** It improves usability but must
  not weaken the storage and versioning invariants.
- **Gotenberg: generic external capability.** SimpleDMS conforms to its HTTP
  routes through a small adapter and must not leak Gotenberg's model into the
  file domain.

### Bounded Contexts

The modular monolith keeps the File context and Preview Conversion context
logically separate:

- The File context owns `File`, `FileVersion`, original `StoredFile`, access
  checks, downloads, and deletion.
- The Preview Conversion context owns conversion status, route selection,
  retry policy, and the relationship between an original and its derived
  artifact.
- The Gotenberg adapter translates source metadata and streams into the
  external HTTP multipart API. Gotenberg response codes and response bodies do
  not become domain language.

The relationship is customer-supplier: the Preview Conversion context consumes
the File context's authorized source and storage capabilities, while the File
context remains the source of truth for document identity and access.

## Functional Requirements

### Source Preservation

1. Upload completion must succeed independently of Gotenberg availability,
   conversion success, or conversion configuration.
2. The original filename, MIME type, content hash, size, storage object, and
   existing `FileVersion` must remain unchanged by conversion.
3. A conversion failure must never delete, replace, or hide the original.
4. A source version must have at most one active preview artifact. Reprocessing
   replaces or removes only the previous derived artifact after the new result
   is safely stored.

### Eligibility and Routing

The classifier examines the stored MIME type and filename extension. A known
supported extension or a known supported MIME type is sufficient to make a
source eligible. The classifier must normalize case and map MIME-only inputs to
a canonical multipart filename extension before calling Gotenberg.

The classifier must route inputs as follows:

| Family | Gotenberg route | Input contract |
| --- | --- | --- |
| HTML | `POST /forms/chromium/convert/html` | Send the source as a single file named `index.html`. |
| Markdown | `POST /forms/chromium/convert/markdown` | Send the source as one `.md` file and a generated `index.html` template containing `{{ toHTML "source.md" }}`. |
| Office | `POST /forms/libreoffice/convert` | Send exactly one source document. |

The implementation must use an explicit allowlist for the pinned Gotenberg
version. The office allowlist covers document families, not every LibreOffice
input:

- Word processing: common Microsoft Word, OpenDocument, RTF, and equivalent
  word-processing formats supported by the pinned Gotenberg image.
- Spreadsheets: common Microsoft Excel, OpenDocument, CSV, and equivalent
  spreadsheet formats supported by the pinned image.
- Presentations: common Microsoft PowerPoint, OpenDocument, and equivalent
  presentation formats supported by the pinned image.

The allowlist excludes plain text, images, graphics, PDF, archives, and generic
unknown formats. The allowlist and MIME mappings must be tested against the
actual pinned Gotenberg image rather than assumed from a browser MIME table.

When extension and MIME disagree, route using whichever identifies a supported
family. The original still remains canonical if Gotenberg rejects the content.

Existing PDFs are already previewable and must not receive a redundant derived
PDF artifact.

### Discovery and Backfill

The conversion worker runs as part of the existing scheduler processing. It
must discover:

- Existing eligible source versions with no conversion record.
- New uploads after upload success and after the source object is available at
  its final storage location.
- New versions through the same query, without a special upload-handler hook.

There is no separate backfill command or admin workflow. The normal scheduler
scan is the backfill and must be idempotent. Sources still in temporary storage,
unfinished uploads, directories, soft-deleted orphan rows, and source objects
that cannot be opened are not sent to Gotenberg.

The scan must process historical versions, not only the current version. If the
same `StoredFile` is reused by multiple file versions, the conversion may be
deduplicated by source `StoredFile` because the source bytes are identical.

### Conversion State

Conversion state must distinguish at least:

- `pending`: eligible source is waiting for an attempt.
- `processing`: an attempt has claimed the source.
- `ready`: the PDF artifact exists at its final storage location.
- `failed`: the source was eligible but conversion reached a terminal failure.

An ineligible source has no conversion record and keeps today's original-only
behavior. Missing Gotenberg configuration is a runtime-disabled condition, not
a reason to fail an upload.

The conversion record must retain:

- source `StoredFile` identity;
- nullable derived PDF `StoredFile` identity;
- state;
- automatic retry count;
- last attempt time;
- next eligible attempt time;
- a stable failure category suitable for the UI and logs.

Do not store the full Gotenberg response body as user-visible document metadata.

### Scheduler Processing

1. The scheduler claims a pending source atomically before making the external
   request. A stale `processing` claim is returned to `pending` after a bounded
   recovery interval.
2. The source is opened through the existing tenant filesystem, so compression,
   encryption, and object-storage details stay outside the converter adapter.
3. The source is streamed to Gotenberg. The response is streamed into the
   existing tenant storage pipeline; it must not require holding the whole PDF
   in memory.
4. The returned file must be validated as a non-empty PDF response before it
   becomes a preview artifact.
5. The derived `StoredFile` uses `application/pdf`, a user-facing filename based
   on the original base name with `.pdf`, the returned byte size, and the same
   tenant storage, compression, encryption, and cleanup rules as other stored
   files.
6. The artifact is first written through temporary storage. Conversion becomes
   `ready` only after the final storage copy is complete and the artifact can be
   opened.
7. A failed attempt must remove any partial derived object and leave no ready
   artifact.

Use one in-flight conversion per application process initially. This avoids
parallel LibreOffice requests and is sufficient for the existing scheduler
design. Reuse the scheduler's bounded batch and polling style; do not add a
new worker service for the first implementation.

### Retry Policy

The initial attempt plus three automatic retries are allowed for transient
failures. The retry schedule must be persisted so a process restart does not
reset or duplicate attempts. Exponential backoff with bounded delays is the
default; the exact delays are implementation details.

Retryable failures include network errors, request timeouts, service-unavailable
responses, and other Gotenberg 5xx responses. Invalid input, a malformed source,
or a converter 4xx response that identifies the document as unreadable is a
terminal failure for the automatic cycle.

After automatic attempts are exhausted, the Preview tab remains visible with a
friendly failure message and a Retry action. Retry:

- is authorized using the source file's existing view permission;
- resets the retry schedule and failure state to `pending`;
- does not alter the original or create a visible version;
- is rejected when Gotenberg is not configured.

### Storage Quota

The derived PDF's uncompressed/original byte size is included in tenant storage
usage because it is a `StoredFile`. Conversion itself must not undo an original
upload when the PDF would exceed the quota. It stores the PDF and may create an
overage. Existing upload quota checks then apply to later uploads while the
tenant remains over quota.

Deleting a derived artifact releases its bytes through the existing quota query.

## Preview UX

### Main and Version Preview

The main file preview and the historical version preview dialog use the same
preview representation:

- An eligible source shows `Preview` and `Original` tabs.
- `Preview` is selected by default when no explicit preview-tab state exists.
- `Original` remains available and always refers to the exact source version.
- The normal app-bar download remains the original download.
- The Preview tab provides a PDF download when the derived artifact is ready.
- A source without a conversion record keeps the existing original-only
  preview.

The preview-tab state must be separate from the existing metadata side-sheet
tab state and must participate in the existing URL state/history behavior. A
user who explicitly selects `Original` must not be silently moved to `PDF` on
every partial refresh.

### Pending, Unavailable, and Failed States

For a known eligible source:

- `pending` or `processing`: select Preview by default, show the generating
  status in its content, and poll only the preview status/content fragment.
- `ready`: show the generated PDF and stop polling.
- `failed`: show a stable friendly error and Retry action in Preview content.
- Gotenberg URL missing: explain that PDF preview is unavailable in Preview
  content while leaving Original fully usable.

Polling must stop on ready or terminal failure. It must not refresh the entire
file details page or reset unrelated side-sheet state. The implementation should
use the existing HTMX partial conventions and a modest interval, such as five
seconds while a preview is pending.

### Original HTML Safety

The Original tab for HTML must show escaped source text, not execute uploaded
HTML in the SimpleDMS application origin. The response serving that view must
use a text content type and appropriate no-sniff headers. The original download
continues to provide the original file.

Markdown and other browser-previewable originals may use the existing native
preview path, subject to the current content-type and download safety rules.

## Data and Lifecycle Invariants

1. Every ready preview artifact maps to exactly one source `StoredFile`.
2. A source can never have more than one active ready artifact.
3. A preview artifact is never linked into `FileVersion`, so it cannot appear in
   file listings, the document version list, duplicate detection, or normal
   document navigation.
4. Access to a preview is granted only after checking access to its source file
   and space. Clients must not be able to fetch an artifact by an unscoped
   internal artifact ID.
5. A source version and its preview artifact have independent storage objects,
   but the source remains the canonical document content.
6. Soft-deleted files retain enough state for restore. When the source version
   is permanently cleaned up, its preview conversion record, artifact row, and
   object are removed together.
7. Re-uploading a new version creates a new conversion lifecycle; it must not
   overwrite the previous version's original or preview.
8. A file rename does not change the bytes or conversion identity of an
   existing version. The derived download name is based on that version's
   original filename.

## Configuration and Deployment

Add the optional application setting:

```text
SIMPLEDMS_GOTENBERG_URL
```

Behavior:

- Trim a trailing slash before constructing route URLs.
- An unset or empty value logs one warning during scheduler startup and disables
  conversion.
- An invalid URL logs an error and disables conversion without stopping the
  application.
- The value is stored in the main system configuration. The environment value is
  used when initializing the app and when database configuration overrides are
  explicitly enabled; runtime consumers read only the stored value. It is not
  stored in tenant data.

Compose requirements:

- Add `gotenberg/gotenberg:8` to `docker-compose.yml`.
- Add the same service to `docker-compose.sample.yml` and make the SimpleDMS
  service depend on it.
- Use `http://localhost:3000` for a host-running development server and
  `http://gotenberg:3000` for the application container in the sample stack.
- Document the matching value in `.env.sample` or the Compose service
  environment so the deployment examples work without code changes.
- Configure the Gotenberg image with private-IP denial for Chromium and
  LibreOffice outbound requests. Do not enable direct remote URL conversion.
- Pin a tested Gotenberg 8.x patch or digest for production releases even if
  development Compose uses the major tag.

## Security and Reliability

- Treat every source as untrusted input. Never pass user cookies, credentials,
  or application authorization headers to Gotenberg.
- Do not expose the Gotenberg service publicly in the sample deployment.
- Keep outbound URL filtering enabled and deny private/link-local destinations
  for Chromium and LibreOffice. This protects against HTML or document content
  attempting to reach internal services.
- Bound the HTTP request time and response size. Respect the existing upload
  limits and avoid unbounded memory buffering.
- Log source tenant, source identifier, route family, attempt number, status
  code, and Gotenberg trace identifier when available. Do not log document
  content or sensitive response bodies at normal log level.
- A Gotenberg outage must degrade to original-only preview, not upload failure.
- Scheduler recovery must be safe after process termination between object
  creation and database state updates. Orphaned temporary objects must be
  removed by the existing cleanup path or a conversion-specific cleanup path.

## Implementation Guidance

- Add the smallest concrete HTTP adapter needed for the three routes. Keep
  Gotenberg-specific route names, form fields, response parsing, and retry
  classification inside that adapter.
- Add a tenant schema entity for the conversion record and use a nullable link
  to the derived `StoredFile`. Do not edit generated Ent code; add the schema,
  generate code, and create the migration through the repository workflow.
- Reuse `S3FileSystem` for opening originals and persisting encrypted/compressed
  derived files. Do not add a second storage format.
- Extend `Scheduler` construction with the optional Gotenberg client and start
  a recoverable conversion loop beside the existing processing loops.
- Reuse `FilePreview` for PDF rendering and the existing `TabBar`/HTMX partial
  patterns for the two preview tabs. Add only the status and download routes
  needed to keep the artifact hidden from normal file navigation.
- Keep command handlers thin: authorization, binding, state transition, and
  rendering belong at the application boundary; route selection and conversion
  policy belong to the conversion component.
- Use public file identifiers and version numbers in URLs. Keep internal
  artifact IDs behind source-scoped server lookups.

## Testing Requirements

### Unit Tests

- Extension and MIME classifier, including case normalization, MIME-only
  canonical extension mapping, mismatches, excluded formats, PDFs, and plain
  text.
- Route and multipart construction for HTML, Markdown, and office inputs.
- Retry classification, attempt counting, backoff eligibility, and manual
  retry reset.
- Derived PDF filename construction and source/artifact state transitions.

### Integration Tests

- Successful conversion using a fake Gotenberg HTTP server and the existing
  tenant storage test setup.
- Upload succeeds while Gotenberg is unavailable or not configured.
- Original remains readable after transient failure, terminal failure, and
  quota overage.
- Scheduler discovers historical versions and newly uploaded versions without
  duplicate conversion records.
- Final-storage readiness prevents a partial artifact from being served.
- Source-scoped authorization prevents cross-space or cross-tenant artifact
  access.
- Cleanup removes conversion metadata and the derived storage object with the
  source's permanent cleanup.

### Browser/E2E Tests

- HTML, Markdown, and office fixtures show the expected tabs and Preview-first
  behavior.
- Original and generated PDF downloads are distinct and correct.
- Pending polling changes to ready without resetting the open preview state.
- Failed conversion keeps Original available and exposes Retry to a viewer.
- HTML Original view displays escaped markup and does not execute a script.
- Historical version preview selects the PDF generated from that exact version.

## Acceptance Criteria

The feature is complete when all of the following are true:

1. Uploading an eligible HTML, Markdown, or office-family file returns normally
   without waiting for Gotenberg.
2. The original remains stored and downloadable regardless of conversion result.
3. The scheduler automatically converts both existing eligible versions and
   future eligible versions.
4. A successful conversion produces one hidden PDF artifact with PDF MIME type,
   source-based `.pdf` filename, tenant storage protection, and a PDF download.
5. The main preview and version preview expose Preview/Original tabs with
   Preview selected by default.
6. Pending, unavailable, and failed states are visible without making Original
   unusable; failed state supports viewer-authorized Retry.
7. Plain text, existing PDF, image, archive, and unknown files do not receive a
   redundant conversion artifact.
8. URL imports convert their downloaded eligible file, while direct remote URL
   rendering is not used.
9. Derived PDF bytes count toward storage usage and may create overage without
   undoing an upload.
10. Removing a source permanently removes its derived preview artifacts and
    conversion metadata.
11. Missing Gotenberg configuration produces a warning and no application
    startup failure.

## Tradeoffs and Rejected Alternatives

- **Derived artifact instead of a visible version:** preserves document history
  and prevents preview implementation details from polluting user workflows.
- **Scheduler instead of synchronous conversion:** keeps upload latency and
  availability independent from a heavy external converter.
- **Explicit allowlist instead of trying every file:** avoids sending arbitrary
  untrusted content to LibreOffice and makes UI status predictable.
- **No direct URL route:** avoids adding a second SSRF-sensitive fetch path; the
  existing URL importer already stores a controlled download result.
- **One in-flight conversion initially:** matches LibreOffice's serialized
  processing behavior and avoids speculative worker/concurrency machinery.
- **Quota counted but not enforced during conversion:** honors the requirement
  that a successful original upload is never rolled back because a later,
  optional preview is too large.

## Open Questions

There are no blocking product questions. The implementation may choose the
exact Gotenberg 8.x patch or digest, retry delay values, stale-claim interval,
HTTP timeout, and the complete MIME alias table as engineering defaults, as
long as the behavior above and the pinned image's tested extension matrix are
preserved.
