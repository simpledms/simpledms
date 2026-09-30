# MCP document intake and filing

Date: 2026-09-18  
Status: slices 01–04 reviewed; extensions 05–09 implemented, complete verification pending

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
  definitions. Slice 05 extends this with the browser's definition management and library import
  operations, as requested on 2026-09-29.
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
| Original bytes | `download_file` |
| Tagging | `list_tags`, `assign_tag`, `unassign_tag` |
| Properties | `list_properties`, `set_file_property`, `remove_file_property` |
| Document types | `list_document_types`, `get_document_type` |
| Apply document type | `set_document_type`, `clear_document_type` |
| Filing destinations | `list_directory`, `create_directory` |
| Finish filing | `file_inbox_document`, `mark_inbox_file_done`, `search_files` |
| Manage Tags | `create_tag`, `edit_tag`, `delete_tag`, `create_and_assign_tag` |
| Organize Tags | `move_tag_to_group`, `assign_sub_tag`, `unassign_sub_tag` |
| Manage fields | `create_property`, `edit_property`, `delete_property` |
| Manage document types | `create_document_type`, `rename_document_type`, `delete_document_type` |
| Configure attributes | `create_document_type_tag_attribute`, `edit_document_type_tag_attribute`, `create_document_type_property_attribute`, `edit_document_type_property_attribute`, `delete_document_type_attribute` |
| Library templates | `list_document_type_templates`, `import_document_types` |
| Notes and history | `list_document_notes`, `get_document_note`, `create_document_note`, `edit_document_note`, `replace_document_note`, `delete_document_note` |
| Filed organization | `rename_file`, `move_file` |

### Connect and inspect

Account settings provide a credential list plus create/edit-label/revoke actions. Creation collects a label,
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

### Metadata management contract

Slice 05 exposes the existing metadata-management model operations. Mutations require a read/write
credential and current access to its bound Space, with the same model/privacy rules as the UI.
Creation takes no tenant/Space selector. Every existing definition, document, group, and composition
reference is resolved inside that Space; broader account membership cannot expand the token.

- Names must be nonblank and at most 300 Unicode characters; field units are bounded to 300.
  Public references are required where applicable and bounded to 100 bytes.
- Tag creation supports `Simple`, `Group`, and `Super` (composed) types and an optional group public
  ID. Editing changes only the name. `move_tag_to_group` accepts an optional group ID; omission or
  an empty value removes grouping. Groups cannot be nested. Composition connects a composed Tag
  to simple sub-Tags. `create_and_assign_tag` creates an assignable Tag and assigns it to the
  document in one transaction, without leaving a definition behind on failure.
- Field creation supports `Text`, `Number`, `Money`, `Date`, and `Checkbox`. Editing changes name
  and optionally unit: an omitted unit is preserved and an explicit empty string clears it. Type
  changes are unavailable, matching the UI's protection of stored values.
- Document types can be created, renamed, and deleted. Attribute creation/editing/removal uses
  the document-type public ID together with exactly one Tag-group or field public ID. These pairs
  are already unique in the schema, so no standalone attribute identifier is exposed. Tag
  attributes have editable name and name-giving state; field attributes have editable name-giving
  state. Edit tools require the boolean explicitly, including `false`. Protected/disabled/required
  flags remain descriptive where the UI provides no editing operation.
- Deletion retains existing foreign-key restrictions. Tools do not clear classification, field
  values, or dependent definitions to force a deletion. Duplicate/in-use constraint failures
  return an ordinary metadata business error, with no raw SQL or partial success.
- Template discovery returns existing library keys and localized names. Import accepts 1–64
  advertised keys and requires a Space with no metadata, as the UI does. Unknown keys fail before
  any import is performed. Imported definitions use the same public-ID defaults as UI creation.

Read discovery remains available to read-only credentials. Results are committed metadata
projections, and renaming preserves public identity. See the [attribute-reference ADR][attribute-adr].

### Filing contract

`mark_inbox_file_done(file_id)` handles completion without a move, including non-folder Spaces.
`file_inbox_document` takes the existing file, destination directory, optional filename, and
optional child-directory name; it handles folder-mode filing. Both use shared lifecycle methods.
The filing rules and preservation requirements are owned by invariant M5.

`create_directory` and `list_directory` supply destinations. `search_files` searches/lists filed
documents with current FTS/sort behaviour and optional public-ID Tag/document-type filters.
Extension 07 adds explicit typed field filters; browser URL/filter state stays in the UI adapter.
Empty query lists
filed documents. Inbox remains a distinct query scope. Reuse current query predicates and limit
results instead of creating an independent index or loading all rows.

