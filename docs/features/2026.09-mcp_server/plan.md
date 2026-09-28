# MCP implementation plan

Date: 2026-09-18  
Status: slice 01 implementation complete; verification and all delivery checkboxes pending

Contract: [specification](spec.md), [durable rules](invariants.md), and the linked
[architecture](../../specs/20260918_mcp_server.md). The source review and all future execution
evidence belong in the [single record][record].

Slice 01 implementation includes an opt-in commit-buffered browser response and a native
`SelectField` for keyboard-accessible Space selection. The remaining UI composition reuses the
existing credential widgets. See the record for migration generation's unrelated-schema-drift
exclusion and the exact unperformed verification boundary.

## Ordered slices

1. [ ] [Connect a scoped client and inspect its Inbox](slices/01-connect_and_inspect.md)
2. [ ] [Upload a document and open it in Inbox](slices/02-upload_to_inbox.md)
3. [ ] [Classify a document with existing metadata](slices/03-classify_document.md)
4. [ ] [File a classified document and find it again](slices/04-file_document.md)

## Why four slices

The complete feature exceeds one implementation session. Connection/credential lifecycle,
storage ingestion, metadata mutation, and filing are independently usable journeys and have
different regression boundaries. Slice 1 is already user-observable, not a foundation-only slice.
The upload checkpoint isolates storage/finalization risk before expanding metadata writes.

Before defining these documents, adjacent discovery/mutation work was merged: Tags, properties,
and document types share one classification slice and one public-ID migration effort. Folder
discovery/creation, completion, and filed search likewise share one filing slice. Validation,
error handling, UI compatibility, migration tests, and browser checks travel with each behaviour.

Each slice is scoped to one session using existing operations. In particular, slice 1 has only
owner-managed create/list/revoke credentials and the four inspection tools; slice 2 has one
bounded Inbox upload; slice 3 applies existing metadata without definition editors or attribute
IDs; slice 4 uses existing filing and search without a general file-management API.

Slice 1 has no feature dependency. Slice 2 depends on 1; slice 3 on 1–2; slice 4 on 1–3. Each
leaves its delivered journeys working and registers only completed tools. All four are needed
for the first-release outcome. There is no later foundation, integration, or testing slice.

## Shared implementation guidance

- Keep `Router.RegisterAction` and its wrappers for browser entry points. Extract execution
  facilities only when consumed, preserving legacy forms, partial dispatch, and manual storage
  transaction handling. No repository-wide router or error-model rewrite is required.
- Reuse the credential `Form`, `Dialog`, `List`, copy controls, and snackbar patterns inspected in
  [`dashboard`][credential-ui]. Keep the new composition local. Reuse existing classification
  partials and filing dialogs; inspect rendered controls before adding Playwright selectors.
- New UI strings follow the repository's four-language translation conventions. Change source
  catalogs and generate through the existing workflow; never edit generated catalogs directly.
- Public IDs, the new credential table, and the MCP source arrive in their consuming slices.
  Follow [generated/additive migration rules][repo-rules]. Backfill is real runtime work, not a
  test-only seed or a presumed effect of Ent's `DefaultFunc`.
- Keep the [recorded assumptions](spec.md#recorded-assumptions). Do not expand later tool families
  merely because their handlers are easy to expose.

## Verification approach for implementation

The following are planned checks, not commands run during planning. Each slice provides its own
scoped checklist. Use public behaviour and existing runtime seams, without production test hooks.

- Put SQLite-backed MCP/HTTP integration tests in `server/mcp_*_test.go` so they can reuse
  [`actionTestHarness`][harness]. Use an SDK client for real `tools/list`/`tools/call` behaviour,
  including HTTP authentication; use in-memory SDK transport for isolated protocol/schema checks.
- Apply relevant existing action regressions when extracting shared logic. Test transaction
  rollback through the transaction-owning boundary, not only by invoking a handler directly.
- Use a disposable S3-enabled instance for ingestion. The harness supplies encryption-on/off
  coverage. Run targeted integrity/cancellation checks relevant to ingestion; do not pull in
  unrelated slow bad-network suites.
- Add `e2e/mcp_workflows.spec.ts` incrementally, using [`helpers.ts`][browser-helpers] and existing
  authentication/fixture setup. Define desktop and touch/mobile (~390px) contexts inside this
  suite; the current global config otherwise runs Desktop Chrome only.
- Browser checks issue MCP HTTP requests through a test-local client helper, then observe actual
  visible UI. Do not bypass credentials or persistence with a browser-test-only API. Browser
  credential creation/revocation and browser-side changes must traverse visible controls.
- In every slice, run its focused Go checks, its tagged desktop/mobile Playwright journey, and
  `go build ./...`; format touched Go files and check the diff. For schema work, also check fresh
  installation and populated upgrade paths, including repeated/restarted backfill.
- Keep an up-to-date instance running for browser checks using [the existing setup][browser-setup].
  Rebuild/restart only as code or embedded assets require. Record missing prerequisites and
  unperformed manual steps explicitly. Do not mark a slice complete based only on written tests.

## Scope-to-slice coverage

| Specification outcome | Owning slice | Additional regression |
| --- | --- | --- |
| 1: create/connect/inspect/revoke | 01 | Credentials reused by every subsequent journey |
| 2: upload readable bytes | 02 | Existing browser ingestion and source filtering |
| 3: apply type, Tags, and values | 03 | Inbox/Browse metadata and upgrade compatibility |
| 4: finish and rediscover | 04 | Last-Inbox-file UI and root/non-folder filing |
| 5: cross-transport correctness | Each owning slice | Relevant earlier journeys retained |

[record]: ../../specs/20260918_mcp_server.md#source-review-and-execution-record
[credential-ui]: ../../../action/dashboard/create_webdav_credential_cmd.go
[repo-rules]: ../../../AGENTS.md#additive-sqlite-migrations
[harness]: ../../../server/action_integration_test.go
[browser-helpers]: ../../../e2e/helpers.ts
[browser-setup]: ../../../e2e/README.md
