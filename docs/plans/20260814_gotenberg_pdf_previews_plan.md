# Gotenberg PDF Preview Conversion Plan

Source specification: `docs/specs/20260814_gotenberg_pdf_previews.md`

Status: Proposed

Date: 2026-08-14

## Business Domain And Outcome

SimpleDMS remains the system of record for uploaded originals and their file
versions. A supporting Preview Conversion capability creates a PDF reading
representation asynchronously, without changing upload behavior or document
history.

Primary actors:

- Users who need a reliable visual preview of office, HTML, and Markdown files.
- Users who need to inspect or download the exact original source.
- Scheduler and storage maintenance processes that must preserve tenant and
  lifecycle invariants.
- Operators who configure the optional Gotenberg service through Compose and an
  environment variable.

The implementation is complete when users can open the main or historical
version preview, switch between `Preview` and `Original`, and continue using the
original when conversion is disabled or fails.

## Goals

- Keep the original `StoredFile` and `FileVersion` canonical and unchanged.
- Convert eligible existing and newly completed source versions in the existing
  scheduler.
- Store one hidden, tenant-protected PDF artifact per source `StoredFile`, with
  deduplication when the same stored source is reused.
- Keep conversion status durable across scheduler restarts.
- Add source-scoped PDF preview and download routes without exposing internal
  artifact identifiers.
- Render pending, disabled, ready, and failed states through server-rendered
  HTMX fragments.
- Make HTML original inspection safe by showing escaped source text.
- Keep the feature optional when `SIMPLEDMS_GOTENBERG_URL` is empty.

## Non-Goals

- Do not convert direct remote URLs through Gotenberg.
- Do not add a separate worker process, queue service, or webhook deployment.
- Do not create visible document versions for generated PDFs.
- Do not convert plain text, images, graphics, archives, existing PDFs, or
  unknown formats.
- Do not support local HTML asset bundles in the first implementation.
- Do not block or roll back uploads when Gotenberg is unavailable or when a
  generated PDF creates tenant quota overage.
- Do not introduce client-side application state for conversion status.

## Current Repository Context

### Storage And Versions

- `db/enttenant/schema/stored_file.go` defines tenant stored objects, upload
  status, storage paths, final-copy timestamps, and content sizes.
- `db/enttenant/schema/file_version.go` links a file version to one stored file.
- `model/tenant/filesystem/s3_file_system.go` opens encrypted/compressed S3
  objects and persists temporary tenant files before final copy.
- `scheduler/process_files.go` copies temporary tenant objects to final storage
  and later removes processed temporary objects.
- `model/tenant/filesystem/storage_quota.go` sums `StoredFile.size`, so derived
  PDF rows count toward usage unless an explicit exclusion is introduced. The
  specification requires no exclusion.

### Scheduler And External Services

- `scheduler/scheduler.go` starts independent recoverable loops and currently
  injects the optional Tika client.
- `scheduler/apply_ocr.go` is the closest existing pattern for iterating all
  tenant databases, opening final files, applying an external service, and
  persisting retry state.
- `server/server.go:startScheduler` reads the Tika URL from application
  configuration and constructs `Scheduler`.
- The new Gotenberg URL is environment-only and must not be added to tenant
  database configuration.

### Preview And Routes

- `action/browse/file_preview_partial.go` renders the main details preview.
- `action/browse/file_version_preview_dialog.go` renders a historical version
  preview and currently embeds `widget.FilePreview` directly.
- `core/ui/widget/file_preview.go` and `file_preview.gohtml` render browser
  previews and PDF.js fallback behavior.
- `core/ui/widget/tab.go`, `tab.gohtml`, and `tab_bar.gohtml` render HTMX tabs.
- `ui/uix/route/download.go` currently supports original inline and attachment
  downloads by file and optional version number.
- `action/common/download_helper.go` streams a selected stored file and sets
  content disposition and MIME headers.
- Existing preview state already uses URL state for side-sheet tabs, so the new
  main preview tab state must use a separate query field.

### Deployment

- `docker-compose.yml` already exposes Tika, Mailpit, VersityGW, and
  Toxiproxy for development.
