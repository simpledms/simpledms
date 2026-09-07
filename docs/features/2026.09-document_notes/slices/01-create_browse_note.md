# 01: Create A Browse Note

Status: Implementation complete. [Delivered verification and caveats](../verification.md).
Unchecked verification items below are remaining optional/manual checks where the full
original wording is not evidenced; they do not indicate missing implementation.

Depends on: none. [Plan and verification contract](../plan.md).

Journey: Open an existing Browse document, add a multiline note, and see the
saved text with your name after reopening the document.

## Acceptance Criteria

- Browse exposes Notes in both Space modes, with an empty state and Add note.
- Two submitted notes persist, retain line breaks, and appear newest first with
  author names and creation times. Equal timestamps have stable ordering.
- Empty/whitespace submissions fail; markup displays as text; cancel changes nothing.
- Reads and creation reject inaccessible documents, directories, and invalid
  references; creation also rejects trashed documents and spoofed authorship.
- Existing upload, move, and version-merge journeys still work with note rows
  present. No new note foreign key blocks or loses data on source deletion.
- No edit, delete, replace, or inactive history controls are stubbed in this slice.

## Implementation Checklist

- [x] Introduce the smallest tenant document-note schema/model needed for text,
  public identity, document membership, nullable author, and creation attribution.
  Use Ent schema sources and privacy conventions; do not attach notes to versions.
- [x] Add scoped query/create behaviour and register the create command, dialog,
  and shared list partial through existing action ownership and `wrapTx`. Keep
  business validation and persistence in the `DocumentNotes` domain owner (no new repository).
- [x] Compose Browse Notes from existing widgets. Preserve full multiline text
  without ListItem truncation; add only the multiline form support actually needed.
- [x] Return success/error feedback using existing command/event patterns; refresh
  only the notes partial and retain the selected document/tab. Translate new text.
- [x] Include the minimal lifecycle foundation now: transfer any source note rows
  inside `FileVersionFromInboxService.MergeFromInbox` before its hard delete.
  Preserve note identities and attribution, and leave target notes unchanged.
- [x] Generate Ent and the new-table migration using the repository workflow.
  `File.notes` remains for lazy legacy compatibility; see slice 02.

## Verification Checklist

- [x] SQLite-backed model/HTTP tests cover persisted creation, validation, ordering,
  escaping, actor attribution, scope/permission failures, directories, and Trash writes.
  Focused tests and the broader suite passed; exact commands are in the record.

- [x] Cover source/target note rows and merge preservation in the new
  `server/document_notes_model_test.go`, with root filing in a new HTTP test file.
- [x] Add a dedicated ordinary new-version upload test with notes present, with
  encryption enabled and disabled, in `server/document_notes_versions_test.go`.
- [x] Add desktop and mobile contexts to `e2e/document_notes.spec.ts`: upload via
  existing helpers, open Details/Notes, create two notes, reload, and assert text,
  order, named author, empty validation, cancellation, and absence of overflow.
- [x] Run focused `go test ./server -run 'TestDocumentNotes|TestFileVersionFromInboxCmd'`,
  tests in the new model package, the notes Playwright suite, and the plan's build,
  formatting, migration, and translation gates.

## Manual Verification

Remaining optional independent walkthrough; desktop/mobile screenshot readability and
scrolling were inspected, and the automated suite covers most steps. Non-folder fixtures
are Go coverage. Keyboard dialog focus/return was not independently audited.

1. On a disposable instance at desktop width, upload a document, open its Details,
   select Notes, and add two distinct multiline notes. Confirm your displayed name.
2. Try whitespace-only text, cancel another draft, and enter literal HTML. Confirm
   useful errors, no extra note, and no interpreted markup.
3. Reload, switch tabs and return, then reopen the document. Confirm saved notes
   and order. Repeat in a non-folder Space.
4. At approximately 390px width, open the details sheet and complete the same
   journey with touch and keyboard. Check reachable actions, readable long text,
   focus on dialog open/close, and no horizontal page overflow.
