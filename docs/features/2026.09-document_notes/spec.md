# Document Notes

Status: Proposed; core behaviour clarified with the user

Date: 2026-09-07

## Outcome

Users can keep multiple authored notes on a document, change current notes, and
inspect notes that were soft deleted or replaced. The same Notes tab is available
wherever the application presents document details.

This document specifies behaviour only. It contains no implementation plan or
architecture decisions. Reuse this feature directory for the feature's lifetime.

## Repository Context

- Browse, Inbox, and Trash have separate document detail tab sets in
  `action/browse/file_tabs_partial.go`, `action/inbox/file_tabs_partial.go`, and
  `action/trash/file_tabs_partial.go`.
- Browse and Trash expose details through side sheets; Inbox has its own document
  details composition. Existing tabs track the selected tab in navigation state.
- `db/enttenant/schema/file.go` already has an optional single-value `notes` field.
  `action/trash/file_metadata_partial.go` displays it when non-empty. This is not
  an authored list with replacement or deletion history.
- `db/enttenant/schema/file_version.go` has a separate optional version `note`.
  Document notes and version annotations are different concepts.
- Adding an Inbox file as a version of another document currently requires an
  explicit warning that source metadata will be lost
  (`action/browse/file_version_from_inbox_cmd.go`).
- Documents belong to a Space within a tenant. Existing access boundaries must
  continue to apply to notes, including historical entries.

## Confirmed Decisions

1. Edit updates the existing note in place. It does not retain the previous text
   as a separate historical entry.
2. Replace creates a new note and retains the original as a read-only replaced
   entry, linked to its replacement.
3. The note author and Space/tenant owners may edit, replace, or delete a note.
   Other users may add their own notes, subject to document access.
4. Notes in Trash are read-only. Users must restore the document before changing
   its notes. The history toggle remains available in Trash.
5. When an Inbox document becomes a version of another document, transfer all
   its notes and history to the target, preserving authorship and note states.

## Requirements

### Coverage And Presentation

1. Add a Notes tab to document details in Browse, Inbox, and Trash, including
   their desktop and mobile presentations and both folder and non-folder modes.
   Entry points that open those document views must expose the same notes.
2. Notes belong to the document, not an individual uploaded file version. A new
   file version must not reset the document's notes.
3. Display notes as a list with the title on the main line and note text on the
   secondary line. Clicking a row opens a read-only dialog with the title, full note,
   creator, creation date, and other metadata. Row tooltips show creation date and
   author; edit attribution and time remain in details. Historical states remain
   identifiable in the list. Do not
   substitute the document's uploader for the note author.
   Replaced rows show only their original title and an italic reference to the
   immediate replacement's title, not the original body. The full retained body
   remains available in details. Details separate the prominent title and readable
   note body from subdued metadata and replacement navigation.
   Deleted rows likewise omit their body and show only the title and italic Deleted
   status; their retained text remains readable in the details dialog.
   Use the standard list-item component. Place Edit, Replace, and Delete in a
   context menu, available by right-click and an Actions button for touch and
   keyboard users, rather than displaying those actions inline.
4. Order entries newest first by creation time, with deterministic ordering for
   equal timestamps. Editing does not move an entry to the top; replacement
   creates a new entry with a new creation time.
5. Show an appropriate empty state when no notes match the current filter.
6. New notes require a title and multiline plain-text note. Create/edit/replace
   dialogs label both Title and Note. Preserve line breaks, render titles and text
   safely, and reject empty or whitespace-only titles or content without changing
   stored notes. Existing titleless notes remain readable with the translated
   display-only title "Note"; reads must not invent stored titles.
7. Notes are shared document information, not private notes. Anyone authorized
   to view the document may view its current and historical notes.
8. Controls must be keyboard accessible and labelled, with state communicated
   without relying only on colour. Application labels and messages must support
   English, German, French, and Italian; user-authored text is not translated.

### Create, Edit, Replace, And Delete

1. A user allowed to change the document may create a note. The authenticated
   user is its author; the author cannot be supplied or impersonated through
   user input.
2. Edit changes the title and text of a current note while retaining its identity,
   original author, and creation time. An owner editing another user's note
   must not become its original author; show who last edited it and when.
3. Replace accepts a title and text and creates a new current note authored by the user
   performing the replacement. The original retains its title, text, and attribution
   and is marked replaced. The relationship must be understandable when history
   is shown, including successive replacements.
4. Replacement succeeds as a whole or leaves the original unchanged. It must
   never hide the original without successfully creating its replacement.
5. Delete soft deletes a current note after confirmation. Its text and
   attribution remain available through the history toggle; it is not physically
   erased by this action.
6. Soft-deleted and replaced entries are read-only. They cannot be edited,
   replaced again, or deleted through this feature. A current replacement may
   itself be edited, replaced, or soft deleted.
7. Deleting a replacement does not reactivate its predecessor.
8. Cancelling an operation makes no changes. A failed operation shows an error
   and must not appear to have succeeded or leave a partial replacement.
9. Successful actions update the list without leaving the document or Notes
   tab, and retain the current history-toggle setting.

### History Toggle

1. Provide a history icon button in the Notes toolbar with a tooltip and accessible
   name indicating that it shows deleted and replaced notes. Use a highlighted,
   pressed state instead of a switch or visible text label. Add note is also an
   icon-only button, with an Add note tooltip and accessible name.
2. By default it is off, and the list contains only current notes.
3. When on, show current, soft-deleted, and replaced entries together. Clearly
   label deleted and replaced states and identify each entry's author.
4. Turning it off hides both historical categories again without changing data.
5. Keep the toggle available even when no historical entries exist, including in
   read-only Trash views. It must have an accessible on/off state.

