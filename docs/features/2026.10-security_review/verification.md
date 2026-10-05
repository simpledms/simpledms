# Security review execution record

Date: 2026-10-03  
Fixes: commit `27d606f`, follow-up test coverage, and fixes from a second code review.

## Scope

Reviewed: authentication (password, temporary password, passkey recovery, assisted
recovery), sessions and cookies, CSRF, HTTP server setup, tenant and Space
authorization (Ent privacy policies, commands, and queries), WebDAV and MCP credential
authentication, downloads and previews, URL import (SSRF), archive extraction, and
template escaping.

## Findings and fixes

Each finding has a regression test. Unless marked otherwise, the test failed before the
fix and passes after it.

| # | Severity | Finding | Fix | Regression test |
| --- | --- | --- | --- | --- |
| 1 | High | A WebDAV credential kept working after its owner was removed from the Space. The bypass context for the credential login lookup became the parent of the request context, which turned off Space membership policies. | `server/webdav/handler.go` keeps the bypass on bootstrap lookups only. Rule: [privacy bypass scope](../../invariants/privacy_bypass_scope.md). | `server/webdav_space_membership_test.go` |
| 2 | High | Stored XSS: inline downloads served uploaded HTML/SVG with its detected type from the application origin. | `SetDownloadSecurityHeaders` (`action/common/download_helper.go`) adds `nosniff` and a `sandbox` CSP, except for inline PDFs. It is used by the download, trash, PDF preview, and HTML source handlers. Decision: [ADR](../../adrs/2026.10-uploaded_file_sandboxing.md). | `server/download_security_headers_test.go` (download, attachment, trash, HTML source routes; needs S3), `action/common/download_helper_test.go` (header policy) |
| 3 | High | `SetInitialPasswordCmd` replaced an existing password without asking for it, so a stolen session could take over the account permanently. | `Account.SetInitialPassword` refuses when a password is already set. It also signs out other sessions, because the temporary password may have been intercepted. | `server/set_initial_password_cmd_test.go` |
| 4 | Medium | Organization user list and settings were only hidden in the navigation. Members could request them directly and see all member names and emails. | `autil.RequireTenantOwner` in `ManageUsersOfTenantPage`, `UserListPartial`, `OrganizationSettingsPage`. It checks the tenant user role, like the Space policy and the user management commands. | `server/tenant_management_query_access_test.go` |
| 5 | Medium | Behind a trusted reverse proxy, sign-in, password reset, and passkey rate limits were keyed on the proxy address, so one client could lock out everyone. The existing WebDAV and maintenance limiters read only the first `X-Forwarded-For` line, which is client-controlled when a proxy adds its own line. | `httpx.ClientIPThroughTrustedProxies` reads all header lines and accepts entries with ports. The router stores the result for `httpx.Request.ClientIP`, which the auth and passkey rate limits use. WebDAV and the maintenance limiter call the same resolver. | `util/httpx/client_ip_test.go`, `server/trusted_proxy_config_test.go`: `TestAuthRateLimitsArePerClientBehindTrustedProxy` (password and passkey), `TestRouterResolvesClientIPOnlyThroughConfiguredProxies` |
| 6 | Medium | A password change left all other sessions signed in. | `Account.ChangePassword` signs out all other sessions of the account. | `server/change_password_cmd_test.go` |
| 7 | Medium | HTTP servers had no timeouts (Slowloris). | `newHTTPServer` sets `ReadHeaderTimeout` and `IdleTimeout` for the main, maintenance, and autocert listeners. Body timeouts are left off on purpose. | `server/http_server_test.go`: `TestNewHTTPServerBoundsHeaderAndIdleTimeButNotBodies` (configuration check, not a timing test) |
| 8 | Low | No anti-framing or `nosniff` default headers. | `newPublicHandler` adds the headers on the main and maintenance handler chains. | `server/http_server_test.go`: `TestNewPublicHandlerDeniesCrossOriginFramingAndSniffing` |
| 9 | Low | The URL import blocklist missed `0.0.0.0/8`, `192.0.0.0/24`, and IPv6 prefixes that embed IPv4 targets (IPv4-compatible, NAT64, Teredo, 6to4). | Added to the blocked prefixes. Loopback is checked first, so dev mode still allows `::1`. | `model/main/temporaryfile/upload_from_url_service_test.go` (added with the fix; not confirmed failing before it) |

Findings 7 and 8 were fixed in `27d606f` before the server setup was pulled into
`newHTTPServer` and `newPublicHandler` to make it testable. Their tests cover the
constructors that all listeners use, not a red/green reproduction.

Before/after confirmation for 2 and 5 (password and passkey): removing the fix made the
route-level test fail (missing CSP; second client rate limited), and restoring it made
the test pass.

## Follow-up code review

A second review of the changes found ten issues. Outcomes:

