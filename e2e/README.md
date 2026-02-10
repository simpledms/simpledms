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

## Environment variables

- `E2E_BASE_URL`: Base URL for the app under test. Default: `https://localhost:7003`
- `E2E_LOGIN_EMAIL`: Login email used by the global setup and auth helpers. Default: `testing+admin@simpledms.app`
- `E2E_LOGIN_PASSWORD`: Login password used by the global setup and auth helpers. Default: `12345678`
- `E2E_ALLOW_STATE_MUTATION`: Set to `1` to run state-mutating tests (for example, successful password/passphrase updates). Default: not enabled

Example:

```bash
E2E_ALLOW_STATE_MUTATION=1 npm run test:e2e
```

## CI

GitHub Actions workflow: `.github/workflows/playwright_e2e.yml`

Set repository secrets for CI login:

- `E2E_LOGIN_EMAIL`
- `E2E_LOGIN_PASSWORD`

## Fixtures

Test upload fixtures live in `e2e/fixtures/`.
