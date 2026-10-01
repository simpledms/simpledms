# 07 — Search by typed field values

Status: implemented and verified
Contract: [typed field search](../spec.md#typed-field-search-contract); M1–M4

## Observable result

A client combines typed field conditions with text, Tag/type selection, ordering, and pagination.
Browse consumes the same predicates with its existing money/date input normalization.

## Implementation

- [x] Resolve property public IDs and bind exactly one correctly typed value per condition.
- [x] Share validated text/numeric/date/Checkbox predicates with Browse.
- [x] Support inclusive ranges, explicit zero/false/empty text, and bounded AND composition.
- [x] Include documents with unrelated fields when the selected Checkbox is unassigned/false.
- [x] Add SDK/HTTP validation, composition, pagination, and cross-Space tests.

## Verification

- [x] Run `go test ./server -run '^TestMCPFieldFilters'` and affected existing Browse property,
  FTS, and filter regressions.
- [x] Exercise desktop/mobile field filter controls against the MCP result, including decimal
  money, open date bounds, checked/unchecked/missing values, and deleted-definition URL state.
- [x] Complete scoped build checks and update the [single execution record][record].

[record]: ../../../specs/20260918_mcp_server.md#source-review-and-execution-record