- `docker-compose.sample.yml` runs the application with Tika and VersityGW.
- `.env.sample` contains host-running development defaults, including the Tika
  URL. Add the optional Gotenberg URL without changing the existing DB-backed
  configuration behavior.

## Domain And Context Design

### Ubiquitous Language

- **Original:** canonical uploaded bytes in the existing `StoredFile`.
- **Source version:** a `FileVersion` and its linked original `StoredFile`.
- **Preview artifact:** a derived PDF `StoredFile` that is not a document
  version and is hidden from normal file navigation.
- **Preview conversion:** the asynchronous operation that creates the artifact.
- **Eligible source:** a source recognized by the explicit MIME/extension
  allowlist.
- **URL import:** the existing downloaded-file flow, not direct remote-page
  rendering.

Use these terms consistently in schema names, model methods, statuses, logs,
routes, templates, and tests.

### Bounded Contexts

Keep one modular monolith with two logical contexts:

- **File context:** owns file identity, versions, canonical originals, access,
  download authorization, and source lifecycle.
- **Preview Conversion context:** owns eligibility, route selection, conversion
  records, retry state, and derived-artifact lifecycle.

The Gotenberg HTTP client is an anti-corruption adapter at the boundary of the
Preview Conversion context. Gotenberg route names, multipart fields, trace
headers, and response bodies must not leak into file-domain models or UI text.

### Modelling Approach

Use a small active-record/transaction-script style model around the existing Ent
storage system. A full domain model, event sourcing, CQRS, separate queue, or
microservice is not justified. The important invariants are the source/artifact
relationship, durable state transitions, authorization, and cleanup ordering.

## Target Design

### Conversion Record

Add a tenant Ent schema entity named `PreviewConversion` in
`db/enttenant/schema/preview_conversion.go`.

The target fields are:

| Field | Purpose |
| --- | --- |
| `source_stored_file_id` | Required unique identity of the original source object. |
| `preview_stored_file_id` | Nullable link to the derived PDF `StoredFile`. |
| `status` | `pending`, `processing`, `ready`, or `failed`. |
| `retry_count` | Number of automatic attempts already consumed. |
| `last_attempted_at` | Last external conversion attempt. |
| `next_attempt_at` | Earliest time a retry may be claimed. |
| `processing_started_at` | Used to recover stale claims after process termination. |
| `failure_category` | Stable, non-sensitive category used by UI and logs. |

Use the existing common mixin for timestamps and the repository's normal Ent
privacy approach. Use explicit `GoType()` enums rather than generated enum
value slices.

The source edge is required and unique. The preview edge is optional and points
to a `StoredFile` row that has no `FileVersion` edge. This keeps the artifact
out of document listings, version lists, duplicate detection, and normal
navigation while allowing the existing storage pipeline to protect it.

The conversion record's access policy must follow the source's accessible file
versions and Spaces. The scheduler may use a narrowly scoped privacy bypass for
maintenance queries, as existing backfill code does. HTTP routes must resolve
the source file and access context before loading the artifact.

Add the schema migration through the repository's Ent/migration workflow. Never
edit generated files under `db/enttenant` manually.

### Eligibility Classifier

Keep pure eligibility and route selection separate from I/O. A small concrete
classifier can live in the Preview Conversion model package or scheduler
package, but it must have no database or HTTP dependency.

Inputs:

- stored MIME type;
- original filename extension;
- whether the source is a directory;
- whether the source is already a PDF.

Rules:

- Normalize MIME and extension case.
- A known supported extension or MIME is enough for eligibility.
- Map MIME-only inputs to a canonical temporary filename extension for the
  multipart request.
- Route HTML to Chromium HTML.
- Route Markdown to Chromium Markdown.
- Route office-family documents to LibreOffice.
- Exclude plain text, images, graphics, PDF, archives, directories, and unknown
  formats.
- Keep the complete office extension/MIME table explicit and test it against
  the pinned Gotenberg image.

### Gotenberg Adapter

Add a small concrete client, likely in
`scheduler/gotenberg_client.go` or a focused Preview Conversion package. Do not
add an interface solely for the client; use an injectable URL and
`httptest.Server` for tests.

Responsibilities:

- Build the base URL safely and append the selected route.
- Construct streaming multipart requests.
- Send `index.html` for HTML conversion.
- Send a deterministic minimal `index.html` template plus one `.md` source for
  Markdown conversion. The template must contain
  `{{ toHTML "source.md" }}`.
