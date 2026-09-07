# 04: Edit A Current Note

Status: Implementation complete. [Delivered verification and caveats](../verification.md).
Unchecked compound items retain remaining optional/manual verification, not missing features.

Depends on: [03](03-annotate_inbox_document.md). [Plan](../plan.md).

Journey: Correct your own note, or correct one as a relevant owner, and see the
original author retained with separate last-editor attribution.

## Acceptance Criteria

- Authors, Space owners, and tenant owners with document access can edit current
  notes in Browse and Inbox. Other users cannot edit them via UI or direct request.
- Saving changes the same entry in place without changing order, original author,
  or creation time; it shows the last editor and edit time, not a history entry.
- Eligible owners can edit authorless legacy notes. Successful materialization
  removes their old representation exactly once and retains unknown authorship.
- Empty edits, cancellation, stale document access, and Trash writes change nothing.
- Filing or merging an edited note preserves edit attribution and does not display
  the moving user as a new editor.

## Implementation Checklist

- [x] Introduce shared current-note authorization using actual document membership,
  authenticated author, and re-read Space/tenant ownership rather than cached roles.
  Validate submitted note/document pairs; do not rely on hidden action buttons.
- [x] Add model edit behaviour and the edit-attribution fields it consumes.
  Keep last text-edit attribution separate from generic transfer/update metadata.
- [x] Add the edit command and populated dialog, using the shared list refresh
  flow in Browse and Inbox and existing dialog behaviour on validation/close.
  Independent keyboard focus/return inspection remains optional.
- [x] Reuse transactional legacy materialization from slice 02 before an owner
  edit. Never assign the acting owner as the legacy note's original author.
- [x] Render authorized per-entry Edit actions and edited labels; no actions in
  Trash. Translate labels and errors and generate schema changes through tooling.

## Verification Checklist

- [x] Model/HTTP tests cover member/author/owner permissions, revoked access, mismatched
  identifiers, and retained author with separate editor attribution. Browser tests cover
  in-place edit and persistence, with Inbox edit in the filing journey and cancel/invalid
  edit in Browse; this is narrower than repeating every case in both views below.

- [ ] Add a role matrix in SQLite-backed tests: author, unrelated Space member,
  Space owner, tenant owner, unrelated global admin, and revoked author. Test both
  UI action visibility and direct requests, including mismatched document/note IDs.
- [ ] Assert unchanged identity/order/author/time, last editor display, unknown
  legacy authorship, failed-edit rollback, and transfer without edit-marker changes.
- [ ] Extend desktop/mobile Playwright: create, edit, cancel an edit, fail an empty
  edit, reload, and verify unchanged entry count plus edited attribution in Browse
  and Inbox. Keep exhaustive multi-account permission assertions in Go tests.
- [ ] Manually edit another author's note as an owner on disposable fixtures;
  confirm both names have distinct labels and the Trash view has no Edit action.
  Remaining optional independent walkthrough; owner/other-author attribution is covered
  by Go fixtures, not a separate multi-account browser session.
- [x] Run focused notes/merge Go tests, notes Playwright, and all applicable plan
  gates, including additive migrations and fresh-schema/upgrade coverage.
