# Maintenance Screen Unlock Plan

Source: [feature specification](spec.md)

## Approach

Keep the existing JSON command and maintenance listener as the single unlock path. First make
identity validation and the listener transition safe for CLI and browser callers. Then add one
maintenance-only, localized `fetch` form to the existing page, with marker-based readiness polling.
Reuse existing visual conventions without extending shared form infrastructure or adding a
dependency.

## Slice Order

- [x] [01 — Safe unlock core](slices/01-safe_unlock_core.md)
- [ ] [02 — Browser maintenance unlock](slices/02-browser_maintenance_unlock.md)

## Dependencies

Slice 1 has no feature dependency. Slice 2 depends on Slice 1's prompt, exactly-once transition and
protected JSON command. No schema, Ent, migration, tenant, account-authentication, or new-dependency
work is required.

## Assumptions and Deferrals

- Rate limiting, stricter method/content-type/body-size rules, and preferred operations
  documentation remain deferred until the specification's open questions are resolved.
- JavaScript is acceptable because the existing Base already requires it; no no-JavaScript unlock
  workflow is required.
- After success, polling continues until a non-maintenance response is detected or the page closes.
  It uses a short fixed retry delay and has no terminal timeout or configuration.
