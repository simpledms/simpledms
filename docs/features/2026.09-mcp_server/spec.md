# MCP document intake and filing

Date: 2026-09-18  
Status: slices 01–02 implemented, verification pending; remaining slices planned

## Outcome

An MCP client can upload a document into a Space's Inbox, inspect it, assign existing Tags,
properties, and a document type, then file it. The browser shows the same persisted state and
continues to use the same model operations through HTMX.

This is the feature's scope authority. The [architecture proposal][architecture] describes the
shared execution design; the [catalog][catalog] maps tools to existing commands. See
[plan.md](plan.md) for delivery order and [invariants.md](invariants.md) for durable rules.

## Recorded assumptions

These choices allow planning without blocking questions; they are not additional user approvals.

- Initial clients can configure a bearer header. Use the proposed account-owned, revocable,
  single-Space MCP credential, with read-only/read-write modes. OAuth onboarding is later work.
- “Tagging, properties, document types” initially means discovering and applying existing
  definitions. Users create definitions or import templates through the current browser UI.
- “Uploads” initially means new documents sent directly through an MCP tool into Inbox.
  Use a bounded base64 argument, with a proposed 10 MiB decoded ceiling. Large-file streaming,
  URL import, batches, and uploading a new version have separate later contracts.
- The first release includes the entire upload/classify/file journey, not just its first slice.
  Notes and standalone rename are later tools. An optional filename during existing filing is
  retained because it is part of that operation.

## First-release tools

All names below are proposed public tool names. Source mappings and ordinary query inputs/results
are in the catalog; the more specific contracts below override its earlier exploratory wording.

| Capability | Tools |
| --- | --- |
| Identify and inspect | `get_space`, `list_inbox`, `get_file`, `read_file_text` |
| Upload | `upload_file` |
| Tagging | `list_tags`, `assign_tag`, `unassign_tag` |
| Properties | `list_properties`, `set_file_property`, `remove_file_property` |
| Document types | `list_document_types`, `get_document_type` |
| Apply document type | `set_document_type`, `clear_document_type` |
| Filing destinations | `list_directory`, `create_directory` |
| Finish filing | `file_inbox_document`, `mark_inbox_file_done`, `search_files` |

### Connect and inspect

Account settings provide a credential list plus create/revoke actions. Creation collects a label,
an accessible Space, and a mode, defaulting to read-only. Display `/mcp` and the generated token
once with copy controls; list/reload views contain metadata only. No-accessible-Spaces, empty-list,
validation, and revoked states must be usable on desktop and mobile. Creation is unavailable in
a temporary setup Session. Changing scope/mode means creating a replacement credential.

Reuse the existing WebDAV credential page/form/list patterns and widgets locally; MCP has its
own credential model and handlers. A tenant administrator can remove the account's access using
existing controls; tenant-wide MCP credential management is outside this release.

`get_file` returns basic file data and, when classification is delivered, assigned metadata with
public IDs and typed values. `read_file_text` exposes existing file-level OCR with explicit
availability and bounded continuation. Do not wait for OCR before reporting upload success.
The architecture's pagination, public projection, error mapping, and protocol rules apply.

### Upload contract

`upload_file` takes `filename` and `content_base64` using standard padded base64. It accepts one
nonempty document, targets the credential's Space root with Inbox state, and returns durable
`file_id`, final filename, size, and `is_in_inbox`. It takes no server-local path or destination
Space argument. MIME/OCR information may still be pending.

Bound the wire input before SDK decoding and the decoded content before successful persistence:

- Maximum decoded size: 10 MiB, further restricted by the effective configured upload-size limit.
  A configured unlimited upload size does not remove this MCP transport ceiling.
- Maximum HTTP MCP request body: 16 MiB, checked even for missing/chunked Content-Length.
  Check the encoded field length before decoding and enforce the decoded bound as well.
- Reject invalid base64, empty bytes, and invalid basenames. Do not normalize a path into an
  accepted filename. Stream the bounded decoded reader into the existing storage pipeline.

This is a deliberate first-release tradeoff: SDK JSON decoding retains the encoded string, so
this is not end-to-end streaming. Do not add a second full decoded buffer or log tool payloads.
Clients that cannot provide bounded file bytes need a future transfer adapter.

