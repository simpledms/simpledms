-- reverse: create index "webauthnchallenge_client_key_ceremony" to table: "web_authn_challenges"
DROP INDEX `webauthnchallenge_client_key_ceremony`;
-- reverse: create index "webauthnchallenge_account_id_ceremony" to table: "web_authn_challenges"
DROP INDEX `webauthnchallenge_account_id_ceremony`;
-- reverse: create index "webauthnchallenge_expires_at" to table: "web_authn_challenges"
DROP INDEX `webauthnchallenge_expires_at`;
-- reverse: create index "web_authn_challenges_challenge_id_key" to table: "web_authn_challenges"
DROP INDEX `web_authn_challenges_challenge_id_key`;
-- reverse: create "web_authn_challenges" table
DROP TABLE `web_authn_challenges`;
-- reverse: create index "passkeycredential_account_id" to table: "passkey_credentials"
DROP INDEX `passkeycredential_account_id`;
-- reverse: create index "passkey_credentials_credential_id_key" to table: "passkey_credentials"
DROP INDEX `passkey_credentials_credential_id_key`;
-- reverse: create index "passkey_credentials_public_id_key" to table: "passkey_credentials"
DROP INDEX `passkey_credentials_public_id_key`;
-- reverse: create "passkey_credentials" table
DROP TABLE `passkey_credentials`;
-- reverse: create index "tenants_public_id_key" to table: "tenants"
DROP INDEX `tenants_public_id_key`;
-- reverse: create "new_tenants" table
DROP TABLE `new_tenants`;
-- reverse: create index "account_email" to table: "accounts"
DROP INDEX `account_email`;
-- reverse: create index "accounts_public_id_key" to table: "accounts"
DROP INDEX `accounts_public_id_key`;
-- reverse: create "new_accounts" table
DROP TABLE `new_accounts`;
