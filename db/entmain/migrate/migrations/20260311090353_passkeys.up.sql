-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_accounts" table
CREATE TABLE `new_accounts` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `created_by` integer NULL, `updated_at` datetime NOT NULL, `updated_by` integer NULL, `deleted_by` integer NULL, `deleted_at` datetime NULL, `public_id` text NOT NULL, `email` text NOT NULL, `first_name` text NOT NULL, `last_name` text NOT NULL, `language` text NOT NULL, `subscribed_to_newsletter_at` datetime NULL, `password_salt` text NOT NULL DEFAULT (''), `password_hash` text NOT NULL DEFAULT (''), `temporary_password_salt` text NOT NULL DEFAULT (''), `temporary_password_hash` text NOT NULL DEFAULT (''), `temporary_password_expires_at` datetime NULL, `temporary_two_factor_auth_key_encrypted` text NOT NULL DEFAULT (''), `two_factory_auth_key_encrypted` text NOT NULL DEFAULT (''), `two_factor_auth_recovery_code_salt` text NOT NULL DEFAULT (''), `two_factor_auth_recovery_code_hashes` json NOT NULL, `last_login_attempt_at` datetime NULL, `passkey_login_enabled` bool NOT NULL DEFAULT (false), `passkey_recovery_code_salt` text NOT NULL DEFAULT (''), `passkey_recovery_code_hashes` json NOT NULL DEFAULT '[]', `role` text NOT NULL);
-- copy rows from old table "accounts" to new temporary table "new_accounts"
INSERT INTO `new_accounts` (`id`, `created_at`, `created_by`, `updated_at`, `updated_by`, `deleted_by`, `deleted_at`, `public_id`, `email`, `first_name`, `last_name`, `language`, `subscribed_to_newsletter_at`, `password_salt`, `password_hash`, `temporary_password_salt`, `temporary_password_hash`, `temporary_password_expires_at`, `temporary_two_factor_auth_key_encrypted`, `two_factory_auth_key_encrypted`, `two_factor_auth_recovery_code_salt`, `two_factor_auth_recovery_code_hashes`, `last_login_attempt_at`, `role`) SELECT `id`, `created_at`, `created_by`, `updated_at`, `updated_by`, `deleted_by`, `deleted_at`, `public_id`, `email`, `first_name`, `last_name`, `language`, `subscribed_to_newsletter_at`, `password_salt`, `password_hash`, `temporary_password_salt`, `temporary_password_hash`, `temporary_password_expires_at`, `temporary_two_factor_auth_key_encrypted`, `two_factory_auth_key_encrypted`, `two_factor_auth_recovery_code_salt`, `two_factor_auth_recovery_code_hashes`, `last_login_attempt_at`, `role` FROM `accounts`;
-- drop "accounts" table after copying rows
DROP TABLE `accounts`;
-- rename temporary table "new_accounts" to "accounts"
ALTER TABLE `new_accounts` RENAME TO `accounts`;
-- create index "accounts_public_id_key" to table: "accounts"
CREATE UNIQUE INDEX `accounts_public_id_key` ON `accounts` (`public_id`);
-- create index "account_email" to table: "accounts"
CREATE UNIQUE INDEX `account_email` ON `accounts` (`email`) WHERE `deleted_at` is null;
-- create "new_tenants" table
CREATE TABLE `new_tenants` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `deleted_at` datetime NULL, `public_id` text NOT NULL, `name` text NOT NULL, `first_name` text NOT NULL DEFAULT (''), `last_name` text NOT NULL DEFAULT (''), `street` text NOT NULL DEFAULT (''), `house_number` text NOT NULL DEFAULT (''), `additional_address_info` text NOT NULL DEFAULT (''), `postal_code` text NOT NULL DEFAULT (''), `city` text NOT NULL DEFAULT (''), `country` text NOT NULL, `plan` text NOT NULL DEFAULT ('Unknown'), `vat_id` text NOT NULL DEFAULT (''), `terms_of_service_accepted` datetime NOT NULL, `privacy_policy_accepted` datetime NOT NULL, `two_factor_auth_enforced` bool NOT NULL DEFAULT (false), `passkey_auth_enforced` bool NOT NULL DEFAULT (false), `x25519_identity_encrypted` blob NULL, `maintenance_mode_enabled_at` datetime NULL, `initialized_at` datetime NULL, `created_by` integer NULL, `updated_by` integer NULL, `deleted_by` integer NULL, CONSTRAINT `tenants_accounts_creator` FOREIGN KEY (`created_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `tenants_accounts_updater` FOREIGN KEY (`updated_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `tenants_accounts_deleter` FOREIGN KEY (`deleted_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION);
-- copy rows from old table "tenants" to new temporary table "new_tenants"
INSERT INTO `new_tenants` (`id`, `created_at`, `updated_at`, `deleted_at`, `public_id`, `name`, `first_name`, `last_name`, `street`, `house_number`, `additional_address_info`, `postal_code`, `city`, `country`, `plan`, `vat_id`, `terms_of_service_accepted`, `privacy_policy_accepted`, `two_factor_auth_enforced`, `x25519_identity_encrypted`, `maintenance_mode_enabled_at`, `initialized_at`, `created_by`, `updated_by`, `deleted_by`) SELECT `id`, `created_at`, `updated_at`, `deleted_at`, `public_id`, `name`, `first_name`, `last_name`, `street`, `house_number`, `additional_address_info`, `postal_code`, `city`, `country`, `plan`, `vat_id`, `terms_of_service_accepted`, `privacy_policy_accepted`, `two_factor_auth_enforced`, `x25519_identity_encrypted`, `maintenance_mode_enabled_at`, `initialized_at`, `created_by`, `updated_by`, `deleted_by` FROM `tenants`;
-- drop "tenants" table after copying rows
DROP TABLE `tenants`;
-- rename temporary table "new_tenants" to "tenants"
ALTER TABLE `new_tenants` RENAME TO `tenants`;
-- create index "tenants_public_id_key" to table: "tenants"
CREATE UNIQUE INDEX `tenants_public_id_key` ON `tenants` (`public_id`);
-- create "passkey_credentials" table
CREATE TABLE `passkey_credentials` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `public_id` text NOT NULL, `credential_id` blob NOT NULL, `credential_json` blob NOT NULL, `name` text NOT NULL DEFAULT (''), `last_used_at` datetime NULL, `created_by` integer NULL, `updated_by` integer NULL, `account_id` integer NOT NULL, CONSTRAINT `passkey_credentials_accounts_creator` FOREIGN KEY (`created_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `passkey_credentials_accounts_updater` FOREIGN KEY (`updated_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `passkey_credentials_accounts_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION);
-- create index "passkey_credentials_public_id_key" to table: "passkey_credentials"
CREATE UNIQUE INDEX `passkey_credentials_public_id_key` ON `passkey_credentials` (`public_id`);
-- create index "passkey_credentials_credential_id_key" to table: "passkey_credentials"
CREATE UNIQUE INDEX `passkey_credentials_credential_id_key` ON `passkey_credentials` (`credential_id`);
-- create index "passkeycredential_account_id" to table: "passkey_credentials"
CREATE INDEX `passkeycredential_account_id` ON `passkey_credentials` (`account_id`);
-- create "web_authn_challenges" table
CREATE TABLE `web_authn_challenges` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `challenge_id` text NOT NULL, `client_key` text NULL, `ceremony` text NOT NULL, `session_data_json` blob NOT NULL, `expires_at` datetime NOT NULL, `used_at` datetime NULL, `created_at` datetime NOT NULL, `account_id` integer NULL, CONSTRAINT `web_authn_challenges_accounts_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON DELETE SET NULL);
-- create index "web_authn_challenges_challenge_id_key" to table: "web_authn_challenges"
CREATE UNIQUE INDEX `web_authn_challenges_challenge_id_key` ON `web_authn_challenges` (`challenge_id`);
-- create index "webauthnchallenge_expires_at" to table: "web_authn_challenges"
CREATE INDEX `webauthnchallenge_expires_at` ON `web_authn_challenges` (`expires_at`);
-- create index "webauthnchallenge_account_id_ceremony" to table: "web_authn_challenges"
CREATE INDEX `webauthnchallenge_account_id_ceremony` ON `web_authn_challenges` (`account_id`, `ceremony`);
-- create index "webauthnchallenge_client_key_ceremony" to table: "web_authn_challenges"
CREATE INDEX `webauthnchallenge_client_key_ceremony` ON `web_authn_challenges` (`client_key`, `ceremony`);
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
