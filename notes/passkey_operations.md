# Passkey Operations

## Canonical origin and RP setup

Configure these variables for each deployment:

- `SIMPLEDMS_PUBLIC_ORIGIN`
- `SIMPLEDMS_WEBAUTHN_RP_ID` (optional)
- `SIMPLEDMS_WEBAUTHN_RP_NAME` (optional)

Recommended setup:

- Use one canonical origin per region deployment.
- Keep RP IDs region-pinned (for example `app.simpledms.eu` and `app.simpledms.ch`).
- Keep `simpledms.app` as discovery/redirect entry point and not as WebAuthn realm.

If `SIMPLEDMS_WEBAUTHN_RP_ID` is empty, the RP ID is derived from the host in `SIMPLEDMS_PUBLIC_ORIGIN`.

## Passkey behavior overview

- The login page keeps email/password and adds `Sign in with passkey`.
- Accounts with passkey requirement cannot use password sign-in.
- Requirement is strictest-wins across active tenant memberships.
- Passkey registration is available only for authenticated sessions.
- Recovery uses one-time recovery codes and admin-assisted recovery.

## Admin-assisted recovery runbook

Goal: issue a new recovery-code set for a locked out user who has passkeys configured.

1. Verify the operator has `Admin` or `Supporter` role.
2. Verify account ownership and identity using your support process.
3. Submit `/-/auth/admin-passkey-recovery-cmd` with the target account email.
4. Communicate codes over an approved secure channel.
5. Ask the user to sign in via recovery code and register a fresh passkey.

Audit signal:

- The server writes an audit log line:
  `admin assisted passkey recovery actor_account_id=<id> target_account_id=<id>`

## Deployment checklist

- Set canonical origin and RP variables.
- Verify canonical host redirects are active for `/-/auth/*` and `/`.
- Verify passkey registration in account settings on the canonical host.
- Verify passkey sign-in and recovery code sign-in.
- Verify admin-assisted recovery with a test account.
