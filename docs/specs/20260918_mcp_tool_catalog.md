# Useful MCP tools from existing SimpleDMS commands

Date: 2026-09-18  
Status: proposed

This catalog maps useful MCP operations to current code. Slices 01–03 now register the connection,
Inbox, upload, and classification tools through `clear_document_type`; those implementations are
awaiting verification. Filing and later entries remain proposed. Read operations are included
because an agent needs to discover files and metadata before issuing commands.

See the [architecture proposal](20260918_mcp_server.md) for shared execution, authentication,
public IDs, bounded results, error handling, and the single
[source-review record](20260918_mcp_server.md#source-review-and-execution-record).
The [feature specification](../features/2026.09-mcp_server/spec.md) is the first-release scope
authority; its [plan](../features/2026.09-mcp_server/plan.md) orders implementation.

## Priority and readiness

- **First:** upload, Tag/property/document-type discovery and assignment, filing, and the reads
  supporting that complete journey. These tools are delivered across the first-release slices.
- **Later:** notes, standalone rename/move, recovery, versions, and configuration/transfer extensions.

Readiness describes required work, not implementation status:

- **Reuse:** a suitable model operation already exists; add the protocol adapter and projection.
- **Extract:** behavior/query code is still coupled to an HTTP handler or partial.
- **Adapt:** current semantics are browser-specific or wider than the proposed MCP scope.
- **Public IDs:** requires the identifier work described in the architecture proposal.

All tools initially operate in the credential's single authorized Space. Parameters ending in
`_id` mean public identifiers, not the numeric IDs used by some existing forms. Inputs below omit
common pagination fields where the architecture already defines them. Optional fields use `?`.

## First: find and understand documents

### `get_space` — read; extract a small data projection

- Current basis: [`SpaceContext`][space-context] and [`SpaceCardsPartial`][space-cards].
- Input: none. Result: tenant/Space public IDs and names, Space description, folder-mode state,
  root-directory public ID, and the connection's read/write mode.
- Purpose: identify the connection's scope and discover the root without guessing identifiers.
  This is a new data projection over existing information, not an existing `*Cmd`.

### `search_files` — read; extract

- Current basis: [`ListDirPartial`][browse-list] and [`ListDirFileQueryService`][browse-query].
- Input: `query?`, `sort?`, pagination. Initially search/list filed documents in the bound Space;
  an empty query lists them. Result: file summaries with public IDs, names, and parent references.
- Purpose: find documents by filename/content using existing FTS semantics. Browse currently
  excludes Inbox files; use `list_inbox` for those. Preserve search sanitization and rank behavior.
- Include public-ID Tag/document-type filters with the filing journey. Advanced property filters
  are later query extensions; reuse the same extracted implementation.

### `list_inbox` — read; extract

- Current basis: [`inbox.FilesListPartial`][inbox-list].
- Input: `query?`, `sort?`, `sources?`, pagination. Result: Inbox file summaries with canonical
  browser URLs and source values.
- Purpose: inspect pending documents and find imports needing classification. Reuse the current
  source enum, search, and sorting rather than the browser's URL/filter state.

### `get_file` — read; extract

- Current basis: [`FileInfoPartial`][file-info], [`FilePropertiesPartial`][file-properties],
  file metadata/Tag partials, and [`FileRepository`][file-repo].
- Input: `file_id`. Result: name, canonical browser URL, parent, Inbox state, source, current
  version, MIME type, size, content hash when present, OCR availability, and assigned metadata.
- Purpose: understand a document before changing it. Project an explicit DTO; omit metadata's
  internal IDs until public IDs are available. Initial lookup covers live files, including Inbox.
  Folder results must avoid methods that assume a stored file/current version exists.

### `read_file_text` — read; extract

- Current basis: [`OCRContentDialog`][ocr] and the file's stored `OcrContent`/OCR status.
- Input: `file_id`, `offset?`, `length?`. Result: bounded plain text, current-version reference,
  availability/status, and continuation information.
- Purpose: summarize, classify, or extract information from already processed documents.
  This reads existing file-level OCR; it does not run new OCR or provide historical-version text.

## First: upload a document into Inbox

### `upload_file` — write; extract shared orchestration and bind MCP bytes

- Current commands: [`inbox.UploadFileCmd`][inbox-upload] and [`browse.UploadFileCmd`][upload].
- Input: `filename`, `content_base64`. Result: a durable file public ID, canonical browser URL,
  filename, size, and Inbox state. The [feature upload contract][upload-contract] defines size
  bounds and failure behaviour.
- Purpose: deliver new documents directly through MCP. Reuse prepare/upload/finalize, add the
  `MCP` source, and revalidate the credential before finalization.
- This is a native bounded tool, not a multipart form, temporary upload token, or client-local
  path. It does not require a new out-of-band HTTP transfer protocol.

## First: discover metadata and filing destinations

### `list_directory` — read; extract

- Current basis: [`ListDirFileQueryService`][browse-query].
- Input: `directory_id?` (root by default), pagination. Result: immediate file/folder children.
- Purpose: find a filing destination in folder mode. Extract an explicit directory query instead
  of trying to reproduce the UI's automatically calculated recursive/filter state.

### `list_tags` — read; extract, public IDs

- Current basis: [`ManageTags` actions][manage-tags] and [`TagService`][tags].
- Input: `group_id?`, pagination. Result: Tag public IDs, names, types, groups, and composition
  references as relevant.
- Purpose: discover valid existing Tags before assignment. Keep Tag groups and composed Tags
  distinguishable from simple assignable Tags.

### `list_properties` — read; extract, public IDs

- Current basis: [`property` actions][properties] and the Space's property query.
- Input: pagination. Result: property public IDs, names, types, and units.
- Purpose: discover which typed fields can be set. Use the existing types: Text, Number, Money,
  Date, and Checkbox.

### `list_document_types`, `get_document_type` — read; extract, public IDs

- Current basis: [`documenttype` actions][document-types], including list, details, and attributes.
- Input: pagination for listing; `document_type_id` for details. Result: names, public IDs,
  protected/disabled state, and applicable Tag/property attributes.
- Purpose: select an existing classification and understand its configured metadata. Attribute
  descriptions omit attribute IDs in the first release; Tag/property references use public IDs.

## First: classify and file documents

### `assign_tag`, `unassign_tag` — write; reuse, public IDs

- Current commands: [`AssignTagCmd`][assign-tag], [`UnassignTagCmd`][unassign-tag].
- Shared operations: `TagService.AssignToFile` and `UnassignFromFile`, strengthened as specified.
- Input: `file_id`, `tag_id`. Result: the affected assignment/public identifiers.
- Purpose: classify documents with known Tags. Prefer explicit assign/unassign operations to
  `ToggleFileTagCmd`, whose result depends on previous state. Resolve both references within the
  bound Space. The current repository can insert duplicates; add the feature's shared
  desired-assignment behaviour rather than claiming the current method is already idempotent.

### `set_file_property`, `remove_file_property` — write; extract, public IDs

- Current commands: [`SetFilePropertyCmd`][set-property],
  [`AddFilePropertyValueCmd`][add-property-value], [`RemoveFilePropertyCmd`][remove-property].
- Input: `file_id`, `property_id`, and one typed value field for set. Result: value or removal.
- Purpose: record invoice dates, amounts, reference numbers, or review Checkboxes.
- The two set/add-value handlers both create or update assignments. Extract a shared model
  operation while retaining their browser-specific input handling. Validate against the stored
  property type; distinguish absent values from valid `0` and `false`. Date values use the same
  parsing/semantics as the UI. Keep the existing cleared-date removal behavior in the browser
  adapter and provide the explicit remove operation for MCP.
- `AddFilePropertyCmd` itself only renders the property picker; it is not another mutation tool.

### `set_document_type`, `clear_document_type` — write; adapt, public IDs

- Current command: [`SelectDocumentTypeCmd`][select-document-type].
- Input: `file_id`, plus `document_type_id` for set. Result: the resulting classification.
- Purpose: classify a document without depending on its previous selection state.
- Current behavior toggles off when the submitted type is already selected. Extract explicit
  set/clear model methods for MCP. Preserve browser toggle behavior by making the UI adapter
  select the appropriate operation; do not expose the toggle as a supposedly deterministic set.

### `mark_inbox_file_done` — write; extract

- Current command: [`inbox.MarkAsDoneCmd`][mark-done].
- Input: `file_id`. Result: file public ID, parent reference, and `is_in_inbox: false`.
- Purpose: finish processing without moving the document. Current behavior requires the file
  to be in Inbox and clears the flag; it does not require a document type or filled metadata.
  Move that lifecycle rule out of the HTTP handler before exposing it.

### `file_inbox_document` — write; extract

- Current commands: [`inbox.MoveFileCmd`][inbox-move] and [`AssignFileCmd`][inbox-assign].
- Input: `file_id`, `destination_directory_id`, `filename?`, `new_directory_name?`.
  Result: final name/parent and Inbox state.
- Purpose: file an Inbox document, optionally rename it or create the destination child folder,
  and mark it done in one transaction. Reuse filesystem `Move` and centralize the Inbox transition.
- This is a folder-mode operation. Use one shared filing operation for the two existing UI entry
  points; the current `MoveFileCmd` explicitly checks Inbox membership, while `AssignFileCmd`
  does not perform the same explicit check. Define that precondition centrally during extraction.

### `create_directory` — write; reuse

- Current command: [`MakeDirCmd`][make-dir], using filesystem `MakeDir`.
- Input: parent public ID and directory name. Result: folder public ID, name, and parent.
- Purpose: create a destination when filing in folder mode.

## Later: notes, standalone organization, and recovery

### `list_document_notes`, `create_document_note` — read/write; reuse

- Current basis: [`DocumentNotes.List/Create`][notes], [`DocumentNotesPartial`][notes-partial],
  and [`DocumentNoteCmd`][note-cmd].
- Listing takes `file_id`, `show_history?`, and pagination, and returns notes/history plus legacy
  text without materializing it. Creation takes `file_id`, `title`, and `body`, and returns the
  authored note. Repeated creation creates another note.
- Useful later for summaries and review annotations, but not part of initial intake/filing.

### `edit_document_note`, `replace_document_note`, `delete_document_note` — write; reuse

- Current command: [`DocumentNoteCmd`][note-cmd], matching operations on [`DocumentNotes`][notes].
- Input: `file_id`, `note_id`; title/body for edit or replace. Result: the changed note, successor,
  or deletion state. Retain the current model's attribution, history, and permission rules.
- The document-scoped `legacy` selector, if exposed, remains a reserved selector, not a public ID.

### `rename_file` — write; reuse

- Current command: [`browse.RenameFileCmd`][rename], using filesystem `Rename`.
- Input: `file_id`, `new_filename`. Result: file public ID and resulting name.
- This standalone tool is deferred. The existing optional name in `file_inbox_document` remains
  available. Rename currently also handles directories and rejects an unchanged name.

### `move_file` — write; reuse with extraction of handler guards

- Current command: [`browse.MoveFileCmd`][move].
- Input: file/destination IDs and optional new name. Result: final public ID, name, and parent.
- Purpose: organize filed documents in folder mode. Reuse filesystem `Move`/`MakeDir` and their
  folder/cycle checks. Keep Inbox filing as the distinct operation above.

### `list_trash` — read; extract

- Current basis: [`TrashListPartial`][trash-list].
- Input: pagination. Result: recoverable deleted file summaries and deletion information.
- Purpose: discover IDs for restoration without including Trash in ordinary file search.

### `trash_file`, `restore_file` — write; extract

- Current commands: [`DeleteFileCmd`][trash], [`RestoreFileCmd`][restore].
- Input: `file_id`. Result: final deletion/Inbox state and parent public ID.
- Purpose: remove documents from active work and recover them later.
- Deletion is soft deletion, with the current actor recorded; only empty folders can be deleted.
  Restoration currently rejects folders. If the original parent is missing, a file is restored
  to the Space root in Inbox. Extract these decisions together with their actor attribution.

## Later: versions, transfers, and processing

### `list_file_versions` — read; extract

- Current basis: [`FileVersionsPartial`][versions].
- Input: `file_id`, pagination. Result: version numbers, timestamps, original filenames, MIME
  types, sizes, and available hashes. Select by file public ID plus version number.

### `find_duplicates` — read; adapt

- Current basis: [`DuplicateMatchesPartial`][duplicates-partial] and
  [`DuplicateDetectionService.FindDuplicates`][duplicates].
- Input: `file_id`, pagination. Result: hash availability and matching file/version references.
- The current service searches account-accessible Spaces within the tenant, including historical
  versions. Restrict its query to credential-authorized Spaces before counting/paginating; a
  Space-bound tool cannot expose the current service's complete result unchanged.

### `merge_inbox_file_as_version` — write; reuse plus command preconditions

- Current command: [`FileVersionFromInboxCmd`][merge-cmd].
- Shared operation: [`FileVersionFromInboxService.MergeFromInbox`][merge].
- Input: `source_file_id`, `target_file_id`, and the existing metadata-loss acknowledgement.
  Result: target file public ID/current version and confirmation that the source was consumed.
- Purpose: replace an existing document's content using an Inbox arrival.
- This has more effects than uploading a version: it uses the source's latest stored version,
  updates the target name/OCR, transfers note history, removes source version links, and deletes
  the source logical file. Source Tags/properties are not transferred. Preserve the existing
  `ConfirmWarning` requirement and execute all changes in the same tenant transaction.

### `retry_pdf_preview` — write; extract

- Current command: [`RetryPDFPreviewCmd`][retry-preview].
- Input: `file_id`, `version_number?`. Result: queued/pending preview state.
- Purpose: retry a failed preview conversion. Current code requires configured Gotenberg and an
  existing failed conversion. A queued retry is not a completed PDF result.

### `unzip_archive` — write; extract manual workflow

- Current command: [`UnzipArchiveCmd`][unzip].
- Input: `file_id`, `delete_on_success?`. Result: created file references and archive state.
- Purpose: ingest a document batch already stored as a ZIP. Preserve extraction location/folder
  behavior, limits, source attribution, rollback, and cleanup from the current workflow.
- It performs storage I/O and manually manages transactions. It cannot use the ordinary short
  mutation wrapper around the whole operation.

### `import_file_from_url` — write; adapt staged import

- Current basis: [`UploadFromURLCmd`][url-import] and the [`openfile` flow][openfile].
- Proposed input: `url`; target is the bound Space's Inbox. Result: durable file public ID after
  completed persistence, not a temporary upload token or browser redirect.
- Purpose: import a remotely accessible document. The existing command first stages an
  account-temporary file and asks the browser to select a Space. Tool orchestration must complete
  that workflow while retaining its URL validation and upload/finalization guarantees.

### Version, batch, and large-file uploads — write; later transfer contracts

- Current basis: [`UploadFileVersionCmd`][upload-version] and
  [`openfile.UploadFilesCmd`][shared-upload], plus the shared storage pipeline.
- A future `upload_file_version` selects the target by file public ID. Multi-file/batch semantics
  and a streaming transfer for files above the native tool limit remain separate contracts.
- New single-file Inbox ingestion is already covered by the first-release `upload_file` tool.

### File download — read; new MCP-accessible delivery contract required

- Current basis: [`download.Download`][download] and the existing streaming download helpers.
- Selector: `file_id`, `version_number?`. Result: an authenticated transfer or a bounded resource
  appropriate for the client, once that delivery contract is implemented.
- Purpose: obtain original bytes when OCR text is insufficient. Existing browser links and
  internal encrypted-storage object keys are not an MCP download API.

## Implemented: metadata configuration tools

Added in slice 05 at the user's request. All configuration changes retain the UI's model/privacy
rules and the credential's read/write mode and single-Space scope. The [management contract][management]
specifies bounds, constraint behavior, and composite attribute references.

| Tools | Current commands / model |
| --- | --- |
| `create_tag`, `edit_tag`, `delete_tag` | [Tag commands][tag-actions], `TagService` |
| `create_and_assign_tag` | [`CreateAndAssignTagCmd`][create-assign-tag] |
| `move_tag_to_group` | [`MoveTagToGroupCmd`][move-tag-group] |
| `assign_sub_tag`, `unassign_sub_tag` | [Sub-Tag commands][tag-actions], `TagService` |
| `create_property`, `edit_property`, `delete_property` | [Property commands][properties] |
| `create_document_type`, `rename_document_type` | [Type commands][document-types] |
| `delete_document_type` | [Type commands][document-types] |
| `list_document_type_templates`, `import_document_types` | [`ImportFromLibraryCmd`][import-types], library service |
| `create_document_type_tag_attribute`, `edit_document_type_tag_attribute` | [Tag attribute commands][document-types] |
| `create_document_type_property_attribute`, `edit_document_type_property_attribute` | [Field attribute commands][document-types] |
| `delete_document_type_attribute` | [Attribute commands][document-types] |

Contracts:

- Tag creation takes a name, existing Tag type, and optional group public ID; edit takes an ID and
  name. Group moves and composition take explicit public-ID pairs. Return the resulting Tag data.
- Create-and-assign takes Tag creation data plus a file ID and remains one atomic operation.
  Restrict this tool to assignable types. The current browser handler creates a Group without
  assigning it; group creation is better expressed through `create_tag` in MCP.
- Property creation takes name/type/unit; editing takes ID/name/unit. Current property editing
  intentionally cannot change its type. Return the property definition.
- Document-type creation/rename takes a name; deletion takes an ID. Attribute editing needs a
  separate typed contract for Tag versus property attributes. Select an attribute with the
  document-type public ID and its Tag/field public ID, rather than an internal attribute ID.
- Library import takes the existing template keys and returns imported definitions. Template
  keys are library identifiers, not numeric database IDs; provide template discovery when exposing
  this operation so the client can choose valid keys.

`delete_document_type_attribute` takes `document_type_id` and exactly one `tag_id` or `property_id`.
Attribute mutations return the updated document-type projection. Definition deletions return
`deleted: true` after commit. Field editing preserves an omitted unit and clears an explicit
empty unit; changing a field's type is unavailable.

[management]: ../features/2026.09-mcp_server/spec.md#metadata-management-contract

## Existing commands that should stay outside the initial tool set

### Presentation-only commands

- `ChangeDirCmd`, `SelectDirCmd`, `SelectDirMakeDirCmd`: browser navigation/picker composition.
  Use data queries and `create_directory` instead of publishing picker actions.
- `ToggleTagFilterCmd`, `ToggleDocumentTypeFilterCmd`, `TogglePropertyFilterCmd`, and
  `UpdatePropertyFilterCmd`: turn their useful filters into explicit query arguments.
- `ToggleTagGroupCmd`: expands/collapses groups in browser URL state.
- `UpdateFileListPreferencesCmd`: changes browser table/list preferences.
- Forms, dialogs, sheets, and `AddFilePropertyCmd`: expose the underlying operation/data query,
  not a command to open a user interface.

### Account and installation administration

Authentication/password/passkey/recovery commands, WebDAV credential management, passphrase
changes, initialization/unlock, maintenance mode, upload-limit settings, user creation/deletion,
Space membership, and Space creation/deletion are not needed for document assistance.

[`spaces` commands][space-actions] and tenant/user management could form a separate future
administrative tool set with explicit tenant-level authority. Space-bound credentials cannot
create arbitrary Spaces, enumerate unrelated Spaces, or acquire wider permissions. A future
`list_spaces` query can reuse existing permitted-Space discovery only after multi-Space grants
are designed.

## Example useful journeys

1. **Upload and file an invoice:** `get_space -> upload_file -> get_file`, inspect available OCR,
   discover Tags/properties/types, apply classification, then complete filing and `search_files`.
2. **Process existing Inbox arrivals:** `list_inbox -> read_file_text`, classify, then
   `mark_inbox_file_done` or `file_inbox_document`, creating a destination if needed.
3. **Handle a possible replacement:** `find_duplicates -> list_file_versions -> get_file`, then
   use the existing acknowledged merge operation only when replacement is intended.
4. **Recover a document:** `list_trash -> restore_file -> get_file`.

The first two journeys are first-release scope. The remaining journeys use **Later** contracts.
These are proposed workflows, not executed verification.

[space-context]: ../../ctxx/space_context.go
[space-cards]: ../../action/spaces/space_cards_partial.go
[browse-list]: ../../action/browse/list_dir_partial.go
[browse-query]: ../../action/browse/list_dir_file_query_service.go
[inbox-list]: ../../action/inbox/files_list_partial.go
[file-info]: ../../action/browse/file_info_partial.go
[file-properties]: ../../action/browse/file_properties_partial.go
[file-repo]: ../../common/file_repository.go
[ocr]: ../../action/browse/ocr_content_dialog.go
[notes]: ../../model/tenant/file/document_notes.go
[notes-partial]: ../../action/browse/document_notes_partial.go
[note-cmd]: ../../action/browse/document_note_cmd.go
[rename]: ../../action/browse/rename_file_cmd.go
[manage-tags]: ../../action/managetags/actions.go
[tags]: ../../model/tenant/tagging/tag_service.go
[properties]: ../../action/property/actions.go
[document-types]: ../../action/documenttype/actions.go
[versions]: ../../action/browse/file_versions_partial.go
[duplicates-partial]: ../../action/browse/duplicate_matches_partial.go
[duplicates]: ../../model/tenant/file/duplicate_detection_service.go
[trash-list]: ../../action/trash/trash_list_partial.go
[assign-tag]: ../../action/tagging/assign_tag_cmd.go
[unassign-tag]: ../../action/tagging/unassign_tag_cmd.go
[set-property]: ../../action/browse/set_file_property_cmd.go
[add-property-value]: ../../action/browse/add_file_property_value_cmd.go
[remove-property]: ../../action/browse/remove_file_property_cmd.go
[select-document-type]: ../../action/browse/select_document_type_cmd.go
[mark-done]: ../../action/inbox/mark_as_done_cmd.go
[inbox-move]: ../../action/inbox/move_file_cmd.go
[inbox-assign]: ../../action/inbox/assign_file_cmd.go
[move]: ../../action/browse/move_file_cmd.go
[make-dir]: ../../action/browse/make_dir_cmd.go
[trash]: ../../action/browse/delete_file_cmd.go
[restore]: ../../action/trash/restore_file_cmd.go
[merge-cmd]: ../../action/browse/file_version_from_inbox_cmd.go
[merge]: ../../model/tenant/file/file_version_from_inbox_service.go
[retry-preview]: ../../action/browse/retry_pdf_preview_cmd.go
[unzip]: ../../action/browse/unzip_archive_cmd.go
[url-import]: ../../action/openfile/upload_from_url_cmd.go
[openfile]: ../../action/openfile/actions.go
[upload]: ../../action/browse/upload_file_cmd.go
[inbox-upload]: ../../action/inbox/upload_file_cmd.go
[upload-version]: ../../action/browse/upload_file_version_cmd.go
[shared-upload]: ../../action/openfile/upload_files_cmd.go
[download]: ../../action/download/download.go
[tag-actions]: ../../action/tagging/actions.go
[create-assign-tag]: ../../action/tagging/create_and_assign_tag_cmd.go
[move-tag-group]: ../../action/tagging/move_tag_to_group_cmd.go
[import-types]: ../../action/documenttype/import_from_library_cmd.go
[space-actions]: ../../action/spaces/actions.go
[upload-contract]: ../features/2026.09-mcp_server/spec.md#upload-contract
