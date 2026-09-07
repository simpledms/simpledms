# PDF Preview Streaming

Status: Proposed; blocked on end-to-end feasibility

Date: 2026-09-05

This directory is the stable home for this feature throughout its lifetime.
This document defines required behaviour, not an implementation plan or an
architecture decision.

## Outcome

Users can start reading a large PDF without waiting for the whole PDF to be
downloaded to the browser or read and decoded from storage by the server.
Subsequent pages can retrieve the needed bytes through HTTP byte-range requests.

End-to-end efficiency is a go/no-go requirement. If it cannot be achieved without
violating the protection and compatibility requirements below, do not implement
this feature. A browser-only improvement is not an acceptable reduced scope.

## Existing Behaviour

- `core/ui/widget/file_preview.gohtml` embeds documents in an HTML `object`,
  preferring the browser's native PDF viewer. Its PDF.js fallback is intended for
  browsers without native embedded viewing, notably mobile.
- PDF.js already receives a URL through `getDocument(pdfUrl)`, not a fully
  buffered application-provided byte array. After loading the document, the
  fallback immediately requests and renders every page.
- Original PDFs use `action/download/download.go` and
  `action/common/download_helper.go`. The latter opens the stored file and
  progressively copies its bytes into a `200 OK` response. It does not handle
  byte ranges or set the decoded content length.
- Generated PDF previews use `action/download/preview.go`, which also copies
  bytes into a `200 OK` response without handling ranges. This path sets a
  content length when the preview's recorded decoded size is positive.
- `model/tenant/filesystem/s3_file_system.go` reads an S3 object, decrypts it with
  age unless file encryption is disabled, and decompresses gzip through a pipe.
  The returned reader is sequential, not a seekable PDF representation. PDF
  offsets do not correspond directly to offsets in the stored object.
- `server/server.go` wraps normal routes in HTTP compression middleware. Storage
  compression and HTTP content encoding are separate concerns.
- `action/browse/file_preview_partial.go` composes original and generated
  previews. `action/browse/file_version_preview_dialog.go` reuses that composition
  for a specific historical version.
- Public download and preview URLs are scoped to a tenant, Space, and source
  file, with an optional version number. Generated artifacts are resolved only
  after looking up the authorized source.

These observations identify limitations in the current delivery path. No browser
timing or storage-throughput measurement was performed during specification.

## Confirmed Behaviour

1. Load pages on demand, with limited prefetch acceptable, rather than eagerly
   downloading every page when a preview is opened.
2. Keep the native browser PDF viewer preferred where supported, with PDF.js as
   the fallback. Native viewers retain control over their own prefetch policy;
   application-controlled on-demand behaviour applies to PDF.js.
3. Improve the complete storage-to-server-to-browser path, including cold
   previews. If that is not feasible, do not implement the feature.

## Scope And Safe Assumptions

- Include uploaded PDFs and ready generated PDF previews, in both the main
  preview and historical-version dialog. Users should not need to know which
  kind of PDF they are viewing to benefit.
- Include existing documents as well as newly stored documents. Any required
  transition for existing storage remains an unresolved decision, not permission
  to rewrite existing data.
- A cold preview has no reusable per-document runtime cache. Success must not
  depend on a previous viewer having already downloaded or decoded the file.
- Small PDFs may be transferred completely when that is naturally cheaper than
  splitting them. No universal first-page timing or byte-percentage target is
  assumed, because PDF structure and page dependencies vary.
- Ordinary non-linearized PDFs are in scope. A PDF may intrinsically require
  bytes from its end or shared resources; that is different from an avoidable
  full-file read caused by the storage representation or delivery code.
- Preserve existing download actions, viewer controls, Preview/Original
  selection, and conversion status/retry behaviour. No new user setting is
  required to enable efficient preview delivery.

## Requirements

### Progressive And On-Demand Viewing

1. A large, valid PDF whose first visible page needs only part of its bytes must
   render that page before the entire PDF has reached the browser, including
   when the PDF has not been previewed since application startup.
