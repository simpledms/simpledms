# 03 — Classify a document with existing metadata

Status: implemented and verified
Depends on: [01](01-connect_and_inspect.md), [02](02-upload_to_inbox.md)  
Contract: [classification](../spec.md#classification-contract); rules M1–M4

## Observable result

The client discovers a Space's existing metadata, assigns a document type and Tags, and sets or
removes typed fields. Browser Inbox/Browse metadata and `get_file` show the same result.

## Acceptance criteria

- All ten classification tools in the specification work for both newly created and upgraded
  metadata. Public IDs remain stable after restart or definition rename; no old associations,
  type attributes, composed Tags, or stored field values are lost in migration.
- A repeat assign/set does not deselect a type or duplicate direct Tags. Removing a direct Tag
  does not falsely report removal of a separately resolved Tag. Group assignment is rejected.
- Property cases cover every type, explicit zero/false/empty text, omission/null, invalid dates,
  multiple/wrong value fields, integer overflow, and money minor units without float drift.
- Foreign IDs, a deleted file/definition, invalid targets, and read-only credentials fail without
  partial writes. One failed tool does not roll back earlier successful classification tools.
- Existing browser type toggles, Tag interactions, add-value validation, cleared dates, dialogs,
  and partial refreshes remain functional in Inbox and Browse on desktop/mobile.

## Implementation checklist

- [x] Add public-ID generation and uniqueness for Tag, Property, and DocumentType with generated
  additive migrations. Stage field/index work per repository rules; no attribute-ID migration.
- [x] Implement bounded, restart-safe backfill through the existing tenant data-migration path.
  Account for populated tables, creation during upgrade, uniqueness conflicts, and completion
  before exposing metadata operations for that tenant. A creation default is not a backfill.
  Protect assigned public IDs from subsequent changes without blocking initialize-only backfill.
  Keep the rollout columns nullable for additive SQLite upgrades, reject empty IDs at every MCP
  projection boundary, and replace the rollout mixin with the required public-ID mixin in a future
  schema-tightening migration that does not rebuild existing tables.
- [x] Extract definition reads into explicit bounded projections, with inline type attributes
  referencing Tag/property public IDs. Extend `get_file` with typed assigned metadata.
- [x] Put desired-state Tag assignment and document-type set/clear behaviour in existing models.
  Resolve file and metadata together under the Space context, then call repository operations.
- [x] Extract property assignment/update/removal from Browse handlers and `functions.go` into
  model behaviour. Normalize browser decimals/presence and MCP typed fields at their adapters.
- [x] Have existing HTMX actions and tools use these model operations. Add write execution and
  commit-before-feedback handling where consumed; preserve browser partial/target contracts.
- [x] Add migration and cross-transport tests plus `@classify` desktop/mobile checks in the shared
  MCP workflow suite. Use browser metadata setup/import rather than MCP definition-management tools.

## Verification checklist

- [x] Run proposed `go test ./server -run '^TestMCPClassification'`, covering the acceptance cases,
  concurrent repeated assignments, correct resolved-Tag results, and rollback on operation or
  commit error through the real execution boundary.
- [x] Verify fresh schemas and a populated pre-feature tenant through generated migrations and
  resumed/repeated backfill. Include existing grouped/composed Tags, properties, document types,
  attributes, and file assignments; inspect generated SQL for unexpected existing-table rebuilds.
- [x] Run affected metadata model/action regressions and confirm query/schema handling never
  serializes Ent internal IDs. Include schema validation and tool business errors separately.
- [x] Run `npm run test:e2e -- e2e/mcp_workflows.spec.ts --grep @classify` in both contexts:
  upload through the earlier tool, classify through MCP, refresh browser metadata, change values
  through actual browser controls, then read them back through MCP. Include a false Checkbox,
  zero, a cleared date, unchanged type selection, and a repeated Tag assignment.
- [x] Exercise affected `e2e/browse_upload_filters.spec.ts` behaviour; extend the new suite's
  mobile coverage for its metadata consumers rather than changing every global browser project.
- [x] Manually classify one Inbox document using known definitions, inspect it in desktop/mobile
  UI, and correct metadata there. Confirm subsequent MCP reads reflect the correction.
- [x] Complete the [plan's common checks][common-checks]
  and add actual commands/results to the single record before marking complete.

Relevant code: [`TagService`][tags], [`property value conversion`][values],
[`document-type selection`][type-selection], [`PublicIDMixin`][public-id],
[`data-migration runner`][backfill], [`metadata browser tests`][browser-tests].

[tags]: ../../../../model/tenant/tagging/tag_service.go
[values]: ../../../../action/browse/functions.go
[type-selection]: ../../../../action/browse/select_document_type_cmd.go
[public-id]: ../../../../db/entx/public_id_mixin.go
[backfill]: ../../../../model/tenant/tenantdatamigration/runner.go
[browser-tests]: ../../../../e2e/browse_upload_filters.spec.ts
[common-checks]: ../plan.md#verification-approach-for-implementation
