# Inbox transfers execution record

Date: 2026-10-01

The original implementation and scoped verification were completed on 2026-10-01.
Tenant owners enable receiving from nonmembers through Edit space → Accepts inbox transfers. Users select Move to another Inbox from an Inbox file
menu, choose a destination, and optionally add a message.

## 2026-10-02 destination eligibility update

Destination selection and submission now share a query that permits existing write access or
receiver opt-in. The success snackbar offers an “Open file” action only when ordinary Space
privacy permits access to the destination; it links directly to the file in that Inbox.
The dialog explains which Spaces are available, uses lowercase “tags” in English,
and has updated German, French, and Italian translations. Regression tests were updated for
writable destinations and revoked destination membership. Tests and browser checks were not rerun
for this update; the results below describe the earlier implementation. `gofmt -w` completed, and
`go generate` in `i18n` succeeded with no new-feature missing translations for de/fr/it. Existing
unrelated translation warnings remain; log: `/tmp/simpledms-inbox-transfer-access-translations.log`.

## Execution and verification

| Command or action | Result |
| --- | --- |
| `go generate .` | Completed; regenerated Ent APIs. |
| `go run ./cmd/migrate inbox_transfers` | Completed after normalizing the boolean default. Tenant migration only adds a column; no table rebuild or row copy. |
| Atlas checksum regeneration through the pinned Go library | Completed after discarding unused generated migrations. Unrelated main-schema WebDAV index drift excluded. |
| `gofmt -w` on changed Go source and tests | Completed. |
| `go generate` in `i18n` | Completed; new de/fr/it strings translated and marked fuzzy. Existing unrelated duplicate/missing warnings remain. Log: `/tmp/simpledms-inbox-translations.log`. |
| Initial transfer tests | Seven tests passed, 8.585s. |
| Expanded scoped server tests (command below) | 40 top-level tests passed, zero skips, 24.893s. Includes transfers, HTTP security, historical migrations, notes, versions, and Space permissions. |
| Scoped `go vet` (command below) | Passed, no output. |
| Current application startup with production migrations | Passed on a disposable database and separate development S3 bucket. |
| Desktop browser, 1440×1000 | Real file transfer with message passed; source emptied, recipient received file and escaped message note. Final success notification visible, no HTMX target errors. |
| Mobile browser, 390×844 | Real file transfer with blank message passed; recipient received file without an empty note. Final success notification visible, no HTMX target errors. |
| Required destination validation | Submission blocked while empty; dialog and entered message preserved. |

Commands:

```sh
go test -v -tags 'sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_icu' ./server -run '^(TestInboxTransfer|TestEditSpaceCmd|TestDocumentNotes|TestDocumentNoteTitlesProductionMigration|TestFileVersionFromInboxCmd|TestSpaceUser|TestSpaceCreatePermissions|TestSpaceDeletePermissions|TestRouterRejectsDeletedSpace)' -count=1

go vet -tags 'sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_icu' ./action/inbox ./action/spaces ./model/tenant/file ./model/tenant/space ./db/entx ./server ./cmd/migrate
```

Logs: `/tmp/simpledms-inbox-transfer-final-tests.txt` and
`/tmp/simpledms-inbox-transfer-vet.txt`. Final browser screenshots:
`/tmp/simpledms-inbox-transfer-desktop-success.png` and
`/tmp/simpledms-inbox-transfer-mobile-success.png`.

Verification found and corrected:

- Transfer success notification could not mount because the dialog was appended to `body`.
  The menu now uses the existing dialog-loading helper and its `#popovers` container.
  Desktop and mobile journeys were repeated on the rebuilt application with no target errors.
- Historical document-note migration fixtures used current Ent Space builders against older
  schemas. They now seed only historical columns, without changing production migrations.
- The new tenant-membership revocation test initially expected a forbidden response. It now
  checks the existing authentication redirect when the user's last membership expires,
  together with unchanged persisted document/notes and absent transfer-success headers.

The Go suite and vet passed before the final menu-only dialog mount correction; the rebuilt
application and desktop/mobile checks verified that final correction. The full repository
suite and unrelated slow network tests were not run. Initial browser upload setup failed
because the disposable instance had no S3 bucket configured; configuration was corrected
before successful real-file journeys. The disposable server and browser session were stopped.
No deployed database was migrated.

## Security review

No confirmed security vulnerability was found in the reviewed transfer paths. Review covered
source authorization, destination discovery and opt-in, tenant isolation, transaction handling,
file/preview/note access after transfer, WebDAV aliases, search scoping, message rendering,
and the existing production CSRF/session middleware.

Executable security coverage is in
[server/inbox_transfer_test.go](../../../server/inbox_transfer_test.go) and
[server/inbox_transfer_security_test.go](../../../server/inbox_transfer_security_test.go).
It includes forged cross-tenant references, unavailable destinations, owner-only opt-in,
revoked permissions/opt-in between dialog and submit, denied source/destination download and
preview URLs after transfer, denied note reads, escaped messages, and rollback behavior.

The HTTP denial tests use metadata fixtures and require explicit authorization/not-found
responses; they do not validate successful byte streaming. Real uploads and receiver visibility
were checked in the browser. CSRF middleware was reviewed in code, not independently
penetration-tested. Ordinary destination notes retain their existing editing semantics.

Design decisions are recorded in the [ADR](../../adrs/2026.10-inbox_transfers.md).
