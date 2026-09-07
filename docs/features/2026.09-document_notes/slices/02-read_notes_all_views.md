# 02: Read Notes In Every Document View

Status: Implementation complete. [Delivered verification and caveats](../verification.md).
Unchecked compound items retain remaining optional/manual verification, not missing features.

Depends on: [01](01-create_browse_note.md). [Plan](../plan.md).

Journey: Read the document's existing notes, including legacy text, from Browse,
Inbox, and Trash without losing notes when the document changes location or state.

## Acceptance Criteria

- Notes is reachable in all three detail tab sets, both Space modes, and compact
  and expanded layouts. All show the same document-owned content and attribution.
- Non-empty legacy text appears exactly once with unknown author/date treatment,
  alongside new notes. Opening a tab does not mutate or convert stored data.
- Trash is read-only, and restore preserves notes. Restoring into Inbox because
  the original parent is missing also leaves notes readable.
- Materializing legacy text for a version merge preserves it exactly once on the
  target, even if the source never opened Notes. Failures lose neither form.
- The existing Trash metadata note display is not left as a contradictory second
  notes UI. No new creation or mutation control is introduced for Inbox yet.

## Implementation Checklist

- [x] Add read-only reuse of the shared notes partial to Inbox and Trash tab
  composition, preserving their navigation state and existing sheet behaviour.
- [x] Scope include-deleted document lookup inside `DocumentNotes` for Trash reads;
  reject mutations on deleted documents without widening mutation contexts.
- [x] Project untouched `File.notes` as one legacy entry in the shared list.
  Use an explicit document-scoped legacy selector and honest unknown attribution;
  preserve the original text, including legacy whitespace, rather than applying
  new-note validation to existing data. Exclude folders.
- [x] Add a narrow transactional legacy-materialization operation for the merge
  path: re-read the field, create one authorless note, and clear the field together.
  Preserve unknown original time in its display. Do not add a background backfill.
- [x] Use it before source deletion in the existing merge service, then transfer
  source notes with the slice 01 path. Retain any target legacy text independently.
- [x] Replace the old Trash metadata text block with the Notes destination;
  localize new unknown-author/date and read-only messaging.

## Verification Checklist

- [x] Go fixtures exercise all tab sets in both folder modes, legacy rendering,
  revoked access, and transactional legacy transfer/retry/rollback. Browser lifecycle
  checks passed. The broader compound scenarios below are not all independently evidenced.

- [x] Add Go render/query tests for all view/mode combinations, revoked access,
  legacy-plus-new entries, unavailable users, unknown date ordering, and no writes
  during read-only requests. Prove migration/materialization is retry-safe.
- [x] Test note survival across Trash/restore, including missing-parent restore,
  and legacy source merge success/rollback through the transaction-owning path.
- [x] Exercise desktop/mobile Playwright Inbox creation, filing to Browse, Trash
  read-only Notes, and restore with retained history. Go fixtures cover legacy data.
- [ ] Manually follow that lifecycle and inspect existing legacy text in a
  disposable upgraded database. Confirm no duplicate Metadata/Notes presentation.
  This independent manual legacy walkthrough remains optional; migration replay and
  legacy rendering/materialization have automated coverage.
- [x] Run focused notes and Inbox-merge Go tests, the notes Playwright suite,
  and the plan's build/format/translation gates; apply schema gates if needed to
  distinguish unknown legacy dates without inventing attribution.
