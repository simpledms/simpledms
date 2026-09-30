# 05 — Manage metadata definitions

Date: 2026-09-29
Status: implementation complete; focused checkpoints passed, verification pending
Depends on: [03](03-classify_document.md)
Contract: [metadata management](../spec.md#metadata-management-contract); rules M1–M4

## Observable result

A client creates, edits, and deletes the same Tag, field, and document-type configuration as the
UI. It can organize Tag groups/compositions, configure document-type attributes, and import
advertised library templates, all inside its credential's Space.

## Implementation checklist

- [x] Expose Tag, field, and document-type CRUD through existing model methods with public IDs.
- [x] Add Tag grouping, composition, and shared atomic create-and-assign behavior.
- [x] Add typed attribute creation/editing/removal using existing unique definition references.
- [x] Expose library discovery/import with the UI's empty-metadata rule and key validation.
- [x] Retain field type immutability, public-ID stability, readiness, constraint handling, and
  current credential authority. Do not grant wider Space or definition permissions.
- [x] Add real SDK/HTTP CRUD, relationship, read-only, and same-tenant cross-Space coverage.
- [x] Update discovery expectations and the tool catalog.

## Verification checklist

- [ ] Run `go test ./server -run '^TestMCPMetadataManagement|^TestMCPBearerCredentialCannotCrossSpaces'`
  for the completed tool set, including duplicates/in-use deletion, stable IDs, grouping,
  composition, attributes, atomic operations, template import, and unchanged state on rejection.
- [ ] Run relevant existing Tag, field, document-type, and library HTTP/model regressions,
  including the shared create-and-assign browser operation.
- [ ] Exercise management with a bearer-capable client and observe create/edit/delete effects
  through desktop/mobile browser management screens. Check the existing discovery journey after
  the new registrations.
- [ ] Complete applicable build/format checks and record actual verification before marking
  the plan delivery checkbox complete.

The [single execution record][record] distinguishes the focused implementation checkpoints from
the unperformed complete verification. No schema migration is required; attribute references use
the [existing unique pairs][attribute-adr].

[record]: ../../../specs/20260918_mcp_server.md#source-review-and-execution-record
[attribute-adr]: ../../../adrs/2026.09-mcp_metadata_attribute_references.md
