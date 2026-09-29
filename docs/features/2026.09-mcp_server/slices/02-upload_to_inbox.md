# 02 — Upload a document and open it in Inbox

Status: implemented; automated review passed, remaining verification pending
Depends on: [01](01-connect_and_inspect.md)  
Contract: [upload](../spec.md#upload-contract); rules M1–M3 and the linked storage rules

## Observable result

A read/write MCP client uploads one document and receives its durable ID. The same document
appears in browser Inbox with source MCP and can be opened/downloaded there.

## Acceptance criteria

- `upload_file` completes the specified bounded native upload without a separate transfer client
  or browser redirect. Its returned ID works with slice 01 reads after persistence succeeds.
- Invalid/oversized encoded input, decoded overflow, empty bytes, invalid filename, collision,
  read-only credentials, quota failure, and storage failure leave no successful partial document.
- Credential revocation or access loss before finalization prevents completion. Ambiguous commits
  retain recoverable canonical bytes according to existing storage rules, not blanket cleanup.
- Ordinary browser Browse/Inbox uploads still work; MIME/OCR work may be pending after success.
  Existing file-source values remain unchanged and the MCP source is displayed/filterable.

## Implementation checklist

- [x] Bind `filename`/`content_base64` with HTTP, encoded-field, and decoded-size bounds from the
  spec. Keep the decoded content as a reader, validate the basename, and do not log arguments.
- [x] Extract the duplicated Browse/Inbox prepare/upload/finalize coordination into the owning
  filesystem model, accepting a reader and trusted source. Make browser handlers and MCP use it
  while preserving multipart binding and current UI responses.
- [x] Reuse quota, expected-byte counting, hash/checksum verification, cleanup, and scheduler
  reconciliation. Implement no new storage format or long request-wide write transaction.
- [x] Integrate MCP credential revalidation with the fresh authorization/finalization transaction
  using the established main-before-tenant locking order. The generic `txx` helper currently
  rechecks membership, not an MCP credential; explicitly cover that added check and its race.
- [x] Append/regenerate the `MCP` source enum and update existing source label/filter consumers
  and translations. Return committed file data even when subsequent MIME enrichment is pending.
- [x] Add bounded-tool/S3 tests in `server/mcp_upload_test.go` and the `@upload` desktop/mobile
  browser journey; use existing encryption-on/off harness support.

Implementation details and the intentionally unperformed verification boundary are recorded in
the [single execution record][record]. `MCP` is displayed as an untranslated protocol name, so no
new translatable source string was introduced.

## Verification checklist

- [ ] Run `go test ./server -run '^TestMCPUpload'` once implemented, covering malformed base64,
  exact/over limits, missing/chunked Content-Length, configured lower/unlimited limits, collision,
  empty input, read-only token, quota, and current-source persistence.
- [ ] Check original-byte round trips with encryption on/off, storage verification failure,
  cancellation during ingestion, revoke/access-loss before finalize, and uncertain-result
  recovery using the existing S3 test facilities. Include only directly relevant network cases.
- [x] Rerun the affected existing upload-size, finalization-authorization, and concurrent upload
  tests. If extraction touches browser version upload, include its existing regression too.
- [x] Run `npm run test:e2e -- e2e/mcp_workflows.spec.ts --grep '@connect|@upload'`: create a token
  in the UI, send fixture bytes through MCP, open/download and compare them through the browser,
  filter source MCP, and exercise one ordinary browser upload on desktop and mobile.
- [ ] Manually upload a representative document from a bearer-capable MCP client, inspect source
  and readable bytes in Inbox, then try a rejected oversized file and a read-only credential.
- [ ] Complete the [plan's common checks][common-checks]
  and record actual evidence. Do not claim immediate OCR readiness as upload verification.

Relevant precedents: [`Inbox upload`][upload], [`Browse upload`][browse-upload],
[`txx`][txx], [`authorization regression`][authorization], [`size tests`][limits],
[`source enum`][source]. Storage rule ownership is in [invariants.md](../invariants.md).

[upload]: ../../../../action/inbox/upload_file_cmd.go
[browse-upload]: ../../../../action/browse/upload_file_cmd.go
[txx]: ../../../../util/txx/main_tenant_tx_helpers.go
[authorization]: ../../../../server/upload_authorization_test.go
[limits]: ../../../../server/upload_size_limit_test.go
[source]: ../../../../model/main/common/filesource/file_source.go
[common-checks]: ../plan.md#verification-approach-for-implementation
[record]: ../../../specs/20260918_mcp_server.md#source-review-and-execution-record
