# Native share authentication execution record

Date: 2026-10-26

## Change

The phone's share POST could omit the Lax session cookie. Authentication redirected
to `/`; the following authenticated GET redirected to the dashboard, discarding the
share body. Secure cookies now use `SameSite=None`, and authenticated requests
refresh existing cookie attributes without unnecessary expiry renewal.

Decision and security tradeoffs: [ADR](../../adrs/2026.10-native_share_session_cookies.md).

## Checks

Run from the `simpledms/` module:

- Passed: `go test ./server ./util/cookiex -run '^(TestNativeShareAuthenticationAndCrossOriginProtection|TestSessionCookieSameSiteMatchesSecureMode|TestRenewSessionCookie.*)$' -count=1`.
  Covers issued/renewed/deleted cookies, migration without extending fresh or
  temporary sessions, the unauthenticated POST-to-dashboard redirect chain,
  and cross-origin rejection with and without Fetch Metadata headers.
- Passed: `go test ./server -run '^(TestSignInCmdSetsSessionAndRedirect|TestTemporarySetupSessionExpiresAtDeletableAt)$' -count=1`.
- Blocked: `go test ./server -run '^(TestUploadFilesCmdCreatesTemporaryFilesAndRedirects|TestConsumeUploadsCmd.*|TestSignInCmdSetsSessionAndRedirect|TestTemporarySetupSessionExpiresAtDeletableAt)$' -count=1`
  with the repository SQLite build tags. Both encrypted and unencrypted shared
  upload cases failed during setup: the S3 service at `localhost:7070` refused
  the connection. This did not exercise file ingestion.
- Module regression: `go test ./... -skip '^(TestConcurrentUploadFileCmdWithSlowS3|TestUploadFileCmdFailsWhenS3Unavailable|TestUploadFilesCmdFailsWhenS3Unavailable)$'`.
  All packages except `server` passed. The server package had 42 failed test
  headings caused by the unavailable S3 service. The original combined native
  share test was blocked before assertions; its authentication/CSRF cases were
  subsequently split from the storage case and passed in the focused run above.
  Unrelated slow bad-network simulations were excluded as required by `AGENTS.md`.
  Full log: `/tmp/opencode/simpledms-go-test.log`.
- Browser: Playwright CLI with Chromium and mocked HTTPS origins confirmed a
  form POST from `https://share-source.invalid/` to
  `https://simpledms.example.test/share` sent the `SameSite=None; Secure;
  HttpOnly` test cookie but omitted the Lax test cookie. A same-site,
  cross-origin control sent both. This checks browser cookie policy, not the
  exact native Android share navigation or its Fetch Metadata headers.

Go server checks used `CGO_ENABLED=1` and
`GOFLAGS='-tags=sqlite_fts5,sqlite_json,sqlite_foreign_keys,sqlite_icu'`.

## Coverage follow-up

The initial S3 blocker above was an endpoint mismatch, not a stopped storage
service: Docker exposes VersityGW on host port `3900`, while the test fallback
uses `7070`.

- Passed with `SIMPLEDMS_S3_ENDPOINT=localhost:3900`:
  `go test ./server -run '^(TestNativeShareUploadReachesInboxAfterSpaceSelection|TestNativeShareAuthenticationAndCrossOriginProtection)$' -count=1`.
  The expanded upload test covers both encryption modes, two shared files, the
  real Space-selection page, no ingestion during that query, the selection
  command, both files in the first Inbox response, ownership and source, and
  replay without duplicates.
- Added `e2e/native_share.spec.ts` to the normal Playwright suite. It tests the
  manifest-driven multipart upload through Space selection to the Inbox, an
  untouched app-issued secure cookie on a cross-site submission, and a deliberate
  Lax-cookie negative control reproducing the dashboard redirect. Only native
  Origin/Fetch Metadata are adapted; the browser chooses the request's cookies.
  The cross-site cases explicitly skip insecure development cookies.
- Passed discovery: `npx --no-install playwright test e2e/native_share.spec.ts --list`
  lists all three Chromium tests. Runtime execution remains blocked because no
  app is listening at `https://localhost:7003`; discovery is not an execution pass.
- Passed module regression with the corrected storage endpoint and the SQLite
  build flags above:
  `go test ./... -skip '^(TestConcurrentUploadFileCmdWithSlowS3|TestUploadFileCmdFailsWhenS3Unavailable|TestUploadFilesCmdFailsWhenS3Unavailable)$'`.
  All packages passed. Only the three unrelated bad-network tests were excluded.
- Passed a final focused rerun of
  `go test ./server -run '^TestNativeShareUploadReachesInboxAfterSpaceSelection$' -count=1`
  in both encryption modes after tightening the assertion to the rendered
  `hx-post` URL and exact `UploadToken` form key/value.

## Manual follow-up

After deployment, reopen the installed app once to refresh an existing cookie,
then share a file from another phone app and select its destination Space.
Actual Android OS share-sheet interaction has not been exercised here.
