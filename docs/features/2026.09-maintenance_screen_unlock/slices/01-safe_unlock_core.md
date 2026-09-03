# Slice 01 — Safe Unlock Core

## Outcome

The existing JSON unlock command and CLI path reject every invalid identity, return HTTP 200
without waiting on their own server shutdown, and perform one race-free startup transition.
If bounded graceful shutdown fails, forced close releases the maintenance listener for startup.

## Scope

In scope:

- Fix the shared identity-decryption contract in
  `model/main/systemconfig/system_config.go` and cover its model caller.
- Harden the maintenance command and runtime handler chain in `server/server.go` without changing
  its method, content-type, body-size, URL, JSON body, or successful status contract.
- Add focused model and server tests for secrecy, cross-origin protection, concurrency, shutdown
  failure, persisted protection, loopback listeners, and listener replacement.
- Keep `cmd/unlocker/main.go` interactive and behavior-compatible. Do not refactor it solely to make
  automated testing easier.

Out of scope:

- Browser form rendering, translations, and browser polling, which belong to Slice 2.
- Rate limiting and stricter request-contract rules left open by the specification.
- Schema, Ent, migrations, account auth, tenant maintenance, and passphrase management.

## Acceptance Criteria

1. `DecryptMainIdentity` returns an error, never `(nil, nil)`, when decrypted plaintext is not a
   valid X25519 identity.
2. Wrong, empty, malformed, or corrupt submissions assign no identity and invoke neither graceful
   shutdown nor forced close.
3. Concurrent valid requests assign one identity and schedule exactly one listener stop sequence.
4. A successful request receives prompt HTTP 200 before the asynchronous stop sequence begins.
5. The stop sequence uses a bounded graceful `Shutdown`. If it returns an error or deadline, it
   calls forced `Close` once. Both errors are logged without secrets, and no retry loop is added.
6. The JSON command remains available at `/-/unlock-cmd` without `?unlock`, preserving the existing
   CLI request, response-body, and successful HTTP 200 contracts.
7. Every command response sends `Cache-Control: no-store` and `Pragma: no-cache`.
8. `http.NewCrossOriginProtection` rejects unsafe cross-site browser requests on the maintenance
   runtime path while allowing requests without browser cross-origin headers.
9. A real HTTP maintenance listener returns prompt 200, stops, releases its address, and allows a
   replacement listener to respond. The forced-close failure path provides the same result.
10. Certificate TLS receives equivalent automated loopback coverage using a generated test
    certificate. Autocert keeps the shared handler and listen-mode wiring without unit ACME calls.
11. A successful runtime unlock does not modify the persisted encrypted identity bytes or
    `IsIdentityEncryptedWithPassphrase`.
12. CLI-shaped handler tests, `go build ./cmd/unlocker`, and manual wrong-then-correct CLI smoke
    checks pass without refactoring the interactive CLI for tests.

## Dependencies

No feature-slice dependency. This slice uses the existing age helper, maintenance listener,
standard-library HTTP protection and synchronization, and current test helpers. It provides the
safe command and transition required by Slice 2.

## Detailed Implementation Checklist

- [x] Add model regression coverage in
  `model/main/systemconfig/system_config_test.go`. Encrypt non-X25519 plaintext with a valid
  passphrase, then assert the helper and `SystemConfig` caller return an error and leave the global
  identity unset.
- [x] Change `DecryptMainIdentity` to propagate the X25519 parse failure instead of returning
  `(nil, nil)`. Keep existing error propagation and never log passphrase material.
- [x] Make the production maintenance stop sequence able to call both the maintenance
  `http.Server.Shutdown` and `http.Server.Close`. Do not add a test-only production seam.
- [x] Guard successful identity assignment and scheduling of the complete stop sequence with the
  smallest per-handler exactly-once synchronization.
- [x] Write HTTP 200 before scheduling the stop sequence in a goroutine. Use the existing bounded
  graceful-shutdown context. If `Shutdown` fails or reaches its deadline, call `Close` once.
