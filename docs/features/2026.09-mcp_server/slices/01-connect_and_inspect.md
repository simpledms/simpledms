# 01 — Connect a scoped client and inspect its Inbox

Status: implementation complete; verification pending  
Depends on: none  
Contract: [connect and inspect](../spec.md#connect-and-inspect); rules M1–M3, M6

## Observable result

A user creates a credential in Account settings, connects an MCP client, reads an existing Inbox
document, and revokes access. The four inspection tools are a useful standalone result while
subsequent tools are not yet registered.

## Acceptance criteria

- The desktop/mobile credential UI supports create/list/revoke, one-time copyable token display,
  read-only default, empty/no-Spaces/error states, and no secret on refresh. Setup Sessions and
  attempts to manage another account's credentials fail.
- Real Streamable HTTP exposes `get_space`, `list_inbox`, `get_file`, and `read_file_text` using
  current access. Document summaries include their canonical browser URL. An empty Inbox and
  unavailable OCR are ordinary bounded data results.
- Unknown/foreign/deleted objects, invalid/revoked credentials, permission loss, invalid Origin,
  and unavailable tenants produce the specified errors without redirects or data leakage.
- Interleaved calls from two actors/Spaces remain isolated; no request identity enters shared
  SDK fields. A revoked token fails even through an existing legacy MCP connection.
- Browser Inbox reads still use the same extracted query behaviour. Pagination, source filtering,
  and OCR continuation have deterministic boundaries; metadata projections contain no numeric IDs.

## Implementation checklist

- [x] Add the owned MCP credential model/schema, privacy, creation and revocation, generated main
  migration, and error-returning access resolution. Scope and mode cannot be edited in place.
- [x] Reuse local credential UI patterns from `action/dashboard/*webdav_credential*`: existing
  `Form`, `Dialog`, list, copy controls, and snackbar behaviour. Wire Account navigation and
  register the new browser actions through the existing router; translate new strings.
- [x] Mount the pinned official SDK at `/mcp`, with bearer/Origin/proxy/HTTPS checks and bounded
  request decoding. Register only these four typed tools and centralize safe error projection.
- [x] Extract the minimal read execution/context path consumed by both adapters. Keep browser
  authentication/redirects in the router and response delivery after commit for new credential
  actions. Preserve untouched wrapper/manual-flow semantics.
- [x] Extract bounded Inbox/basic file/OCR queries; resolve public IDs and project data inside
  transactions. Keep the UI's source/search behaviour and explicit OCR availability.
- [x] Add tests with the existing SQLite action harness and an SDK HTTP client, plus the
  desktop/mobile `@connect` journey in `e2e/mcp_workflows.spec.ts`.

Implementation notes and generation commands are in the [single execution record][record].
The destination picker uses a native `SelectField`: the existing list-radio input is hidden and
cannot provide keyboard selection. A focused regression test now protects the one-time token
response against `HX-Reswap: none`. The remaining checks below, including browser visibility
and actual client connection, remain pending.

## Verification checklist

- [ ] Run proposed `TestMCPConnection*` cases with
  `go test ./server -run '^TestMCPConnection'`; exercise actual HTTP discovery/calls as well as
  malformed/unknown tools, schema errors, safe internal errors, cancelled reads, and M1/M2/M6.
- [ ] Exercise fresh schema and upgrade of a populated main DB, revoke across restart, and
  failed credential commit with no token-success response. Reuse existing DB seams.
- [ ] Run affected existing tenant/Space permission and Inbox query tests, and test any modified
  router wrapper's error/partial handling. Do not run unrelated slow storage suites.
- [ ] Run `npm run test:e2e -- e2e/mcp_workflows.spec.ts --grep @connect` on desktop and mobile
  contexts: create through visible controls, call MCP, compare the browser Inbox, revoke, retry.
  Include copy/keyboard navigation, validation, empty state, and narrow-screen dialogs.
- [ ] Manually connect a bearer-capable MCP client using a newly copied token, inspect a real
  document, and revoke it from a desktop and a mobile-sized browser. Record the client/version.
- [ ] Complete the [plan's build/format/migration checks][common-checks]
  and update the single execution record with actual commands, results, and limitations.

Relevant precedents: [`router.go`][router], [`WebDAV credential actions`][credentials],
[`credential tests`][credential-tests], [`Inbox queries`][inbox], [`browser helpers`][helpers].

[router]: ../../../../server/router.go
[credentials]: ../../../../action/dashboard/create_webdav_credential_cmd.go
[credential-tests]: ../../../../server/webdav_credential_action_test.go
[inbox]: ../../../../action/inbox/files_list_partial.go
[helpers]: ../../../../e2e/helpers.ts
[common-checks]: ../plan.md#verification-approach-for-implementation
[record]: ../../../specs/20260918_mcp_server.md#source-review-and-execution-record
