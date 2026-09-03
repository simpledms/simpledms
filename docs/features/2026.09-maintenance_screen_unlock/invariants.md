# Maintenance Screen Unlock Invariants

These rules make the contract in the
[feature specification](spec.md#invariants) durable across the
[safe unlock core](slices/01-safe_unlock_core.md) and
[browser maintenance unlock](slices/02-browser_maintenance_unlock.md) slices.

## 1. Valid identity gate

**Rule:** A loaded identity is assigned only after decryption yields a non-nil, parseable X25519
identity. Every failure, including malformed decrypted plaintext, returns an error.

**Why:** A `(nil, nil)` result can falsely unlock startup and makes every caller unsafe.

**Enforced in:** [Slice 1](slices/01-safe_unlock_core.md), shared `DecryptMainIdentity` contract.

**Minimum regression tests:** A valid encrypted identity succeeds. A wrong passphrase and encrypted
non-identity plaintext return an error and leave the identity unset in both model and maintenance
command callers.

## 2. Reveal is presentation only

**Rule:** Revealing `?unlock` is presence-only presentation and never authorization or mutation.

**Why:** The parameter is discoverability, not a credential.

**Enforced in:** [Slice 2](slices/02-browser_maintenance_unlock.md), maintenance render handler.

**Minimum regression tests:** `?unlock`, `?unlock=`, `?unlock=false`, and duplicate values
reveal the form. No reveal request loads identity or calls the stop sequence.

## 3. Invalid input cannot transition startup

**Rule:** Empty, malformed, invalid, and corrupt-identity submissions cannot assign identity or
begin the maintenance stop sequence.

**Why:** Startup must stay locked unless a valid application identity is available.

**Enforced in:** [Slice 1](slices/01-safe_unlock_core.md), shared validation and JSON command.

**Minimum regression tests:** Each failure keeps the existing HTTP 400 command contract, leaves the
identity nil, and records zero graceful-shutdown and forced-close calls.

## 4. Successful transition is prompt, bounded, and exactly once

**Rule:** A valid unlock assigns identity and schedules one stop sequence. The sequence attempts
bounded graceful `Shutdown`, then calls forced `Close` if graceful shutdown returns an error or
reaches its deadline. It has no retry loop. The initiating request receives HTTP 200 before stopping
begins.

**Why:** Concurrent retries must not repeat startup, deadlock on their own request, or leave the
maintenance listener blocking startup after graceful shutdown fails.

**Enforced in:** [Slice 1](slices/01-safe_unlock_core.md), maintenance command transition.

**Minimum regression tests:** Concurrent valid requests pass the race detector and cause one stop
sequence. A real loopback server returns prompt 200 and releases its address. An injected graceful
shutdown failure invokes `Close` once and still permits listener replacement. Both logged errors
omit the secret.

## 5. Plaintext passphrase is confined

**Rule:** The plaintext passphrase exists only in transient input/memory and the unlock POST body.
It never appears in markup, data attributes, request URLs or headers, redirects, logs, response
bodies or headers, browser URLs, or browser history. The field is cleared after every failed or
successful attempt.

**Why:** The startup credential must not leak, persist, or return through navigation state.

**Enforced in:** [Slice 1](slices/01-safe_unlock_core.md) and
[Slice 2](slices/02-browser_maintenance_unlock.md).

**Minimum regression tests:** Use a distinctive sentinel passphrase. Capture Go logs, request URLs,
response bodies, all response headers including `Location`, and rendered HTML/data attributes.
Browser network observation may find the sentinel only in unlock POST data. Assert it is absent from
current and final URLs, and that Back after `location.replace` cannot restore the unlock URL or
secret.

## 6. Normal application access remains locked

**Rule:** Normal routes and account authentication remain unavailable until startup unlock succeeds.
Account credentials never substitute for the application passphrase.

**Why:** Startup lacks normal session infrastructure and uses a distinct credential.

**Enforced in:** [Slice 2](slices/02-browser_maintenance_unlock.md), unchanged maintenance
catch-all.

**Minimum regression tests:** Ordinary application paths return the marked 503 maintenance page
while locked, with no account sign-in workflow.

## 7. Persisted passphrase protection is unchanged

**Rule:** Unlock loads only the runtime identity. The encrypted identity bytes and
`IsIdentityEncryptedWithPassphrase` remain unchanged and apply again after process restart.

**Why:** Unlock is not passphrase removal, replacement, or recovery.

**Enforced in:** [Slice 1](slices/01-safe_unlock_core.md), command path with no configuration
writes.

**Minimum regression tests:** Snapshot both persisted values before successful unlock and compare
them byte-for-byte afterward while confirming that the runtime identity was loaded.

## 8. Ordinary maintenance behavior is preserved

**Rule:** Visitors without `unlock` retain the ordinary 503 page, no credential prompt, and the
60-second refresh. A visible form remains 503 but has no meta refresh.

**Why:** The feature must not expose a credential prompt to visitors or interrupt operator input.

**Enforced in:** [Slice 2](slices/02-browser_maintenance_unlock.md), conditional render.

**Minimum regression tests:** Cover absent and present parameters on root and arbitrary paths,
including status, form visibility, maintenance content, and refresh metadata.

## 9. Browser and CLI paths remain equivalent

**Rule:** Browser and CLI unlocks use the same JSON command and startup transition. The command does
not require `?unlock`, and successful CLI-compatible requests remain HTTP 200.

**Why:** Browser support must not fork startup behavior or regress the operator CLI.

**Enforced in:** [Slice 1](slices/01-safe_unlock_core.md) and
[Slice 2](slices/02-browser_maintenance_unlock.md).

**Minimum regression tests:** A CLI-shaped POST without the reveal parameter succeeds, the CLI
binary builds, and manual wrong-then-correct CLI smoke checks pass against an externally prepared
locked instance without refactoring the interactive CLI for tests.

## 10. Credential responses are protected

**Rule:** Form and command responses are non-cacheable. Unsafe cross-origin browser submissions are
rejected without blocking the existing non-browser CLI request.

**Why:** Credentials and credential-processing responses must not be cached or exposed cross-origin.

**Enforced in:** [Slice 1](slices/01-safe_unlock_core.md) and
[Slice 2](slices/02-browser_maintenance_unlock.md).

**Minimum regression tests:** Assert `Cache-Control: no-store` and `Pragma: no-cache`. Reject a
cross-site browser POST while accepting a CLI-shaped POST.

## 11. Readiness uses the maintenance marker

**Rule:** Polling removes every `unlock` value, preserves path, other query parameters, and
fragment, and waits through network failures and marker-bearing responses. It replaces the location
only when the marker is absent, regardless of response status.

**Why:** Normal routes may legitimately return 503, and listener replacement creates a network gap.

**Enforced in:** [Slice 2](slices/02-browser_maintenance_unlock.md), marker and polling script.

**Minimum regression tests:** Assert the marker on maintenance pages. Browser tests preserve the
cleaned target, wait through marked 503 responses, accept a marker-free 503 as normal, and verify
replacement rather than addition of the unlock history entry.