### Authorization And Lifecycle

1. All reads and changes require access to the note's actual document, Space,
   and tenant. Knowing a note identifier never grants access.
2. Editing, replacing, and deleting additionally require the current note's
   author or an owner of its Space or tenant, and permission to change the
   document. Authorship alone must not preserve access after access is revoked.
3. Other users cannot change someone else's note. No additional global-admin
   exception is introduced by this feature.
4. Trash allows only reading and filtering notes. Direct mutation requests for
   a trashed document must also be rejected.
5. Moving a document normally, including filing it from Inbox, preserves its
   notes, attribution, and history. Trashing and restoring the document do not
   independently delete or reactivate individual notes.
6. Historical content must not silently change as a side effect of changing a
   current replacement. Stale requests targeting an already deleted or replaced
   note must fail without creating conflicting current replacements.
7. Preserve any existing single-value document notes. Surface non-empty legacy
   text in Notes without inventing authorship; use an explicit unknown-author
   label where authorship is unavailable. Do not duplicate it as two notes.
8. When an Inbox document is consumed as a new version of another document,
   transfer its notes and history to the target alongside the version operation.
   Preserve text, authorship, timestamps, states, and replacement relationships;
   retain the target's existing notes. A failed operation must not partially
   transfer or lose notes. Any metadata-loss warning must accurately distinguish
   preserved notes from metadata that is still lost.

## Acceptance Criteria

1. Opening document details in Browse, Inbox, or Trash exposes Notes on desktop
   and mobile, regardless of the Space's folder mode.
2. Creating two notes produces two separate list entries, newest first, each
   showing the creating user's name and the submitted multiline text.
3. Submitting empty or whitespace-only text fails with a useful validation
   message and leaves the notes unchanged.
4. Editing a note changes that entry in place, preserves its author and creation
   time, and shows edit attribution without creating a replaced entry.
5. Replacing a note shows a new current entry attributed to the replacing user.
   With history off, the original is hidden; with history on, its unchanged text,
   original author, replaced state, and replacement relationship are visible.
6. A failed replacement leaves the original current and creates no partial
   replacement. A stale second replacement of the same original is rejected.
7. Confirming deletion hides a current note by default. Turning history on
   reveals the same text and author with a deleted label. Cancelling deletion
   leaves the note current.
8. A list containing current, deleted, and replaced notes shows only current
   entries with the toggle off and all three categories with it on. Toggling
   does not mutate notes or grant additional access.
9. Historical entries offer no mutation actions. Deleting a current replacement
   does not restore earlier entries to the default list.
10. Authors and the relevant Space/tenant owners can change current notes on an
    editable document. Other users cannot change those notes through either the
    UI or direct requests, but eligible users can create their own notes.
11. A user without document access cannot read or change its notes, even with
    known identifiers or the history toggle enabled.
12. Trash exposes the notes list and toggle but no mutation actions; direct
    mutation attempts fail. Restoring the document preserves individual note
    states and makes permitted actions available again.
13. Normal document moves, Inbox filing, and uploading a new document version
    preserve the notes and their history.
14. Existing non-empty legacy document notes remain readable in Notes and are
    not falsely attributed to an uploader, owner, or viewing user.
15. Note actions and the toggle work with a keyboard, have understandable labels
    and states, and show localized application messages without translating note
    content.
16. Adding an Inbox document as a version transfers its current, deleted, and
    replaced notes to the target without losing attribution or replacement
    relationships, duplicating entries, or changing the target's existing notes.
    If the operation fails, neither document's notes are partially transferred.

## Non-Goals

- Implementation planning, schema design, migration generation, or code changes.
- Rich text, Markdown rendering, attachments, mentions, threaded replies,
  notifications, reactions, or private notes.
- A revision history for in-place edits or a general audit-log viewer.
- Restoring, permanently deleting, or modifying historical note entries.
- Full-text note search, export, or a new external notes API.
- Notes on folders or separate notes per uploaded document version.
- Changing existing version annotations or document access roles.
- Introducing a new permanent-document-deletion or retention policy.

## Durable Invariants

1. A note and all its history remain within the access boundary of their document.
2. New note authorship comes from the authenticated actor, never client input.
3. Editing preserves original authorship; replacing creates separately attributed
   content and preserves the predecessor.
4. Soft deletion and replacement do not erase the retained note text.
5. Historical entries are not current and cannot be changed through note actions.
6. Showing history is presentation state, not a data mutation or permission grant.
7. Trashed documents cannot receive note mutations.

## Safe Assumptions

- "All document views" covers existing document detail presentations in Browse,
  Inbox, and Trash, not a new details panel inside every preview dialog.
- "Toolbar" means the Notes tab's toolbar rather than a global application
  toolbar. One toggle controls both deleted and replaced entries.
- The initial toggle state is off; no account-wide saved preference is required.
  Its setting survives note actions within the currently displayed Notes tab.
- Plain text, newest-first order, and creation/edit attribution provide the
  baseline list experience without adding a rich-text or audit subsystem.
- Author names follow existing account display-name behaviour. If an author is
  unavailable, show an explicit fallback without attributing their note to
  another person. Historical name snapshots are not required.
- Legacy document notes are preserved and shown with unknown authorship where
  necessary. Only eligible owners can change an authorless current legacy note.

## Unresolved Questions

None currently identified that block the behavioural specification.

The edit/replacement distinction, author-and-owner permissions, and read-only
Trash behaviour, plus transfer of notes during Inbox version merges, were
confirmed by the user. No dedicated architectural follow-up has been requested
or performed.