- Send exactly one office input to LibreOffice.
- Set `Gotenberg-Output-Filename` to the source-based PDF base name.
- Bound request time and response size.
- Validate HTTP status, content type, non-empty body, and `%PDF-` magic bytes.
- Return a stream suitable for the existing storage pipeline.
- Classify network/timeouts/5xx as retryable and document/content 4xx failures
  as terminal.
- Log trace identifiers through the scheduler without exposing document bytes.

Do not call `/forms/chromium/convert/url`. Do not send cookies, credentials, or
SimpleDMS authorization headers.

### Derived Artifact Storage

Extend `S3FileSystem` with the smallest purpose-specific operations needed to
persist a derived PDF through the existing path:

1. Generate tenant-scoped temporary and final storage metadata for a PDF.
2. Stream the Gotenberg response through the existing compression, encryption,
   hashing, and S3 upload pipeline.
3. Create the derived `StoredFile` with `application/pdf`, source-based PDF
   filename, uncompressed byte size, storage byte size, hashes, upload status,
   and temporary/final paths.
4. Let the existing final-copy scheduler move the object to its final path.
5. Mark `PreviewConversion` ready only after the derived object is copied and
   can be opened.
6. Remove partial temporary/final objects and rows when a conversion attempt
   fails or is abandoned.

Do not attach the derived row to `FileVersion`. Do not run the normal upload
quota rejection against the derived output. Its `StoredFile.size` must still be
included in the existing tenant usage sum.

### Scheduler Worker

Add a recoverable `convertPreviews` loop and start it from `Scheduler.Run` only
when the optional client is configured. A missing URL logs one warning and
leaves the rest of the scheduler running.

Each pass should:

1. Iterate initialized tenant databases using the existing scheduler pattern.
2. Discover final, completed source stored files with no conversion record.
3. Create conversion records idempotently with the unique source constraint.
4. Claim due `pending` records atomically as `processing`.
5. Recover stale `processing` records to `pending`.
6. Open the source from final storage using the tenant identity.
7. Call Gotenberg and stream the result into a derived temporary artifact.
8. Let the existing copy loop move the derived object to final storage.
9. On a later pass, verify the derived object and mark the conversion `ready`.
10. On failure, clean partial objects, persist the category, and schedule the
    initial attempt plus up to three automatic retries for transient errors.

Use one in-flight conversion per application process initially because
LibreOffice serializes conversions. Reuse the scheduler's existing bounded batch
and polling style. A process restart must not lose retry state or create a
second artifact for the same source.

### Manual Retry

Add a viewer-authorized command, likely under `action/browse`, that accepts a
public file ID and optional version number. It must:

- resolve the source through the current tenant and Space context;
- verify the caller can view the source;
- find the source-scoped failed conversion;
- reset retry count, failure category, and next-attempt time;
- transition to `pending`;
- return a snackbar/event response suitable for HTMX refresh;
- reject the action when Gotenberg is not configured.

The command must not accept a preview artifact ID as its security boundary.

### Preview And Download Routes

Add route helpers and handlers for source-scoped operations:

- Original source inspection for HTML, forcing escaped `text/plain` output and
  no-sniff headers.
- Derived PDF inline preview.
- Derived PDF attachment download.
- Preview conversion status/polling partial.
- Viewer-authorized retry command.

The PDF route must accept the public file ID and optional version number, resolve
that exact source version, check existing Space access, then load its
`PreviewConversion`. It must return not found or an appropriate unavailable
state when no ready artifact exists. It must never expose a route keyed only by
an internal preview artifact ID.

Reuse `action/common/download_helper.go` patterns for streaming and headers, but
keep original and derived download behavior distinct. The normal app-bar
download remains the original; the Preview tab supplies a separate PDF download.

### Preview UI Composition

Create one shared server-rendered preview composition used by:

- `action/browse/file_preview_partial.go` for the main preview;
- `action/browse/file_version_preview_dialog.go` for historical versions.

The shared composition must:

- render `Preview` and `Original` peer tabs only for eligible sources or known
  disabled/failed conversion records;
