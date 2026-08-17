# Gotenberg PDF Preview Conversion Checklist

Specification: `docs/specs/20260814_gotenberg_pdf_previews.md`

## Preparation And Decisions

- [x] Confirm the Gotenberg 8.x image version tested: local
  `gotenberg/gotenberg:8` resolved to `8.34.0`; production pin remains open.
- [x] Build the explicit office-family extension and MIME allowlist from the
  tested Gotenberg 8.34.0 image.
- [ ] Confirm all existing source-version, soft-delete, permanent-cleanup, and
  stored-object cleanup paths.
- [x] Confirm the existing tenant migration and direct Ent generation commands
  work in the current checkout.
- [x] Record the conversion invariant document in implementation review notes.

## Schema And Migration

- [x] Add `PreviewConversion` to `db/enttenant/schema/preview_conversion.go`.
- [x] Add a required unique source `StoredFile` edge.
- [x] Add a nullable derived-preview `StoredFile` edge.
- [x] Add the `pending`, `processing`, `ready`, and `failed` status enum using
  `GoType()`.
- [x] Add retry count and persisted attempt timestamps.
- [x] Add stale-claim timestamp and next-attempt timestamp.
- [x] Add a stable failure category without storing full converter response
  bodies as user metadata.
- [x] Add indexes for source uniqueness and due scheduler work.
- [x] Add tenant privacy rules based on source file/Space access.
- [ ] Generate Ent code with `go generate ./...`.
- [x] Create the tenant migration through the repository migration workflow.
- [ ] Verify migration for existing tenants and initialization for new tenants.
- [x] Do not edit generated files manually.

## Eligibility And Gotenberg Adapter

- [x] Implement pure extension/MIME classification.
- [x] Normalize MIME and extension case.
- [x] Map MIME-only sources to a canonical multipart filename extension.
- [x] Route HTML to `/forms/chromium/convert/html` as `index.html`.
- [x] Route Markdown to `/forms/chromium/convert/markdown` with one source `.md`
  file and the generated `index.html` template.
- [x] Ensure the Markdown template contains `{{ toHTML "source.md" }}`.
- [x] Route office-family inputs to `/forms/libreoffice/convert` one file at a
  time.
- [x] Exclude plain text, images, graphics, PDFs, archives, directories, and
  unknown types.
- [x] Do not call the direct Gotenberg URL route.
- [x] Build streaming multipart requests without buffering full originals.
- [x] Set `Gotenberg-Output-Filename` from the source base name.
- [x] Bound request timeout and response size.
- [x] Validate status, content type, non-empty response, and `%PDF-` magic bytes.
- [x] Classify network, timeout, service-unavailable, and 5xx failures as
  retryable.
- [x] Classify unreadable/malformed content and relevant 4xx responses as
  terminal.
- [x] Preserve trace IDs and source context in logs without logging content.
- [ ] Add unit tests for every route and failure classification.

## Derived Storage

- [x] Add the smallest filesystem operation for creating a tenant-scoped derived
  PDF through temporary storage.
- [x] Reuse existing compression, encryption, hashing, S3, and final-copy logic.
- [x] Set derived MIME type to `application/pdf`.
- [x] Set the derived user-facing filename to the source base name plus `.pdf`.
- [x] Persist uncompressed PDF size in `StoredFile.size`.
- [x] Persist storage size and hashes using existing conventions.
- [x] Set upload status and temporary/final storage timestamps correctly.
- [x] Keep the derived row unattached to `FileVersion`.
- [x] Ensure the existing quota sum includes ready derived `StoredFile` rows.
- [x] Do not reject or roll back derived storage because of tenant quota.
- [x] Remove partial derived rows and objects after failed attempts.
- [ ] Add tests for streaming, final-copy readiness, and partial cleanup.

## Scheduler And Backfill

- [x] Read `SIMPLEDMS_GOTENBERG_URL` in scheduler startup wiring.
- [x] Log one warning and disable conversion when the URL is empty.
- [x] Log an error and disable conversion when the URL is invalid.
- [x] Inject the optional concrete Gotenberg client into `Scheduler`.
- [x] Start a recoverable preview-conversion loop beside existing scheduler
  loops.
- [x] Iterate initialized tenant databases using existing scheduler conventions.
- [x] Discover existing eligible final source stored files without conversion
  records.
- [x] Discover newly completed uploads and newly uploaded versions through the
  same scan.
