# Document Notes Invariants

Status: Implemented contract; verification scope and remaining checks recorded separately

Source: [Specification](spec.md). Delivery and checks: [Plan](plan.md).

## Rules

| ID | Durable rule | First enforcement and subsequent extensions |
| --- | --- | --- |
| N1 | Every note read or mutation is scoped to its actual document, Space, and tenant. Public identifiers and history state never confer access. | Slice 01; all later slices |
| N2 | New authorship comes from the authenticated actor. Original authorship and creation time survive edits, deletion, replacement, and transfer. Unknown legacy authorship is never invented. | Slices 01-02; 04-06 |
| N3 | Creating requires document mutation access. Changing an existing current note additionally requires authorship or relevant Space/tenant ownership. Losing document access revokes note access, including for the author. | Slice 01 creation; slice 04 shared change authorization |
| N4 | A trashed document permits note reads but no note mutations, regardless of caller role or request origin. Restoring it does not independently change note states. | Slice 01 rejection; slice 02 Trash journey; later actions reuse it |
| N5 | Editing changes current text in place. Replacement creates separately authored content and makes its predecessor read-only in one atomic transition. | Slices 04 and 06 |
| N6 | Deleted and replaced entries retain their text, attribution, and relationships. Note actions cannot change historical entries or reactivate a predecessor. At most one replacement can succeed for a current entry. | Slices 05-06 |
| N7 | The history toggle changes only visibility. Including history cannot bypass document authorization or turn a soft-delete request into physical deletion. | Slice 05; mixed history in 06 |
| N8 | Ordinary document moves, new file versions, Trash, and restore preserve notes. Inbox version merges transfer all notes before source deletion, atomically with the merge, without duplicates or changes to target notes. | Slices 01-03; historical rows in 05-06 |
| N9 | Legacy text is represented once. Materialization and clearing the old field succeed together; read requests never perform that write. Missing original dates are not fabricated. | Slice 02; mutation paths in 04-06 |
| N10 | Transfer and lifecycle bookkeeping do not masquerade as a text edit or overwrite last-editor attribution. | Slice 04; 05-06 |

These remain the feature contracts. See [verification](verification.md) for evidence
and unperformed checks, and the [technical note](../../../wiki/technical/document-notes.md)
for durable implementation guidance. Implementation completion is not a claim that every
planned manual or concurrency scenario has been independently exercised.