2. PDF.js must prioritize visible pages and must not request or render all pages
   solely because the document has opened. Limited adjacent-page prefetch is
   permitted, but must not become an unconditional background full download.
3. Scrolling to later pages must retrieve missing data and render the correct
   pages without restarting a full document download.
4. Do not run a redundant application-owned full-document loader alongside a
   working native viewer.
5. Closing, replacing, or navigating away from a PDF.js preview must stop its
   outstanding loading/rendering work and release associated resources.
6. Loading failures must not leave an indefinite blank or loading state. Preserve
   a usable full-download path and communicate failures without exposing storage
   details. Existing conversion failures retain their existing behaviour.

### HTTP Delivery

1. Both original-PDF inline delivery and generated-PDF inline delivery must
   support HTTP byte ranges compatible with PDF.js and native browser viewers.
2. A satisfiable single byte range must return `206 Partial Content`, the exact
   requested decoded PDF bytes, a correct `Content-Range`, and a matching
   `Content-Length`. Support bounded, open-ended, and suffix byte ranges.
3. Advertise `Accept-Ranges: bytes` for range-capable PDF responses. An authorized
   unsatisfiable range must return `416 Range Not Satisfiable` and
   `Content-Range: bytes */<decoded-length>`.
4. Ordinary requests without ranges must continue to return a complete usable
   PDF. `HEAD` must expose the corresponding full-response metadata without
   sending a body or reading and decoding the whole stored object.
5. Invalid, overlapping, excessive, or multi-range requests must have bounded,
   standards-compliant handling. Multipart range support is not itself required
   by this feature; malformed input must not cause crashes or unbounded work.
6. Content type, range offsets, lengths, and content encoding must consistently
   describe the PDF representation delivered to the client, not encrypted or
   gzip-compressed storage bytes. HTTP compression must not break range use.
7. All ranges assembled into one displayed document must belong to the same
   exact version and PDF representation. A concurrent new version or artifact
   replacement must not produce mixed bytes or silently switch document content.

### End-To-End Efficiency

1. Serving the first useful bytes of a cold preview must not require reading,
   decrypting, decompressing, or staging the entire stored document first.
2. A small range near the end of a large PDF must not require a sequential scan
   from the start of the stored document. Repeated ranges must not repeatedly
   read or decode the entire object, or its growing prefix.
3. Storage reads and decoding must be related to requested data plus bounded
   metadata and processing overhead, rather than inherently proportional to the
   whole document for every request.
4. Per-request memory and temporary-storage use must be bounded; loading a
   complete PDF into memory or a temporary file merely to answer ranges does not
   satisfy this requirement.
5. Client cancellation must stop unnecessary storage reads and decoding. Any
   retained resources must have a bounded lifecycle and preserve tenant isolation.
6. The encrypted deployment path must qualify, not just deployments with file
   encryption disabled. Turning off encryption or compression is not an approved
   shortcut.
7. Evidence must cover cold storage access and decoded work as well as browser
   network traffic. A `206` response or warm-cache speedup alone is insufficient.

### Security And Compatibility

1. Authenticate and authorize every body or metadata request through the existing
   source file, tenant, Space, and version access boundary, including repeated
   ranges. Previously permitted access must not authorize a later request after
   that access has been revoked.
2. Preserve configured at-rest protection, integrity guarantees, tenant isolation,
   and file-storage lifecycle rules. This specification does not authorize
   persistent plaintext copies, public storage URLs, or independent artifact-ID
   access.
3. Existing stored content must remain readable. Originals, hashes, filenames,
   source metadata, and version history must not be changed by viewing a PDF.
4. Keep original downloads and generated-PDF downloads complete and correct,
   including filenames and inline/attachment semantics.
5. Do not regress non-PDF previews or ordinary downloads that share delivery
   code. Conversion availability must remain independent of original access.

## Acceptance Criteria

1. With cold runtime caches and a large multi-page PDF, the first visible page
   renders while only part of the PDF has reached the browser and without a full
   storage read/decode prerequisite. This holds for original and generated PDFs,
   current and historical versions, and encrypted storage.