Preserve the current Inbox upload collision policy: a conflicting live Inbox filename fails
without overwrite or silent suffixing. A repeated request is not a deduplication key; clients
can inspect Inbox after a lost response. Register the new immutable file source `MCP` by
appending its enum value, preserving existing stored enum values and source filters.

Share the browser's prepare/upload/finalize orchestration, with MCP credential revalidation in
the fresh authorization step before finalization. Keep the existing storage integrity and
recovery contracts linked from the invariants; do not replace them with a new upload algorithm.

### Classification contract

Tags, properties, and document types need stable public IDs for existing and new rows before
their tools are available. Introduce their migrations and backfill with this behaviour. The
existing required/unique `PublicIDMixin` and its `DefaultFunc` alone do not backfill old rows.
The browser's numeric joins remain internal. Document-type attributes can be inline descriptions
referencing Tag/property public IDs; omit attribute IDs until attribute editing is implemented.

Mutations apply to a live document in the bound Space. Define shared model-level assignment
methods so browser binding and MCP binding cannot make different classification decisions.

- `assign_tag` ensures the direct assignment exists without adding another on repeat.
  `unassign_tag` removes direct assignments and is a no-op when absent. A Group cannot be assigned
  directly. Preserve composed/resolved-Tag behaviour; return direct and resolved Tags distinctly
  so an inherited Tag remaining after unassignment is not reported as a failure.
- `set_document_type` means set, including when already selected; `clear_document_type` means
  clear. The browser may retain its click-to-toggle interaction by choosing the matching shared
  operation within the same transaction. Selection does not automatically create field values or
  remove unrelated metadata. Protected/disabled flags remain visible as configuration metadata;
  this slice must not invent a new selection rule solely from those flags.
- `set_file_property` is an upsert with exactly one typed value field matching the stored property
  type: `text_value` (string), `number_value` (integer), `money_minor_units` (integer), `date_value`
  (`YYYY-MM-DD`), or `checkbox_value` (boolean). Property units supply the meaning of money values.
  Explicit empty text, zero, and false differ from omission; missing/null/wrong-type values fail.
  Integer values must fit storage and the JSON safe-integer range. MCP money uses existing minor
  units directly; keep the browser's decimal-to-minor-unit conversion at its binding boundary.
- `remove_file_property` removes an assignment, including an existing date, and is a no-op when
  absent. Retain the browser's existing empty-date clearing and add-value form validation.

Each tool is its own transaction. A sequence assigning a type, Tag, and properties is not an
atomic batch. Report each committed result so a client can inspect and resume an interrupted
classification sequence. Do not add a generic metadata patch language.

### Filing contract

`mark_inbox_file_done(file_id)` handles completion without a move, including non-folder Spaces.
`file_inbox_document` takes the existing file, destination directory, optional filename, and
optional child-directory name; it handles folder-mode filing. Both use shared lifecycle methods.
The filing rules and preservation requirements are owned by invariant M5.

`create_directory` and `list_directory` supply destinations. `search_files` searches/lists filed
documents with current FTS/sort behaviour and optional public-ID Tag/document-type filters;
advanced property-filter UI state is not part of its initial JSON contract. Empty query lists
filed documents. Inbox remains a distinct query scope. Reuse current query predicates and limit
results instead of creating an independent index or loading all rows.

## Acceptance outcomes

1. A full browser Session can create a scoped credential, inspect its Inbox through a real MCP
   HTTP client, and revoke it; failed access never becomes a browser login redirect.
2. A read/write client can upload bytes and subsequently read that file's metadata in MCP and
   open/download it through the browser. Failed ingestion is handled by the existing recovery
   policy; a read-only credential cannot ingest.
3. Existing and newly created metadata definitions can be discovered and applied using public
   IDs. Repeated explicit assignments do not toggle off or duplicate metadata. The browser and
   MCP show the same persisted type, Tags, and property values after refresh.
4. A client can finish processing in folder and non-folder modes, find the filed document, and
   observe that it disappeared from Inbox. Failed filing leaves the original state intact.
5. Each journey preserves relevant browser desktop/mobile behaviour, scoped authorization,
   bounded results, transactional error handling, and the linked durable rules.

Later catalog candidates include notes, standalone rename/move, Trash, version merges/uploads,
definition management, archive extraction, URL imports, binary downloads, and multi-Space grants.

[architecture]: ../../specs/20260918_mcp_server.md
[catalog]: ../../specs/20260918_mcp_tool_catalog.md
