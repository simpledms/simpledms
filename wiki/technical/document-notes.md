# Document Notes

Notes belong to a tenant document, not a file version. Browse, Inbox, and Trash share
the Notes partial; folders are excluded. Trash is read-only even for owners. New notes
are plain multiline text, ordered by authored time descending with an ID tie-breaker;
undated legacy entries follow dated entries.

## Access and Lifecycle

`model/tenant/file/DocumentNotes` owns access checks and transitions. Every request checks
the actual document/Space/tenant and current membership; public IDs and history state
do not grant access. Creation uses the authenticated actor. Editing, deleting, or replacing
also requires authorship or relevant Space/tenant ownership. Global admin status alone
does not grant that authority. Commands run through the router's transaction wrapper;
callers must roll back the entire transaction on any transition error.

Edits retain identity, author, and authored time, recording a separate last editor/time.
Replacement atomically creates a newly authored successor and links the retained original.
Conditional current-row writes and SQLite's transaction writer lock prevent two successful
successors. Deleted/replaced notes are immutable history; deleting a successor never
reactivates its predecessor. `deleted_at` is explicit, without `SoftDeleteMixin`, because
that mixin's skip context can turn a delete into physical deletion. History is only a filter.

## Legacy and Transfers

Untouched `File.notes` is displayed once with unknown author/date, without writes on reads.
The document-scoped `legacy` selector materializes it only on an authorized mutation or
before an Inbox merge deletes its source. Claiming/clearing the legacy value and inserting
the note occur in the same transaction; no backfill or invented authored time is used.

Ordinary filing, versions, Trash, and restore retain document-owned notes. Inbox version
merge materializes source legacy text and reparents **all** source notes before hard
deletion, including deleted/replaced entries and replacement links. Target notes and legacy
text remain intact. Transfer is not a text edit and does not change author/editor metadata
or require authority to edit each transferred author's content.

## HTMX and Mobile Details

`DocumentNoteContent` and its template live in `ui/uix/partial`, alongside other
application-specific UI such as navigation composition, not the generic
`core/ui/widget` package. Application partial templates are registered alongside
core templates in production and test renderers, and included in Tailwind scanning.

- History starts off per opened tab. The partial's `hx-vals` retain the rendered
  `ShowHistory` value for command refreshes; the history icon sends its inverse.
  No switch or hidden checkbox is involved.
- The partial and history button synchronize requests with `#documentNotes:replace` so stale
  responses cannot win rapid toggles. Commands trigger the shared notes refresh event.
- Use the note form dialog's own auto-close, not global `closeDialog`: the latter also
  closes the enclosing mobile Details sheet.
- The toolbar uses icon-only history/add buttons with tooltips and accessible names.
  History has `aria-pressed` and a Primary-colored selected icon, following the
  [Material 3 standard icon-button spec](https://m3.material.io/components/icon-buttons/specs).
  Unselected icons/state layers use On surface variant; selected ones use Primary.
  Hover adds an 8% state layer, with no persistent tonal container fill.
  The optional `IconButton`
  label/toggle fields and explicit HTMX role leave existing icon controls unchanged.
- Reuse `TextArea`; render full escaped, wrapping text rather than truncated list labels.
- Each note uses `ListItem.Content` for full text and attribution. Eligible current
  notes expose Edit/Replace/Delete through `ListItem.ContextMenu`, with a trailing
  vertical-three-dot icon button (Actions tooltip) for touch/keyboard use.
   Historical and Trash entries omit that menu.
- Rows show the note title on the main line and note text on the secondary line.
  Creator and creation date appear in details and in the row's native title tooltip.
  Edit attribution/time remains in details;
  Deleted/Replaced status remains outside the secondary line. Clicking
  the row (or activating its content button by keyboard) opens `DocumentNoteDialog`
  with operation `view`, which uses scoped `DocumentNotes.Get` without mutation
  authorization. Details include full metadata and replacement navigation.
- Create/edit/replace require nonblank title and body and label both form fields.
  The nullable title column is added by migration `20260907210921` without a table
  rebuild. Existing titleless and legacy notes use translated "Note" as a display
  fallback, without persisting invented titles. Titles follow body edit/history and
  transfer semantics.
- Replaced summaries omit the retained body and display an italic, translated
  replacement reference using the immediate successor's escaped title. Full details
  retain the body, with separated title/content, attribution, and replacement sections.
- Read rendering computes owner eligibility once, then uses loaded authorship per row;
  commands independently recheck authorization.
- Inbox-root filing must carry the resolved hidden `CurrentDirID` and may file a document
  even when its parent already equals that root; clearing Inbox state is still a change.

See the [feature plan](../../docs/features/2026.09-document_notes/plan.md),
[invariants](../../docs/features/2026.09-document_notes/invariants.md), and
[verification record](../../docs/features/2026.09-document_notes/verification.md) for
delivery evidence, optional manual checks, and the unrelated recursive-generation caveat.