2. In the PDF.js fallback, opening the preview and leaving it on the first page
   does not request/render every page or continue fetching the entire document
   merely to fill a background cache. Scrolling later retrieves needed bytes.
3. The native viewer remains preferred where supported, retains its existing
   controls, and can receive correct range responses. Its own prefetch behaviour
   is not treated as a failure of the PDF.js on-demand requirement.
4. Authorized bounded, open-ended, and suffix ranges return byte-for-byte correct
   `206` responses. Unsatisfiable ranges return `416` with the decoded length.
   Ordinary GET and HEAD responses preserve the specified full-file semantics.
5. A cold small tail-range request does not read/decode the entire file or scan
   its prefix. Repeated separated ranges demonstrate bounded overhead rather
   than repeated whole-object work; memory does not grow with full PDF size.
6. The same HTTP semantics hold through the application's normal middleware,
   including clients that advertise compression support.
7. Unauthorized, cross-tenant, cross-Space, and revoked-access requests cannot
   retrieve PDF bytes or protected document metadata through range or HEAD
   requests. Historical-version selection never serves the current version.
8. If a new version or replacement artifact appears between requests, the
   displayed PDF remains consistent or the load fails/restarts safely; ranges
   from different representations are never assembled together.
9. Closing or switching a PDF.js preview stops obsolete requests and rendering;
   failures do not silently leave an indefinite loading state. Explicit downloads
   remain usable whenever the underlying authorized file remains available.
10. Existing PDFs, generated previews, source downloads, non-PDF previews, and
    conversion status/retry flows remain usable without changing canonical bytes
    or weakening storage protection.
11. If the end-to-end criteria cannot be met, the feature is not implemented or
    relabelled as a browser-only optimization.

## Non-Goals

- Replacing native desktop PDF viewing with PDF.js everywhere.
- Adding a new PDF viewer toolbar, search, annotation, editing, or offline mode.
- Forcing browser-native viewers to follow an application prefetch policy.
- Improving only browser traffic while retaining full-file server preparation.
- Treating a warm full-document cache as proof of end-to-end streaming.
- Redesigning Gotenberg conversion, its eligibility rules, or retry lifecycle.
- Optimizing video, audio, WebDAV, or general download-resume behaviour.
- Guaranteeing that every possible PDF renders from a fixed small byte fraction.
- Selecting a storage format, cache design, migration, or implementation sequence
  in this specification.

## Durable Invariants

1. Original content remains canonical and unchanged by preview delivery.
2. Preview bytes and metadata are always authorized through the source context;
   range access is not a separate security boundary.
3. A displayed document consists of bytes from one exact representation of one
   source version.
4. HTTP PDF offsets and lengths describe the decoded PDF representation.
5. Efficient access never weakens configured storage protection or crosses tenant
   boundaries.
6. The existing [file-storage safety invariants](../../invariants/file_storage_safety.md)
   and [generated-preview invariants](../../invariants/gotenberg_pdf_previews.md)
   remain authoritative. A conflicting design requires an explicit decision,
   not silent relaxation.
7. End-to-end efficiency remains a release gate; browser-only partial success
   must not be presented as completion of this feature.

## Unresolved Questions

1. Can bounded random access to decoded PDFs be achieved with the current
   gzip-plus-age storage representation and required protection guarantees? The
   inspected sequential reader does not provide it; feasibility is not yet
   established.
2. If a different stored representation or durable supporting data is necessary,
   what transition for existing PDFs is acceptable, including preparation time,
   extra storage/quota cost, backup/restore compatibility, and availability while
   preparation is incomplete? No migration or duplicate representation is
   approved by this specification.
3. What concrete limits on prefetch, range overhead, memory, and retained
   resources demonstrate bounded behaviour for the supported storage system?
   These limits must be explicit before implementation acceptance, not inferred
   from HTTP status codes or a warm browser measurement.

These material storage decisions require a separate architecture follow-up using
this specification path and these unresolved questions. No architecture pass,
implementation plan, or feasibility claim is included here. If a feasible design
cannot satisfy the confirmed end-to-end requirement, close the feature without
implementation.
