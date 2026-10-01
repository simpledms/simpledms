# 09 — Organize filed documents and directories

Status: implemented and verified
Contract: [filed organization](../spec.md#filed-organization-contract); M1–M5

## Observable result

A writable client renames/moves filed entries without changing identity, classification, source,
versions, notes, or child-parent references. Inbox completion remains the existing lifecycle flow.

## Implementation

- [x] Resolve live filed sources/destinations and reuse filesystem name/movement/cycle rules.
- [x] Keep root and Inbox entries outside standalone organization and retain folder-mode rules.
- [x] Keep optional child creation and movement atomic; return ordinary conflict errors.
- [x] Add SDK/HTTP preservation, collision, readonly, and cross-Space coverage.

## Verification

- [x] Run `go test ./server -run '^TestMCPOrganization'` and affected existing Browse move/rename
  and filing regressions, including rollback and directory descendants.
- [x] Exercise renamed/moved entries, navigation, breadcrumbs, and preserved notes/metadata in
  browser desktop/mobile views. Check same-parent child creation and non-folder rename.
- [x] Complete scoped build checks and update the [single execution record][record].

[record]: ../../../specs/20260918_mcp_server.md#source-review-and-execution-record