- [x] Exclude unfinished uploads, temporary-only sources, directories, orphans,
  and sources that cannot be opened.
- [x] Create conversion records idempotently under the unique source constraint.
- [x] Process historical versions, not only current versions.
- [x] Deduplicate work when the same source `StoredFile` is reused.
- [x] Atomically claim due `pending` records as `processing`.
- [x] Recover stale `processing` claims.
- [x] Keep one in-flight conversion per application process.
- [x] Open originals through the existing tenant filesystem and identity.
- [x] Stream successful output into a derived temporary artifact.
- [x] Mark `ready` only after final-copy completion and an open/read check.
- [x] Persist retry count, next-attempt time, and failure category.
- [x] Implement the initial attempt plus three automatic transient retries.
- [x] Reset failed conversion state through the manual Retry command.
- [x] Ensure restart/repeated scheduler passes do not duplicate artifacts.
- [x] Ensure stale or unfinished stored-file rows do not block recent uploads.
- [ ] Add scheduler tests for success, backfill, restart recovery, retry, and
  duplicate prevention.

## Lifecycle And Quota

- [x] Keep original `StoredFile` and `FileVersion` values unchanged.
- [x] Ensure a new source version gets a new conversion lifecycle.
- [x] Ensure old version artifacts are not overwritten by later versions.
- [x] Preserve soft-deleted source state for restore.
- [ ] Remove conversion metadata, derived row, and object during permanent source
  cleanup.
- [ ] Verify deleting a derived artifact releases quota usage.
- [x] Verify conversion can create quota overage after upload.
- [x] Verify later original uploads still use normal quota checks while overage
  exists.
- [ ] Add cleanup and quota integration tests.

## Access, Download, And Security

- [x] Resolve preview requests by public file ID and optional version number.
- [x] Check tenant, Space, and existing source-file view access before loading
  conversion state or artifact data.
- [x] Keep internal preview artifact IDs out of URLs, forms, and client state.
- [x] Add source-scoped inline PDF preview route.
- [x] Add source-scoped PDF attachment download route.
- [x] Keep the normal app-bar download pointed at the original.
- [x] Add source-scoped safe HTML source inspection.
- [x] Force safe HTML inspection to escaped `text/plain` output with no-sniff
  headers.
- [x] Do not pass user cookies, credentials, or SimpleDMS authorization headers
  to Gotenberg.
- [x] Do not publish Gotenberg in the sample deployment.
- [x] Configure Chromium private-IP denial.
- [x] Configure LibreOffice private-IP denial.
- [ ] Add cross-tenant and cross-Space authorization tests.
- [ ] Add a test proving HTML source scripts are not executed in the application
  origin.

## Preview UI And HTMX

- [x] Create one shared preview composition for main and version previews.
- [x] Add `Preview` and `Original` peer tabs for eligible sources.
- [x] Select Preview by default whenever no explicit tab is stored.
- [x] Keep Original usable during pending, unavailable, and failed states.
- [x] Render pending and unavailable status in the Preview tab content.
- [x] Keep a failed Preview tab visible with a friendly error and Retry action.
- [x] Add a separate `preview_tab` URL state field instead of reusing the
  metadata side-sheet `tab` field.
- [x] Preserve explicit Original selection through partial refreshes and browser
  history.
- [x] Poll only a stable preview status/content fragment while pending.
- [x] Stop polling on ready or terminal failure.
- [x] Ensure polling does not reset side-sheet state.
- [x] Extend `widget.Tab` only as needed for an accessible disabled state.
- [x] Add selected, focus, loading, and failure states.
- [x] Preserve existing TabBar scrolling and active-state behavior.
- [x] Reuse `widget.FilePreview` for the generated PDF.
- [x] Provide PDF download from the Preview tab.
- [x] Use the exact historical version for its Original, PDF, and download paths.
- [x] Reuse the shared Preview/Original composition in the Inbox.
- [ ] Add main-preview and version-dialog integration tests.
- [ ] Add browser coverage for pending-to-ready polling and failure Retry.

## Translations And Accessibility

- [x] Add English source strings for Original/Preview labels and all conversion
  states/actions.
- [x] Add German translations with informal `du`, Swiss spelling, `fuzzy: true`,
  and the required translator comment.
