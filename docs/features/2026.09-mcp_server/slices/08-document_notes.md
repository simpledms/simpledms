# 08 — Manage document notes and history

Status: implemented and verified
Contract: [notes and history](../spec.md#notes-and-history-contract); M1–M3 and existing note rules

## Observable result

A client reads bounded previews/bodies and creates, edits, replaces, or deletes notes through the
existing authorship/ownership and retained-history model. Read-only credentials remain read-only.

## Implementation

- [x] Share the authorized/history-aware query with paginated MCP reads.
- [x] Project public references and preserve legacy unknown authorship and nullable timestamps.
- [x] Reuse note model mutations, history conflicts, owner/author checks, and Trash restrictions.
- [x] Bound input and body windows, and add SDK/HTTP lifecycle and permission coverage.

## Verification

- [x] Run `go test ./server -run '^TestMCPNotes|^TestDocumentNotesModel'`, including readonly,
  owner/author, legacy, pagination, history, Trash, pairing, and Space-isolation behavior.
- [x] Change notes through both transports and compare browser desktop/mobile current/history
  views, including replacement and deletion. Exercise interrupted/concurrent mutations.
- [x] Complete scoped build checks and update the [single execution record][record].

[record]: ../../../specs/20260918_mcp_server.md#source-review-and-execution-record
