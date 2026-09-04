# Maintenance Screen Unlock

Status: Proposed

Date: 2026-09-03

## Outcome

An operator who knows the application passphrase can unlock SimpleDMS from a
browser when startup is blocked by the application lock. The ordinary
maintenance screen remains unchanged for visitors who do not deliberately ask
to see the unlock form.

This removes the command-line unlocker as a prerequisite for routine startup
without turning the maintenance screen into an account sign-in screen or a
general administration interface.

## Existing Behaviour

- When the persisted application identity is passphrase-protected, startup
  serves only the maintenance handler until the identity is decrypted.
- The maintenance page returns `503 Service Unavailable`, displays a waiting
  message, and refreshes every 60 seconds.
- `/-/unlock-cmd` already accepts the application passphrase in a JSON request.
  A valid passphrase loads the identity and allows normal startup to continue.
- `cmd/unlocker` is the existing operator interface to that unlock capability.
- Normal routes, sessions, and administrator actions are unavailable while the
  startup maintenance handler is active.
- Account passwords and the application passphrase are separate credentials.

Relevant existing behaviour is defined in `server/server.go`,
`server/server_maintenance_mode_test.go`, and
`model/main/systemconfig/system_config.go`.

## Users

| User | Need |
| --- | --- |
| Authorized operator | Reveal the form, enter the application passphrase, and continue startup. |
| Ordinary visitor | See the existing maintenance message without being prompted for a credential. |
| CLI operator | Continue using the existing command-line unlock workflow without regression. |

Possession of the application passphrase authorizes this startup operation.
Account authentication cannot be required because the account and session
infrastructure is not available before the application identity is loaded.

## Workflows

### View Ordinary Maintenance Screen

1. A visitor requests an application URL while SimpleDMS is locked during
   startup.
2. The URL does not contain the `unlock` query parameter.
3. The visitor sees the existing maintenance heading and waiting message.
4. No credential field or unlock action is rendered.

### Reveal And Submit The Unlock Form

1. An operator appends the presence-only `unlock` query parameter to the
   maintenance URL, for example `/?unlock`.
2. The maintenance screen additionally shows a required password field labelled
   `Application passphrase` and an `Unlock application` action.
3. The operator submits the passphrase without placing it in the URL.
4. The form becomes busy and prevents duplicate submissions while the request
   is pending, without showing a separate interim status message.

### Handle An Invalid Passphrase

1. The application rejects an empty or invalid passphrase.
2. The application remains locked and the normal application does not start.
3. The form remains visible and displays an accessible, user-facing error.
4. The submitted passphrase is not redisplayed or retained in rendered output.
5. The operator can enter another passphrase and retry.

### Continue After A Valid Passphrase

1. The valid passphrase decrypts and loads the application identity.
2. The maintenance server yields to normal application startup.
3. The browser automatically returns to the originally requested application
   path after the normal application becomes available.
4. The `unlock` query parameter is removed from the resulting application URL;
   unrelated query parameters are preserved.
5. The persisted passphrase protection remains enabled, so a later process
   restart requires the application passphrase again.

## Requirements

### Reveal Parameter

1. The query parameter name must be `unlock`.
2. Parameter presence alone must reveal the form. `?unlock`, `?unlock=`, and
   `?unlock=false` therefore have the same reveal behaviour.
3. The parameter value must never be interpreted as or used to carry the
   application passphrase.
4. The parameter must only alter maintenance-screen presentation. It must not
   unlock the application, authenticate a user, or grant access to normal
   application routes.
5. The operator must be able to add the parameter to whichever application URL
   currently renders the maintenance screen.

### Maintenance Screen

1. Without the `unlock` parameter, the current maintenance content and lack of
   an unlock form must be preserved.
2. With the `unlock` parameter, the same maintenance context and status must
   remain visible alongside the unlock form.
3. A maintenance-page response must remain `503 Service Unavailable`, whether
   or not the form is revealed.
4. Automatic maintenance-page refresh must not interrupt passphrase entry,
   clear an in-progress form, or cause duplicate submission.
5. The form, labels, actions, progress state, and errors must be available in
   every language supported by the existing maintenance screen.

### Credential Entry

1. The credential must be named `Application passphrase` in user-facing text.
2. The field must use password-entry semantics so its value is visually
   obscured.
3. The field must be required and support keyboard submission.
4. The passphrase must be sent only in a request body and must never appear in a
   URL, redirect location, browser history entry, or rendered page.
5. Empty input must be rejected without changing application state.
6. Invalid input must produce a clear error without disclosing encrypted
   identity details or other internal errors.
7. A failed attempt must leave the form available for another attempt.

### Successful Unlock

1. Only successful decryption of a valid application identity may unlock the
   application.
2. A successful browser unlock must trigger the same application state
   transition as the existing unlock command: load the identity, end startup
   maintenance mode, and continue normal startup.
3. The browser must communicate that startup is continuing rather than showing
   an unexplained failed request while the maintenance listener transitions.
4. The browser must load the normal application automatically when it is ready;
   the operator must not need to manually refresh.
5. Repeated submission or browser retries must not cause multiple conflicting
   startup transitions.
6. Unlocking must not remove, replace, or weaken the configured passphrase
   protection.

### Compatibility

1. The existing `cmd/unlocker` workflow must remain supported.
2. Existing clients of the JSON unlock command must not need to supply the
   reveal query parameter.
3. HTTP, configured TLS certificates, and automatic TLS must retain their
   existing maintenance-mode behaviour.
