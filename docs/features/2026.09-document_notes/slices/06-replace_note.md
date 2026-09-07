# 06: Replace A Note

Status: Implementation complete. [Delivered verification and caveats](../verification.md).
Unchecked compound items retain remaining optional/manual verification, not missing features.

Depends on: [05](05-delete_and_show_history.md). [Plan](../plan.md).

Journey: Supersede a current note with newly authored text and use the existing
history toggle to understand the retained original and successive replacements.

## Acceptance Criteria

- Authorized authors/owners can replace a current note in Browse and Inbox.
  The new note has their authorship and a new creation time; the original text
  and attribution remain unchanged and are hidden when history is off.
- With history on, current, deleted, and replaced entries show distinct states;
  predecessor/successor relationships are understandable through repeated replacement.
- A replaced entry rejects all note mutations. Editing the replacement changes
  only that entry; deleting it never reactivates its predecessor.
- Invalid input, cancellation, and failed replacement leave the original current.
  Concurrent/stale attempts cannot produce two successors for the same original.
- Trash and all access restrictions still apply. A full replacement chain survives
  filing, restore, and Inbox version merge with identities/attribution intact.

## Implementation Checklist

- [x] Add the narrowly scoped replacement relationship and model transition.
  Check the original is still current in the write condition and commit successor
  creation and predecessor transition in the same request transaction.
- [x] Reuse ownership, document/Trash guards, and legacy materialization. Attribute
  the successor to the acting user while preserving unknown legacy authorship.
- [x] Add Replace dialog/command and per-current-entry action; reuse validation,
  snackbar/events, and list refresh with history state from earlier slices.
- [x] Extend current-note filtering and all edit/delete guards to exclude replaced
  entries. Render replaced labels and clear links between list entries without
  requiring a new history page or changing newest-first order. Translate new text.
- [x] Keep merge transfer inclusive of all note states and replacement relationships;
  reparent the complete set before deleting the source, preserving edit metadata.
  Generate Ent and migrations using the additive workflow.

## Verification Checklist

- [x] Go and browser tests cover successor attribution, retained predecessor text,
  legacy replacement, historical-write rejection, invalid input/cancel, and transferred
  replacement relationships. Focused/race tests passed as recorded in verification.

- [ ] Add success, whitespace, cancel, permission, legacy, and historical-write
  rejection tests. Assert successor attribution and unchanged predecessor content.
- [x] Exercise two concurrent replacement attempts against real SQLite and assert
  exactly one successor; verify rollback using a real SQLite constraint failure
  during legacy replacement, without production test hooks.
- [ ] Add explicit edit-versus-replace and delete-versus-replace concurrent tests
  (remaining optional verification; the passing race run is not evidence of these cases).
- [ ] Extend merge and restore tests with mixed current/deleted/replaced notes and
  successive replacements, including rollback and both source/target histories.
- [x] Extend desktop/mobile Playwright: replace twice, inspect linked history,
  edit the current successor, delete it, and confirm neither predecessor becomes
  current. Merge an Inbox document with history and inspect the target's Notes.
- [ ] Manually check long histories and names in both sheet layouts, keyboard
  Replace/cancel, readable state labels, and no mutation actions in Trash/history.
  Remaining optional independent audit. Desktop/mobile screenshots were inspected
  for readability/scrolling; browser tests cover history controls and read-only actions.
- [x] Run the complete focused notes/merge regression set, notes Playwright, and
  plan gates. Review all [invariants](../invariants.md) and specification acceptance
  criteria against the accumulated tests; do not defer gaps to another test slice.
