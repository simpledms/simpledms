# Slice 02 — Browser Maintenance Unlock

## Outcome

An operator can reveal a localized maintenance unlock form, submit the application passphrase with
same-origin `fetch`, and return to the cleaned original URL when the normal server is ready.
Ordinary visitors retain the current maintenance page.

## Scope

In scope:

- Conditionally compose the maintenance page in `server/server.go` from presence-only `unlock`
  semantics while retaining the existing Base, heading, message, layouts, and 503 status.
- Add `MaintenanceUnlockForm` in `core/ui/widget/maintenance_unlock_form.go` with its matching
  `.gohtml` template. Reuse TextField, button, and layout conventions where practical.
- Keep accessibility and behavior maintenance-specific. Do not broaden shared `Form`, `TextField`,
  `Button`, or `FormHelper` APIs for this one form.
- Implement native form behavior with same-origin `fetch` and explicitly opt out of inherited HTMX.
- Add marker, cache, render, localization, secrecy, Playwright, and E2E documentation coverage.

Out of scope:

- A no-JavaScript workflow, new frontend dependency, normal dashboard/sign-in changes, account
  authorization, tenant maintenance, passphrase management, or test-only production setup seams.
- Rate limiting, stricter endpoint request contracts, and preferred operations guidance pending the
  specification's open questions.

## Acceptance Criteria

1. Without `unlock`, every locked application path returns the existing 503 maintenance content,
   no credential form, and the 60-second meta refresh.
2. `req.URL.Query().Has("unlock")` controls visibility. Empty, false, arbitrary, and duplicate
   values reveal the form without mutating state or gating the JSON endpoint.
3. A revealed form remains 503, includes the same maintenance context, sends no-store/no-cache
   headers, and disables the 60-second meta refresh.
4. The form has a persistent localized `Application passphrase` password label, required semantics,
   keyboard submission, and a localized `Unlock application` button.
5. Submission uses only a relative same-origin `fetch` JSON body to `/-/unlock-cmd`, not HTMX, a
   URL, redirect, or normal `FormHelper` flow.
6. The UI blocks duplicate submission, exposes progress/startup with `role=status`, and exposes
   localized required/invalid feedback with `role=alert`. It associates errors with the field and
   clears the field after every failed or successful attempt.
7. Maintenance pages carry `X-SimpleDMS-Maintenance: true`. Polling uses the marker rather than
   status because a normal route may itself return 503.
8. After command success, the browser polls the cleaned URL through network failures and marked
   maintenance responses at one fixed short delay. It uses no terminal timeout or retry state.
9. URL cleanup removes every `unlock` value while preserving path, unrelated query parameters, and
   fragment. `location.replace` removes the unlock page from browser history.
10. A distinctive sentinel passphrase appears only in transient field input and the unlock POST
    body. It is absent from rendered HTML/data attributes, request URLs/headers, response
    bodies/headers including `Location`, current/final URLs, and restored browser history.
11. English source strings use `widget.T`. English, German, French, and Italian labels and
    messages render according to `Accept-Language`; manual translations remain fuzzy with
    translator comments.
12. A dedicated Playwright configuration runs only serial maintenance tests with one worker, no
    normal sign-in setup, and an externally prepared locked instance.

## Dependencies

Depends on [Slice 01 — Safe unlock core](01-safe_unlock_core.md) for prompt HTTP 200, the bounded
exactly-once stop sequence, shared validation, command cache policy, and cross-origin protection.
The existing Base remains the page shell, but maintenance submission and polling do not use HTMX.

## Detailed Implementation Checklist

- [ ] Add `MaintenanceUnlockForm` and its matching template in
  `core/ui/widget/maintenance_unlock_form.go` and
  `core/ui/widget/maintenance_unlock_form.gohtml`. Keep one struct in the snake-case Go file and
  expose only the labels, messages, IDs, and children required by this page.
- [ ] Model the password field and submit action on existing TextField/button visual and focus
  conventions. Keep maintenance-specific accessibility and JavaScript in the focused template
  instead of expanding shared widgets.
- [ ] Use `widget.T` for `Application passphrase`, `Unlock application`, and a concise startup
  status. Reuse `Passphrase is required.`, `Invalid passphrase.`, and existing generic retry text.
- [ ] In the maintenance handler, use `req.URL.Query().Has("unlock")` and preserve current content
  and status. Append the form only when present and enable 60-second refresh only when absent.
- [ ] Set `X-SimpleDMS-Maintenance: true` on every catch-all maintenance page. Set
  `Cache-Control: no-store` and `Pragma: no-cache` on revealed-form responses before writing 503.