- [x] Log graceful-shutdown and forced-close errors independently, without request bodies,
  passphrases, or derived secret values. Do not add retry state after forced close.
- [x] Set no-store and no-cache headers before every JSON command response, including all error
  paths, without changing request or response bodies.
- [x] Extend `server/server_maintenance_mode_test.go` so malformed JSON, empty passphrase, wrong
  passphrase, and malformed decrypted identity all keep the existing 400 behavior with no
  assignment or stop call.
- [x] Add concurrent valid-submission coverage with a synchronized start, stable loaded identity,
  and atomic shutdown/close counts. Assert one stop sequence under the race detector.
- [x] Add a shutdown-failure test where graceful shutdown returns a deadline/error. Assert prompt
  200, one forced close, secret-free logs, listener release, and successful listener replacement.
- [x] Use a distinctive sentinel passphrase in secrecy tests. Capture Go logs and inspect the
  request URL, response body, and every response header, including `Location`. Permit the sentinel
  only in the unlock POST body and transient test input.
- [x] Snapshot the persisted encrypted identity bytes and
  `IsIdentityEncryptedWithPassphrase` before a successful command. Assert both are unchanged
  byte-for-byte afterward while the runtime identity is loaded.
- [x] Put `http.NewCrossOriginProtection` on the maintenance runtime handler path, matching the
  normal server's standard-library protection without adding middleware or dependencies.
- [x] Test a CLI-shaped POST without `?unlock` and browser cross-origin headers. Assert HTTP 200 and
  the existing response contract. Test that an unsafe cross-site browser POST cannot transition.
- [x] Add a real `127.0.0.1:0` HTTP loopback test for prompt response, graceful exit, address
  rebind, and replacement response.
- [x] Add the same automated lifecycle test for certificate TLS using a generated temporary test
  certificate and key. Do not depend on repository or external certificates.
- [x] Preserve the pure listen-mode tests in `server/server_start_listen_flow_test.go`. Verify that
  HTTP, certificate TLS, and autocert still select the same maintenance handler and transition.
- [x] Keep autocert unit coverage limited to mode and shared-wiring assertions. Never contact an
  external ACME service from tests.
- [x] Leave `cmd/unlocker/main.go` unchanged unless compatibility is broken. Do not extract its
  interactive input solely for automated testing.

## Verification Checklist

### Listener verification matrix

- [x] HTTP: run the automated real-loopback prompt response, graceful stop, forced-close failure,
  rebind, and replacement-response cases.
- [x] Certificate TLS: run the same automated lifecycle with a generated test certificate.
- [x] Autocert: run pure mode/shared-handler wiring tests without ACME. When an autocert deployment
  is available, manually reveal the maintenance page and complete a browser unlock on its listener.

### Automated checks

- [x] Run focused model tests for `DecryptMainIdentity`, caller failure, and persisted protection.
- [x] Run focused command, cross-origin, secrecy, concurrency, shutdown, and listener tests.
- [x] Run `go test -race ./server -run 'Maintenance|Unlock'`.
- [x] Run `go test ./...`.
- [x] Run `go build ./...`.
- [x] Run `go build ./cmd/unlocker`.
- [x] Run
  `go build -tags "sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_icu" ./...`.
- [x] Run `go vet ./...`.

### Manual CLI smoke

- [x] Prepare a real locked instance with a known distinctive sentinel passphrase.
- [x] Run `cmd/unlocker` with a wrong passphrase. Confirm its error behavior, that the instance
  stays locked, and that captured server logs contain neither submitted passphrase.
- [x] Run `cmd/unlocker` with the correct sentinel passphrase. Confirm HTTP success, normal startup,
  and listener replacement. Confirm the sentinel appears only in the POST body/transient input.
- [x] Run `git diff --check` and inspect for accidental request-contract, schema, migration,
  dependency, generated-file, or interactive CLI changes.
