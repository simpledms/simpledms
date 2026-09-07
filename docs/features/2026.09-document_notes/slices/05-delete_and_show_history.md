# 05: Delete And Reveal A Note

Status: Implementation complete. [Delivered verification and caveats](../verification.md).
Unchecked compound items retain remaining optional/manual verification, not missing features.

Depends on: [04](04-edit_current_note.md). [Plan](../plan.md).

Journey: Confirm deletion of a current note, then reveal its retained text using
the Notes toolbar's history toggle.

## Acceptance Criteria

- Authorized authors/owners can confirm soft deletion in Browse and Inbox.
  Cancel leaves the entry current; successful deletion hides it by default.
- One labelled toolbar toggle, off initially, reveals deleted entries alongside
  current notes in all three views, including otherwise-empty lists and Trash.
- Historical entries show a deleted state and no mutation actions. Direct edit
  or delete requests against them fail; viewing history never physically deletes.
- Create/edit/delete refreshes preserve the toggle state and selected Notes tab.
- Filing, Trash/restore, and Inbox merges retain deleted note text and attribution
  without making those notes current again. Legacy deleted notes do not reappear
  from the old field on reload or retry.

## Implementation Checklist

- [x] Add explicit nullable `deleted_at` state (without `SoftDeleteMixin`) and a model
  transition restricted to a current note; reuse the author/owner policy from 04.
  Make the edit path reject deleted notes in the same slice.
- [x] Add the confirmation dialog/command and integrate legacy materialization
  atomically with deletion. Do not use an include-deleted context for a delete call.
- [x] Compose the existing toolbar/Switch widgets with a labelled on/off state.
  Query history read-only with explicit current/history predicates and document scope;
  no note soft-delete hook or skip context is needed.
- [x] Pass the filter through list refreshes and dialog returns; show deleted
  status in text and retain original/last-editor attribution. Translate new text.
- [x] Extend merge note selection to include deleted rows before source deletion.
  Transfer without changing lifecycle or edit metadata. Generate schema/migrations.

## Verification Checklist

- [x] Automated tests cover retained deleted entries, historical-write rejection,
  role restrictions, and history carried through transfer/filing/Trash/restore.
  The recorded browser runs also cover rapid toggles and default-off reopen behaviour.

- [ ] Test cancel/confirm, role matrix, soft retention, invalid/stale deletes,
  legacy retry safety, direct edit denial, and history queries under revoked access.
  Assert no history query widens Space/tenant scope and no action physically deletes.
- [x] Test deleted notes through filing, new versions, Trash/restore, and Inbox
  merge with transaction rollback; prove none is reactivated or omitted in transfer.
  `document_notes_versions_test.go` additionally verifies ordinary version upload
  with deleted history in both encryption modes.
- [x] Extend desktop/mobile Playwright to delete notes, toggle on/off,
  assert text/state and action visibility, create/edit while history is on, and
  inspect the same history in Trash. Check keyboard switch focus/toggling and labels.
- [ ] Manually cancel a deletion, then confirm it, toggle repeatedly, and reopen
  the Notes tab. Confirm default-off initial state and preserved state during actions.
- [x] Run notes and merge Go tests, notes Playwright, and the plan's build,
  format, translation, and schema upgrade gates.
