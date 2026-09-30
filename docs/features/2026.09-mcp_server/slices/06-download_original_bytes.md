# 06 — Download original bytes

Status: implemented; focused checkpoint passed, verification pending
Contract: [original-byte download](../spec.md#original-byte-download-contract); M1–M3

## Observable result

A read-capable client retrieves canonical bytes with its MCP bearer credential, including existing
versions, without requiring a browser Session or receiving internal storage identifiers.

## Implementation

- [x] Read bounded plaintext ranges through the existing encrypted/compressed storage reader.
- [x] Pin continuation to the returned version and bound allocations, byte offsets, and EOF.
- [x] Expose typed metadata/base64 output and protect read-only and cross-Space behavior.
- [x] Add SDK/HTTP encryption-on/off round-trip coverage.

## Verification

- [ ] Run `go test ./server -run '^TestMCPDownload'`, including invalid versions/ranges, canonical
  bytes, read-only access, same-tenant Space isolation, and storage/cancellation errors.
- [ ] Exercise continuation across a concurrent new-version upload and compare the completed
  bytes with a browser download. Check a representative large file and historical version.
- [ ] Complete scoped build checks and update the [single execution record][record].

[record]: ../../../specs/20260918_mcp_server.md#source-review-and-execution-record
