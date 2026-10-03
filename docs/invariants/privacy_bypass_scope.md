# Privacy Bypass Scope Invariants

## Scope

These invariants apply to every use of `privacy.DecisionContext(ctx, privacy.Allow)`
from `db/entmain/privacy` or `db/enttenant/privacy` in request paths, including the
browser router, WebDAV, MCP, and model services.

## One Decision Key For Both Databases

Rule: The generated `entmain` and `enttenant` privacy packages share Ent's single
decision context key. An `Allow` decision created for a main-database lookup also
disables every tenant policy, including Space membership filtering, for any context
derived from it.

## Bypass Contexts Never Become Request Contexts

Rule: A bypass context is passed only to the specific query or mutation that needs it.
It must never be the parent of a `VisitorContext`, `MainContext`, `TenantContext`, or
`SpaceContext`, and must never be assigned back to a request-scoped `ctx` variable.

Credential bootstrap (resolving the credential's account and tenant before an
authenticated context exists) uses a separate local bootstrap context. The resulting
Space context is built from the original request context so that Space policies apply.

Executable checks:
`TestWebDAVCredentialStopsWorkingAfterUserIsRemovedFromSpace` (WebDAV) and
`TestMCPSpaceAccessLossKeepsTenantMembershipButDeniesOnlySpace` (MCP).