- select Preview by default whenever no explicit preview-tab state exists;
- keep Original usable in pending, unavailable, and failed states;
- render pending, unavailable, and failed status inside Preview content;
- retain a friendly failure error and Retry action in Preview content;
- preserve separate side-sheet tab state;
- use the exact source version for both tabs and downloads;
- stop polling after ready or terminal failure.

Keep `TabBar` scroll, selected-state, and keyboard/focus behavior intact.

Use a separate URL state field such as `preview_tab`, not the existing metadata
`tab` field. The state must survive HTMX swaps and browser history. Poll only a
stable preview status/content fragment with an HTMX `every` trigger while the
conversion is pending.

### Configuration And Compose

Add `SIMPLEDMS_GOTENBERG_URL` to `.env.sample` and read it at startup in the
server's scheduler wiring. Use:

- `http://localhost:3000` for a host-running development server;
- `http://gotenberg:3000` for the containerized sample stack.

Add `gotenberg/gotenberg:8` to both Compose files. The sample application must
depend on the service but must not publish Gotenberg externally. Configure
Chromium and LibreOffice private-IP denial for untrusted source content.
Production releases should pin a tested Gotenberg patch or digest.

### Translations And Accessibility

Add translations for all fixed labels and states, including:

- `Original`;
- `PDF`;
- generating/unavailable/failed preview messages;
- `Retry PDF generation`;
- `Download PDF`.

Follow repository translation rules: informal German with Swiss spelling,
French and Italian entries, `fuzzy: true` for new translations, and a
`Translated from English by Codex.` comment. Do not translate user-provided
filenames.

The tabs and Retry action must have visible focus, selected, disabled, and
loading states. Preserve keyboard order, screen-reader labels, adequate touch
targets, and the existing Material 3 tab composition on compact and expanded
layouts.

## Schema And Migration Strategy

1. Add the `PreviewConversion` schema and enum in `db/enttenant/schema`.
2. Add source/preview edges and a unique index on the source stored file.
3. Add indexes supporting due work ordered by status, next attempt, and ID.
4. Generate Ent code with the repository's existing `go generate ./...` flow.
5. Create the tenant migration through the existing migration workflow.
6. Verify existing tenant migration and new-tenant initialization behavior.
7. Do not add a synchronous migration that scans or converts files. The
   scheduler performs the backfill after deployment.

## Security And Failure Handling

- Resolve every source and artifact through tenant and Space access checks.
- Keep internal artifact IDs out of public URLs and form data.
- Never pass application credentials or user cookies to Gotenberg.
- Force HTML source inspection to escaped text with a text content type and
  no-sniff behavior.
- Deny private/link-local outbound destinations in Gotenberg modules.
- Bound request time, response size, and memory usage.
- Preserve the original on every conversion failure.
- Do not mark an artifact ready before final storage copy and open verification.
- Recover stale claims and clean orphaned temporary objects after interruption.
- Count ready derived bytes in quota usage, but allow conversion to create
  overage after upload.
- Keep friendly errors in the UI and detailed converter traces in logs only.

## Testing Strategy

### Unit Tests

Add focused tests for:

- extension and MIME eligibility, including case normalization and mismatch;
- MIME-only canonical extension mapping;
- all included office-family aliases and all excluded families;
- HTML, Markdown, and office route/multipart construction;
- PDF response validation and output filename construction;
- retryable versus terminal error classification;
- attempt counts, backoff eligibility, stale claims, and manual reset;
- conversion state transitions and duplicate source protection.

### Integration Tests

Use an `httptest.Server` as a fake Gotenberg endpoint and the existing tenant
storage test setup to verify:

- successful streamed conversion and final-storage readiness;
- upload success while Gotenberg is missing, unavailable, or failing;
- historical backfill and new-version discovery;
- no duplicate conversion records or artifacts after repeated scheduler passes;
- original readability after every failure mode;
- derived bytes included in quota usage and overage behavior;
- source-scoped access across Spaces and tenants;
- cleanup of conversion records, artifact rows, and objects.

### Browser And E2E Tests

Cover HTML, Markdown, and office fixtures for:

- Preview-first tabs in the main preview and version dialog;
- exact Original and PDF content for current and historical versions;
- original and PDF downloads;
- pending polling without resetting side-sheet or preview-tab state;
- failed Retry action for a viewer;
- unavailable state when the environment variable is absent;
- escaped HTML source that cannot execute a script.

