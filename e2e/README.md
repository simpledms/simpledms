# E2E Tests (Playwright)

This repository contains browser end-to-end tests in `e2e/*.spec.ts`.

Current coverage includes auth flows, user/space management, browse/upload flows, search and filters, and owner permission checks.

## Run locally

```bash
npm run test:e2e
```

Run with Playwright UI:

```bash
npm run test:e2e:ui
```

## Important

The user used for testing must have the `admin` role and have English as default language.

## Environment variables

- `E2E_BASE_URL`: Base URL for the app under test. Default: `https://localhost:7003`
- `E2E_LOGIN_EMAIL`: Login email used by the global setup and auth helpers. Default: `dev+admin@simpledms.app`
- `E2E_LOGIN_PASSWORD`: Login password used by the global setup and auth helpers. Default: `12345678`
- `E2E_ALLOW_STATE_MUTATION`: Set to `1` to run state-mutating tests (for example, successful password/passphrase updates). Default: not enabled

Example:

```bash
E2E_ALLOW_STATE_MUTATION=1 npm run test:e2e
```

## Maintenance unlock

The maintenance suite uses its own Playwright configuration and does not run the normal sign-in
setup. Prepare an externally running, locked instance before starting it:

1. Sign in as an administrator and set a distinctive application passphrase from the System
   dashboard.
2. Stop SimpleDMS, then restart it so startup waits on the maintenance screen.
3. Run the dedicated suite with the same passphrase:

```bash
E2E_MAINTENANCE_PASSPHRASE='distinctive-test-passphrase' npm run test:e2e:maintenance
```

Set `E2E_BASE_URL` if the locked instance is not available at `https://localhost:7003`. The final
test submits the valid passphrase and unlocks the instance, so restart it before another run.

## CI

GitHub Actions workflow: `.github/workflows/playwright_e2e.yml`

Set repository secrets for CI login:

- `E2E_LOGIN_EMAIL`
- `E2E_LOGIN_PASSWORD`

## Fixtures

Test upload fixtures live in `e2e/fixtures/`.
