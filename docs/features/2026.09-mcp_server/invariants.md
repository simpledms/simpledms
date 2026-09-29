# MCP durable rules

Status: slices 01–02 enforcement and tests implemented but unverified; remaining rules planned

Source: [specification](spec.md). Delivery: [plan](plan.md). These are the durable rules newly
needed at the shared-command/MCP boundary. Slice 01 covers M1/M2/M3/M6; later-slice extensions
remain planned. Existing storage rules remain in their linked owners.
The slice checklists reference these IDs rather than define another competing contract.

## M1 — Credential authority is bounded and current

A credential acts only as its owning account, in its one tenant/Space, with its allowed mode,
intersected with current account membership and Space access. Browser Sessions, WebDAV secrets,
tool annotations, and MCP connection/session IDs cannot substitute for that authority. The
server rechecks it on calls; uploads also recheck before durable finalization. Revocation cannot
be bypassed by a previously authenticated connection or a request-local cached role.

First enforced in slice 01; write/finalization coverage in 02–04.

## M2 — External references do not expose or grant internal identity

MCP entity references are stable public IDs resolved within the permitted scope. No numeric
database ID, mutable label, or incomplete backfill is published as an external identifier.
New and upgraded metadata definitions receive the same contract. Reference possession alone
does not grant access.

First enforced in slice 01; metadata upgrade/creation coverage in 03, directories in 04.

## M3 — A transport cannot weaken execution guarantees

Equivalent HTMX and MCP operations execute the same model behaviour with its validation and
authorization. Query execution is read-only. Mutations release success only after their owned
transactions commit; failures cannot emit a success event or result. Storage operations retain
their separate prepare/I/O/finalize lifetime. Multi-tool sequences and separate SQLite databases
do not acquire implied atomicity or exactly-once delivery from MCP.

First enforced for reads/credential creation in 01; document writes in 02–04.

## M4 — Metadata mutations express desired state

Explicit assignment does not toggle, accumulate duplicate direct Tags, or treat missing input as
zero/false. Property values match their stored type, and removing an assignment is distinct from
storing an empty value. Changing a document type leaves other metadata intact. Browser toggle
interactions choose a desired-state model operation; resolved/composed Tags retain their meaning.

Enforced in slice 03, with preservation coverage in 04.

## M5 — Filing is one lifecycle transition

Only a live Inbox document can be completed. Folder filing changes parent/name and clears Inbox
state together, including when the root is already its parent. Non-folder completion keeps its
parent. No new mandatory metadata gate is introduced. Failed filing leaves source/destination
state intact, including any requested child-directory creation. Successful filing preserves the
document's identity, versions, source, Tags, properties, document type, and existing notes/history.

Enforced in slice 04; use existing note-preservation tests as regression evidence rather than
reimplementing note rules here.

## M6 — Credential secrets are not persistent presentation data

Credential creation returns the generated secret once after commit. Persistence contains only
its verification hash and metadata; list/reload views and logs cannot recover the token.
Only the owner can create or revoke through the initial credential-management feature.

Enforced in slice 01; upload payloads likewise must not enter request logs in 02.

## Existing rule owners

- [File storage safety](../../invariants/file_storage_safety.md): canonical-byte preservation,
  uncertain outcomes, promotion, and cleanup apply unchanged to MCP ingestion.
- [Shared ingestion/source rules](../../invariants/webdav_inbox_ingestion.md): use the shared
  upload integrity, quota, finalization, source, and recovery sections. WebDAV-specific method,
  zero-byte probe, path-alias, and Basic-auth rules do not define MCP behaviour.
- [Document note invariants](../2026.09-document_notes/invariants.md): existing note/history
  preservation remains a filing regression even though MCP note tools are deferred.

Numeric payload limits, tool lists, pagination defaults, and release priorities belong in the
specification/design documents; they are not additional durable domain invariants.