## Implementation Phases

1. **Preparation:** confirm the Gotenberg 8.x image/digest, complete the tested
   office extension/MIME matrix, and identify all current file cleanup paths.
2. **Schema:** add `PreviewConversion`, edges, indexes, generated code, and the
   tenant migration.
3. **Classification and adapter:** implement eligibility, route selection,
   streaming multipart requests, response validation, and failure categories.
4. **Storage:** add derived-PDF persistence using existing encrypted/compressed
   temporary storage and final-copy behavior.
5. **Scheduler:** add idempotent discovery/backfill, atomic claims, stale-claim
   recovery, retries, ready finalization, and cleanup.
6. **Access and commands:** add source-scoped preview/download routes and the
   viewer-authorized Retry command.
7. **Preview UI:** compose shared Preview/Original tabs for main and version
   previews, safe HTML source rendering, status states, and HTMX polling.
8. **Deployment and language:** update environment samples, both Compose files,
   Gotenberg network hardening, translations, and accessibility labels.
9. **Verification:** run focused tests, browser tests, full Go tests/build, and
   inspect migration and storage behavior before enabling Gotenberg in a
   deployment.

## Likely Files To Touch

### Schema And Model

- `db/enttenant/schema/preview_conversion.go`
- `db/enttenant/schema/stored_file.go`, only if an edge or index is required
- `model/tenant/previewconversion/preview_conversion.go`
- `model/tenant/filesystem/s3_file_system.go`
- `model/tenant/filesystem/storage_quota.go`, only if generated artifacts need a
  query adjustment; the intended behavior is to count them without exclusion
- generated Ent files through `go generate ./...`
- tenant migration files through the normal migration workflow

### Scheduler And Integration

- `scheduler/scheduler.go`
- new focused scheduler conversion/client files
- `scheduler/process_files.go`, only for final-copy or cleanup coordination
- `scheduler/apply_ocr.go` as a pattern, not as a shared implementation target
- `server/server.go`
- `.env.sample`
- `docker-compose.yml`
- `docker-compose.sample.yml`

### Preview And Routes

- `action/browse/file_preview_partial.go`
- `action/browse/file_version_preview_dialog.go`
- new browse preview status/retry action files as needed
- `action/common/download_helper.go`
- `ui/uix/route/download.go` and related route files
- `core/ui/widget/tab.go`
- `core/ui/widget/tab.gohtml`
- `core/ui/widget/file_preview.go` and `file_preview.gohtml`
- `ui/uix/event/*`, only if an event is needed beyond polling
- `server/*` route registration and integration tests

### Language And Tests

- `i18n/*/messages.gotext.json`
- scheduler, model, filesystem, and server integration tests
- `e2e/` fixtures and specs

## Verification Commands

Run focused checks during implementation, then the full suite:

```bash
gofmt -w <changed-go-files>
go generate ./...
go test ./scheduler ./model/tenant/... ./server
go test ./...
go build ./...
go vet ./...
git diff --check
```

If browser behavior changes, run the repository's existing Playwright/E2E
workflow with Gotenberg available and verify both a ready and a failing
conversion path.

## Rejected Alternatives

- A visible PDF `FileVersion` would pollute document history and version UI.
- Synchronous upload conversion would couple upload availability to Gotenberg.
- Direct remote URL conversion would add a second SSRF-sensitive fetch path.
- A separate queue or worker service is unnecessary for the initial serialized
  conversion workload.
- Trying every unknown file is less predictable and sends arbitrary content to
  LibreOffice.
- Client-side state or WebSockets are unnecessary while HTMX polling handles the
  pending fragment.

## Implementer Handoff

Preserve these invariants while implementing:

- Original bytes and source version identity never change.
- A ready preview is final-storage backed and source-scoped.
- Preview artifacts never become visible document versions.
- A source has at most one active derived artifact.
- Conversion failure never blocks or removes the original.
- Historical versions are processed, not only current versions.
- PDF bytes count toward quota, but conversion may create overage.
- HTML originals are rendered as escaped source text.
- Missing Gotenberg disables conversion and leaves the application usable.