### Original-byte download contract

`download_file(file_id, version_number?, offset?, length?)` reads original plaintext bytes through
the same decrypt/decompress reader as browser downloads. It is available to read-only credentials.
The result contains `content_base64`, file/name/MIME/size metadata, the selected version number,
the byte offset, and `has_more`/`next_offset`. The whole-version content hash is included when known.

Each call returns at most 1 MiB decoded bytes, also the default length. Offset zero without a
version selects the latest version. A continuation requires the returned version number; explicitly
selected versions must be positive. Offsets use bytes and must be within the recorded file size
and JSON's safe-integer range. Reading exactly at EOF returns an empty final chunk. Unknown,
deleted, directory, or foreign-Space files and unknown versions fail without content disclosure.
Storage paths, presigned URLs, and browser Session cookies are not part of this contract.

### Typed field-search contract

`search_files` additionally accepts up to 32 `property_filters`, ANDed with the existing text,
Tag/document-type filters and each other before pagination. Each filter takes a property public
ID, an operator, and exactly one typed value matching its stored type. Money uses integer minor
units, dates use `YYYY-MM-DD`, and zero/false/empty text remain distinct from missing values.

- Text: `equals`, `contains`, `starts_with`, using the existing case-insensitive predicates.
- Number/Money/Date: `equals`, `greater_than`, `less_than`, `greater_than_or_equal`,
  `less_than_or_equal`, `between`. Inclusive `between` needs the matching `end_number_value`,
  `end_money_minor_units`, or `end_date_value` and ordered bounds.
- Checkbox: `equals` or `is_checked`; false includes documents without an assignment of that
  Checkbox, including those with unrelated fields assigned.

Text filter values are limited to 1,000 Unicode characters. Unsupported operators, mismatched or
omitted values, invalid dates/integers/ranges, and foreign field IDs fail rather than silently
ignoring a requested condition. Browse and MCP share typed predicates; browser decimal money
and open date bounds remain adapter-level normalization.

### Notes and history contract

Note reads/mutations call `DocumentNotes`, retaining current authorship, owner, historical-entry,
and Trash rules. A writable token does not grant permission to edit another author's note. Edits
preserve attribution, replacement retains the predecessor, and deletion retains read-only history.
Every note ID is paired with a scoped file public ID. The document-scoped `legacy` selector keeps
unknown authorship/date; reading it does not materialize a row. Owner-authorized changes use the
existing materialization behavior. Historical mutation conflicts use the `conflict` error code.

`list_document_notes(file_id, show_history?, offset?, limit?)` pages real note rows (default 50,
maximum 100), with 1,000-character previews and separate `legacy_note` on page zero. Body
continuation metadata points to `get_document_note(file_id, note_id, offset?, length?)`, which
uses character offsets, default length 12,000, and maximum 50,000. New/edited titles are bounded
to 300 characters and bodies to 50,000. Author/editor projections contain public user IDs and
display names, with nullable original timestamps preserved. `can_change` also reflects token mode.
Notes in Trash remain readable through existing access but cannot be mutated.

### Filed organization contract

`rename_file(file_id, new_filename)` and
`move_file(file_id, destination_directory_id, filename?, new_directory_name?)` organize live,
already-filed documents and ordinary directories. They do not complete Inbox items or organize
the Space root. Move requires folder mode; rename retains the filesystem's existing non-folder
behavior. All endpoints resolve through the bound Space. Existing filename, conflict, current
location, and cycle rules apply, with one transaction including requested child-directory creation.
Moving into a newly created child of the current parent is an actual move and is supported.
Identity, source, versions, classification, values, notes/history, and child-parent links survive.
Results report public ID, final name/parent, and Inbox state after commit; conflicts disclose no SQL.

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
6. A writable client can manage the same Tag/field/document-type configuration as the browser,
   including grouping, composition, attributes, and library import, without accessing another Space.

Later catalog candidates include Trash, version merges/uploads, duplicate discovery,
archive extraction, URL imports, large-file streaming transfers, and multi-Space grants.

[architecture]: ../../specs/20260918_mcp_server.md
[catalog]: ../../specs/20260918_mcp_tool_catalog.md
[attribute-adr]: ../../adrs/2026.09-mcp_metadata_attribute_references.md