- [ ] Mark the form as not HTMX-boosted and attach its local submit handler without changing Base's
  global HTMX behavior or adding a library.
- [ ] On submit, prevent navigation and show inline localized required feedback for empty input.
  Disable repeat submission, expose a non-color pending state, and send the passphrase only in a
  relative same-origin JSON POST body.
- [ ] Map command 400 to localized invalid feedback and unexpected failures to existing generic
  retry feedback. Never display response internals. Re-enable retry and restore useful focus after
  failure.
- [ ] Clear the password field after every failed or successful request. On HTTP 200, keep
  submission disabled, announce startup status, and begin polling without retaining the passphrase.
- [ ] Build the polling target from the current URL. Delete the `unlock` search parameter so every
  value is removed, while preserving path, unrelated parameters, and fragment.
- [ ] Poll the cleaned same-origin URL with caching disabled. Retry after network errors and marked
  responses using one fixed short delay, no terminal timeout, and no indefinite server-side state.
- [ ] When a response lacks the marker, call `location.replace` with the cleaned URL regardless of
  status. A marker-free normal 503 therefore completes readiness.
- [ ] Expand `server/server_maintenance_mode_test.go` with table-driven render cases for absent,
  empty, false, arbitrary, and duplicate `unlock` values on root and arbitrary paths.
- [ ] Assert 503, ordinary content, form visibility, refresh behavior, marker/cache headers, labels,
  password/required semantics, role relationships, and absence of secret-bearing values.
- [ ] Add an `Accept-Language` table for `en`, `de`, `fr`, and `it`. Assert each localized form
  label, button, required message, invalid message, generic retry message, and startup status for
  that locale.
- [ ] Use a distinctive sentinel in Go render tests. Capture logs and inspect request URL, response
  body, all response headers including `Location`, and rendered HTML/data attributes. The sentinel
  must appear nowhere in these surfaces.
- [ ] Add source entries through normal i18n extraction. Manually edit only
  `i18n/locales/de/messages.gotext.json`, `i18n/locales/fr/messages.gotext.json`, and
  `i18n/locales/it/messages.gotext.json`. Set new entries to `fuzzy: true` with a
  `Translated from English by Codex` comment.
- [ ] Never hand-edit `out.gotext.json` or `*.gen.go`. Generate them through the repository process
  and inspect the missing-translation output for all four maintenance languages.
- [ ] Add `playwright.maintenance.config.ts` with no global setup, only the maintenance spec, the
  existing base URL/TLS defaults, one worker, and serial execution.
- [ ] Add a `test:e2e:maintenance` package script and a serial maintenance spec using semantic
  locators. Cover ordinary/revealed pages, keyboard submission, localized required/invalid feedback,
  duplicate prevention, field clearing, startup status, successful unlock, and final navigation.
- [ ] In Playwright, use a distinctive configured sentinel as the correct passphrase and another
  distinctive value for failure. Observe every browser request and response. Permit each sentinel
  only in its unlock POST data and transient field input.
- [ ] Assert sentinels are absent from request URLs/headers, response bodies/headers including
  `Location`, rendered HTML/data attributes, and current/final URLs. Verify the field is empty after
  both attempts.
- [ ] Establish a prior browser history entry before opening the unlock URL. After
  `location.replace`, use Back and assert the unlock-bearing URL and sentinel cannot be restored.
- [ ] Update `e2e/README.md` with the dedicated command, required
  `E2E_MAINTENANCE_PASSPHRASE`, and steps to prepare and restart a passphrase-protected locked
  instance. State that the final successful test unlocks the instance.

## Verification Checklist

- [ ] Run focused maintenance render/command tests for query variants, headers, secrecy, roles, and
  form semantics.
- [ ] Run the `Accept-Language` table for English, German, French, and Italian labels/messages.
- [ ] Run `go test -race ./server -run 'Maintenance|Unlock'`.
- [ ] Run `go test ./...`.
- [ ] Run `go build ./...`.
- [ ] Run
  `go build -tags "sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_icu" ./...`.
- [ ] Run `go vet ./...`.
- [ ] Run `go generate ./i18n 2> /tmp/simpledms-missing-translations.txt`.
- [ ] Inspect generated diffs and confirm no missing German, French, or Italian entries. English
  missing-entry diagnostics may be ignored according to repository guidance.
- [ ] Prepare the locked instance with a distinctive sentinel and run
  `E2E_MAINTENANCE_PASSPHRASE=... npm run test:e2e:maintenance`.
- [ ] Inspect Playwright network observations and browser history assertions for sentinel leakage.
- [ ] Run `git diff --check` and confirm there are no schema, migration, dependency, dashboard,
  test-only production-seam, or manually edited generated-file changes.