4. The feature must not require access to normal application routing,
   authenticated sessions, tenant data, or encrypted runtime configuration
   before unlock succeeds.

### Security And Privacy

1. The `unlock` parameter is a discoverability control, not a security control.
2. The application passphrase must not be logged, included in analytics, or
   exposed through error details.
3. Responses that render or process the passphrase form must not be stored by
   shared or browser caches.
4. Unlock submission must be limited to same-origin browser interaction and
   must not make the passphrase available to cross-origin callers.
5. Invalid submissions must not change the loaded identity, stop the
   maintenance server, or expose whether any account exists.
6. A successfully submitted passphrase must not be retained by the maintenance
   page after the unlock attempt completes.

### Accessibility

1. The passphrase field must have a persistent programmatic label.
2. Validation and unlock errors must be programmatically associated with the
   field or announced as status messages.
3. Progress and disabled states must be conveyed without relying only on color.
4. Focus order, keyboard operation, and visible focus treatment must follow the
   existing application accessibility conventions.

## Non-Goals

- Accepting an account password, passkey, recovery code, or administrator
  session in place of the application passphrase.
- Changing, removing, resetting, or recovering a forgotten application
  passphrase.
- Locking an already running application.
- Exposing the application passphrase in a query parameter.
- Treating knowledge of the `?unlock` URL as authorization.
- Building a general maintenance administration page or status dashboard.
- Changing tenant migration maintenance mode.
- Replacing or removing `cmd/unlocker`.
- Redesigning normal account sign-in or the authenticated System dashboard.

## Acceptance Criteria

1. Given a locked application, when a visitor opens `/`, then the response is a
   maintenance page with status 503 and no passphrase field.
2. Given a locked application, when an operator opens `/?unlock`, then the same
   maintenance page includes a required, obscured `Application passphrase`
   field and an `Unlock application` action.
3. Given any value for the `unlock` parameter, when the maintenance page is
   rendered, then the form is visible because presence, not value, controls
   visibility.
4. Given no `unlock` parameter, when any application path falls back to the
   maintenance page, then no unlock form or credential prompt is rendered.
5. Given an empty submission, when the operator submits the form, then the page
   reports that the passphrase is required and the application remains locked.
6. Given an incorrect passphrase, when the operator submits the form, then the
   page reports an invalid passphrase, does not redisplay the submitted value,
   and the application remains locked.
7. Given the correct passphrase, when the operator submits the form, then the
   application identity is loaded exactly once and normal startup continues.
8. Given a successful unlock, when the normal application becomes available,
   then the browser automatically loads the originally requested path without
   the `unlock` parameter.
9. Given a revealed form, when the operator is entering a passphrase, then the
   existing maintenance refresh behavior does not erase the input or interrupt
   submission.
10. Given any unlock attempt, when requests, responses, logs, redirects, and
    rendered output are inspected, then the plaintext passphrase appears only
    in the submitted request body and transient processing memory.
11. Given each supported maintenance-screen language, when the form is revealed
    and validation fails, then the form controls and user-facing error are
    localized.
12. Given the feature is available, when an operator uses `cmd/unlocker`, then
    the existing successful and unsuccessful CLI workflows still behave as
    before.
13. Given HTTP, certificate-based TLS, or automatic TLS configuration, when the
    application starts locked, then the browser unlock workflow is available on
    the same configured listener.
14. Given an unlock attempt, when operated with only a keyboard or assistive
    technology, then the field, action, progress, and error state are usable and
    understandable.

## Invariants

1. The application remains locked until the stored identity is successfully
   decrypted into a valid application identity.
2. Revealing the form does not mutate state and does not prove authorization.
3. An empty, malformed, or invalid submission cannot load an identity or begin
   the transition to the normal application.
4. The plaintext application passphrase never appears in a URL, log entry,
   redirect, rendered response, or persistent application storage.
5. Normal application routes and account authentication remain unavailable
   until startup unlock succeeds.
6. Application-passphrase protection remains configured after a successful
   unlock and applies again after the next process restart.
7. Visitors who do not supply `?unlock` continue to see only the ordinary
   maintenance experience.
8. Account credentials never substitute for the application passphrase.
9. Browser unlocking and CLI unlocking result in the same valid loaded identity
   and startup state.

## Assumptions

- "Password field" means an HTML password-style field that accepts the existing
  application passphrase, not a user account password.
- "Application is locked" means startup is waiting for decryption of the
  passphrase-protected application identity, not tenant migration maintenance
  mode or an account lockout.
- The user-confirmed reveal contract is presence-only `?unlock`.
- An operator may start from any application path currently showing the
  maintenance screen and should return to that path after unlock.
- The existing maintenance listener and application passphrase remain the
  sources of truth; this feature adds no second credential or unlock state.
- Existing supported maintenance languages are English, German, French, and
  Italian.
- Hiding the form by default is intended to avoid presenting an administrative
  credential prompt to ordinary visitors, not to conceal the unlock capability
  from an attacker.

## Open Questions

1. Should failed maintenance unlock attempts be rate-limited, and if so, what
   retry and recovery policy should apply to both browser and CLI operators?
2. Should the existing unlock command be restricted to `POST`, an expected
   content type, and a bounded request size as part of this feature, or should
   that endpoint hardening be specified separately?
3. Should operations documentation advertise `?unlock` as the preferred
   workflow, or retain `cmd/unlocker` as the primary documented recovery path?
