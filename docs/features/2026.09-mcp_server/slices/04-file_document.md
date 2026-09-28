# 04 — File a classified document and find it again

Status: planned  
Depends on: [01](01-connect_and_inspect.md)–[03](03-classify_document.md)  
Contract: [filing](../spec.md#filing-contract); rules M1–M5

## Observable result

The client finishes an Inbox document in either Space mode, creates/selects a destination when
needed, and finds the filed document with its classification intact. This closes the complete
first-release upload/classify/file journey.

## Acceptance criteria

- `list_directory`, `create_directory`, `file_inbox_document`, `mark_inbox_file_done`, and
  `search_files` complete the specification's journey; no client must guess a directory ID.
- Root filing succeeds even when root is the document's current parent. Optional child-directory
  creation and filename changes participate in the filing transaction. Non-folder completion
  works without folders or mandatory classification values.
- Invalid/foreign/deleted destinations, non-directory targets, wrong Space mode, filename
  conflicts, already-completed documents, and read-only credentials fail without partial state.
- Filing preserves M5's related data. The document leaves Inbox, appears in filed search, and
  remains accessible with the same public ID and values through both adapters.
- Last-file empty state, next-file selection, dialogs, snackbar/events, and browser navigation
  continue to work on desktop/mobile. Search and filters do not include Inbox or other Spaces.

## Implementation checklist

- [ ] Extract the lifecycle rule shared by Inbox `MoveFileCmd`, `AssignFileCmd`, and
  `MarkAsDoneCmd`. Reuse filesystem movement/creation; enforce the formerly inconsistent Inbox
  precondition centrally and use the ordinary shared write execution boundary.
- [ ] Expose destination reads/creation through scoped public IDs. Preserve filename, mode,
  conflict, and filesystem cycle validation instead of making a second tree implementation.
- [ ] Extract filed-query inputs/projections from `ListDirFileQueryService`, preserving FTS,
  resolved-Tag and document-type filtering, sorting, and deterministic bounded pagination.
  Keep browser state/callbacks in the adapter and initial property-filter expansion out of scope.
- [ ] Register the five tools and preserve existing Inbox/Browse response contracts and selection
  refreshes. Do not treat an MCP mutation as an automatic event in another browser tab.
- [ ] Add `server/mcp_filing_test.go`, extend relevant current filing regressions, and add `@file`
  desktop/mobile checks with uploaded and classified fixtures from earlier slices.

## Verification checklist

- [ ] Run proposed `go test ./server -run '^TestMCPFiling'`, covering both modes, same-parent root
  filing, optional new directory/name, empty metadata, bad destinations, conflicts, repeated and
  concurrent completion, rollback including a newly created folder, and public-ID isolation.
- [ ] Run `go test ./server -run '^TestMarkAsDoneCmd'` and the relevant existing
  `TestDocumentNotesHTTPInboxRootFilingPreservesHistory` and
  `TestDocumentNotesHTTPMissingParentRestoreAndNonFolderFiling` regressions for preserved data/UI.
- [ ] Check extracted search against existing filename/content, resolved-Tag, document-type,
  ordering, no-match, and pagination cases, using the existing SQLite/FTS test configuration.
- [ ] Run `npm run test:e2e -- e2e/mcp_workflows.spec.ts` for the complete desktop/mobile journey.
  File via MCP then refresh the browser; also file via visible browser controls and confirm MCP
  state. Include the final Inbox item, multiple-item selection, folder and non-folder Spaces.
- [ ] Run affected existing Browse/upload/filter consumer checks after the final extraction.
- [ ] Manually upload, classify, choose/create a folder, file, and rediscover a representative
  document. Repeat completion in a non-folder Space and check browser desktop/mobile views.
- [ ] Complete the [plan's common checks][common-checks]
  and update the single execution record. Mark first release complete only when all four slice
  journeys and their required checks have evidence.

Relevant code/tests: [`MoveFileCmd`][move], [`AssignFileCmd`][assign], [`MarkAsDoneCmd`][done],
[`filesystem`][filesystem], [`file queries`][queries], [`last Inbox item`][last-item],
[`filing preservation`][preservation].

[move]: ../../../../action/inbox/move_file_cmd.go
[assign]: ../../../../action/inbox/assign_file_cmd.go
[done]: ../../../../action/inbox/mark_as_done_cmd.go
[filesystem]: ../../../../model/tenant/filesystem/file_system.go
[queries]: ../../../../action/browse/list_dir_file_query_service.go
[last-item]: ../../../../server/mark_as_done_cmd_test.go
[preservation]: ../../../../server/document_notes_filing_test.go
[common-checks]: ../plan.md#verification-approach-for-implementation
