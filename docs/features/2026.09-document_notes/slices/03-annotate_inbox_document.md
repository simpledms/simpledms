# 03: Annotate An Inbox Document

Status: Implementation complete. [Delivered verification and caveats](../verification.md).
Unchecked compound items retain remaining optional/manual verification, not missing features.

Depends on: [02](02-read_notes_all_views.md). [Plan](../plan.md).

Journey: Add a note while processing an Inbox document, then find it on the
document after ordinary filing or on the target after adding it as a version.

## Acceptance Criteria

- An eligible Inbox user can create notes through the same validated form used
  by Browse; saving stays in Inbox with Notes selected.
- Filing in folder and non-folder modes retains note identities and attribution.
- Adding the Inbox file as a version transfers its notes and legacy text to the
  target, retains target notes, and removes the source without duplicates.
- The merge warning distinguishes preserved notes from other metadata still lost.
  Cancelling or failing the merge leaves the notes and document states unchanged.
- Trash remains read-only; Inbox entry points cannot bypass the shared access rules.

## Implementation Checklist

- [x] Enable the shared create journey in Inbox with its document/tab context and
  stable partial-refresh target. Avoid a duplicate model or separate create policy.
- [x] Verify `move_file_cmd.go` and `assign_file_cmd.go` preserve document identity;
  change only a proven note integration gap in these existing paths.
  Root filing needed a resolved hidden `CurrentDirID` and allowance for an already-root
  parent when clearing Inbox state; a dedicated HTTP test covers the fix.
- [x] Reuse the already-safe merge/materialization path from slices 01-02.
  Update `file_version_from_inbox_dialog.go`, its list/confirmation UI, and command
  validation text so the metadata-loss warning accurately excludes notes.
- [x] Refresh affected document notes after a merge using the existing events;
  do not change original author or text-edit metadata on a transfer. Translate text.

## Verification Checklist

- [x] HTTP root and non-folder filing preserve mixed history; transfer/merge fixtures cover legacy,
  attribution, same-Space enforcement, rollback, retry, and deleted-source rejection.

- [ ] Add command/render tests for Inbox creation, whitespace errors, correct
  author, direct unauthorized submissions, and preserving selected view context.
- [ ] Extend filing/assignment and merge tests for notes on both documents,
  unknown-author legacy notes, same-Space enforcement, rollback, and retry after
  source deletion. Verify no notes are duplicated or stranded on the deleted source.
- [x] Extend desktop/mobile Playwright checks: upload to Inbox, add a note, file
  normally and read it in Browse; repeat with another Inbox file and merge it into
  a target already containing a note. Verify both authors/texts on the target.
- [ ] Manually check both filing modes and cancel the revised merge confirmation.
  Confirm that only confirmed successful operations change document state.
  Remaining optional independent walkthrough: browser coverage uses folder filing;
  non-folder tab fixtures are Go coverage. The browser merge journey also cancels
  and reopens its confirmation before the successful merge.
- [x] Run focused notes/filing/merge Go tests, notes Playwright, and the plan's
  build, format, and translation gates. No new schema is expected.