- [x] Add French translations with `fuzzy: true` and the required comment.
- [x] Add Italian translations with `fuzzy: true` and the required comment.
- [x] Do not edit generated `out.gotext.json` files.
- [x] Verify user-provided filenames remain untranslated.
- [ ] Verify keyboard focus order and visible focus states.
- [ ] Verify disabled and selected tab contrast and semantics.
- [ ] Verify tab and Retry touch targets on compact layouts.

## Configuration And Compose

- [x] Add `SIMPLEDMS_GOTENBERG_URL` to `.env.sample`.
- [x] Document `http://localhost:3000` for a host-running development server.
- [x] Configure the sample application to use `http://gotenberg:3000`.
- [x] Add `gotenberg/gotenberg:8` to `docker-compose.yml`.
- [x] Add `gotenberg/gotenberg:8` to `docker-compose.sample.yml`.
- [x] Add sample-stack dependency ordering for the Gotenberg service.
- [x] Avoid publishing Gotenberg from the sample stack.
- [x] Configure Gotenberg Chromium and LibreOffice private-IP denial.
- [ ] Pin a tested patch or digest for production release configuration.
- [x] Add a Compose smoke check for service reachability and health behavior.

## Tests And Verification

- [x] Run focused model/classifier tests.
- [x] Run focused Gotenberg adapter tests with `httptest.Server`.
- [ ] Run focused scheduler and filesystem integration tests.
- [x] Run focused server and preview integration tests.
- [ ] Run browser/E2E tests with Gotenberg available.
- [ ] Verify missing Gotenberg still permits normal uploads and previews.
- [ ] Verify a failing Gotenberg keeps the original usable.
- [ ] Verify existing historical backfill after a clean deployment.
- [x] Run `gofmt -w <changed-go-files>`.
- [ ] Run `go generate ./...`.
- [x] Run `go test ./...`.
- [x] Run `go build ./...`.
- [x] Run `go vet ./...`.
- [x] Run `git diff --check`.
- [ ] Review migration, object cleanup, and quota behavior before release.

## Completion Criteria

- [ ] Original bytes and source version identity are unchanged.
- [ ] Eligible existing and new versions convert asynchronously.
- [ ] Ready PDFs are hidden artifacts, source-scoped, final-storage backed, and
  downloadable.
- [ ] Main and historical previews expose Preview-first Preview/Original tabs.
- [ ] Pending, unavailable, ready, and failed states are correct and accessible.
- [ ] Viewer-authorized Retry works after automatic retries are exhausted.
- [ ] PDF bytes count toward quota and conversion may create overage.
- [ ] Permanent source cleanup removes preview metadata and objects.
- [ ] Missing Gotenberg disables only conversion, not application startup or
  upload.
- [ ] All required tests, builds, migrations, translations, and Compose checks
  pass.

## Verification Notes

- `go test ./...`, `go build ./...`, `go vet ./...`, and `git diff --check` pass.
- Live `gotenberg/gotenberg:8` resolved to 8.34.0; health, HTML, Markdown, and
  LibreOffice CSV conversions returned valid PDFs.
- `go generate ./...` still stops in the existing `enumer` generator when it
  imports `golang.org/x/text/language` (`binary export format 'v' is no longer
  supported`). Ent and i18n generation were run separately; generated locale
  output files were not modified.
- The existing Playwright suite stops in `e2e/global.setup.ts` because the
  password Sign in locator also matches the passkey button.
- Isolated Playwright smoke with Gotenberg available passed: the ready Preview tab
  and attachment download returned valid PDF responses, explicit Original
  selection survived polling, and the HTML source response was `text/plain`
  with `nosniff` and `sandbox` protection.
- The full Playwright suite now passes global setup with the exact password
  Sign in selector, but 29 existing tests fail against the temporary preview
  tenant because it lacks the suite's expected seeded account/space state and
  other broad locators remain ambiguous.
- Isolated Inbox smoke passed with an uploaded HTML file: Preview and Original tabs
  rendered, Original remained selected after the HTMX swap, the inbox URL kept
  its `preview_tab` state, and the source response remained safe plain text.
- Isolated pending-state smoke passed: Preview was selected by default, status
  appeared only in Preview content, and switching to Original removed it.
- The active development tenant exposed a final-copy backlog: an old missing
  object stopped the copy loop before recent uploads. The copy queue now skips
  unfinished rows, continues past individual copy errors, and prioritizes recent
  uploads.