- Fixed: passkey rate limits still used the proxy address. `X-Forwarded-For` read only
  the first header line and rejected entries with ports. Idle keep-alive connections
  had no timeout. The URL import blocklist missed IPv6 transition prefixes. The
  session sign-out rule lived in the command handler; it now lives in the `Account`
  model and also covers the initial password.
- Refactored: three copies of the client IP resolution (router, WebDAV, maintenance
  limiter) became `httpx.ClientIPThroughTrustedProxies`. The owner check became
  `autil.RequireTenantOwner`. The sandbox policy was exported and is now used by the
  preview handlers. Code style: line length and one field per line.
- Not changed: credentials created during a session survive a password change. See
  open items.

## Reviewed without changes

- CSRF: commands are POST-only behind `http.CrossOriginProtection`, session cookies
  are `SameSite=Lax`, and pages use read-only transactions.
- Space isolation in the database layer (Space mixin policies), MCP authentication and
  read-only tool enforcement, WebDAV and MCP secrets (generated, salted or hashed,
  constant-time comparison), password hashing (PBKDF2-SHA512, 250k iterations),
  passkey recovery codes (100 bits, rate limited), archive extraction (virtual paths,
  enforced sizes), templates (`html/template`, only trusted `template.HTML`).
- A tenant owner can edit owned member accounts on purpose
  (`TestEditAccountCmdAllowsOwningTenantAdmin`).

## Open items

- A password change signs out browser sessions but keeps WebDAV and MCP credentials and
  passkeys. An attacker with a stolen session can create such a credential and keep
  access. These credentials are kept on purpose: they belong to scanners and
  integrations, and are listed and revocable on the credential pages. Alternatives:
  revoke them on password change, or require the current password to create them.
- Session tokens are stored unhashed in the main database, so a database copy allows
  session hijacking. Hashing them signs everyone out once when deployed.
- Account enumeration: passkey-only accounts get a specific sign-in error, and unknown
  emails skip PBKDF2, so responses are measurably faster.
- URL import with `HTTP(S)_PROXY` set: connections go to the proxy, so the IP
  blocklist doesn't apply to the target. The proxy must block internal targets itself.
- Functional bug: `Account.ChangePassword` fails with "Temporary password expired" when
  an expired temporary password is still stored, even if the current password is
  correct.

## Verification

Second-review results:

| Command | Result |
| --- | --- |
| `go build ./...`, `go vet ./server/... ./action/... ./util/... ./model/main/...`, `gofmt -l` | Passed, no output. |
| Security regression tests (command below) | 16 tests passed. S3 at `localhost:3900` from `.env`. |
| `go test ./server/... ./action/... ./util/... ./model/main/... -skip 'WithSlowS3\|WhenS3Unavailable\|Benchmark'` | 20 packages ok. 5 failures, which also fail on clean `3d9e05e`: `TestInboxTransferHTTPPreservesDocumentAndHistoryClearsClassification`, `TestMCPMetadataManagementCRUD`, `TestUploadFromURLStagedFilePersistsAfterRequestCancellation`, `TestUploadFromURLFirstInboxResponseContainsImportedFile`, `TestDevelopmentSchemaCreateAddsIndexedColumnsWithoutRebuildingTables`. |
| `go generate` in `i18n` (first pass) | Completed. New de/fr/it strings translated and marked fuzzy. Existing unrelated missing entries remain. No new strings in the second pass. |

Not performed: browser checks of inline previews under the sandbox CSP (HTML, image,
video, PDF in Chromium and Firefox).

```sh
go test -v ./server ./server/webdav ./action/common ./model/main/temporaryfile ./util/httpx -count=1 -run '^(TestWebDAVCredentialStopsWorkingAfterUserIsRemovedFromSpace|TestMCPSpaceAccessLossKeepsTenantMembershipButDeniesOnlySpace|TestDownloadRoutesSandboxUploadedActiveContent|TestSetDownloadSecurityHeadersSandboxesEverythingExceptInlinePDFs|TestSetInitialPasswordCmdOnlySetsMissingPassword|TestTenantManagementQueriesRequireTenantOwner|TestRouterResolvesClientIPOnlyThroughConfiguredProxies|TestAuthRateLimitsArePerClientBehindTrustedProxy|TestClientIPThroughTrustedProxiesIgnoresClientControlledEntries|TestWebDAVRateLimitRemoteAddrUsesOnlyTrustedForwardedChain|TestChangePasswordCmdSignsOutOtherSessions|TestNewHTTPServerBoundsHeaderAndIdleTimeButNotBodies|TestNewPublicHandlerDeniesCrossOriginFramingAndSniffing|TestUploadFromURLRejectsInternalTargetAddresses|TestUploadFromURLAllowsLoopbackOnlyInDevMode|TestRouterTrustsForwardedHTTPSOnlyFromConfiguredProxy)$'
```
