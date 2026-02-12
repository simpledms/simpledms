-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_tenants" table
CREATE TABLE `new_tenants` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `deleted_at` datetime NULL, `public_id` text NOT NULL, `name` text NOT NULL, `first_name` text NOT NULL DEFAULT (''), `last_name` text NOT NULL DEFAULT (''), `street` text NOT NULL DEFAULT (''), `house_number` text NOT NULL DEFAULT (''), `additional_address_info` text NOT NULL DEFAULT (''), `postal_code` text NOT NULL DEFAULT (''), `city` text NOT NULL DEFAULT (''), `country` text NOT NULL, `plan` text NOT NULL DEFAULT ('Unknown'), `vat_id` text NOT NULL DEFAULT (''), `terms_of_service_accepted` datetime NOT NULL, `privacy_policy_accepted` datetime NOT NULL, `two_factor_auth_enforced` bool NOT NULL DEFAULT (false), `x25519_identity_encrypted` blob NULL, `maintenance_mode_enabled_at` datetime NULL, `initialized_at` datetime NULL, `created_by` integer NULL, `updated_by` integer NULL, `deleted_by` integer NULL, CONSTRAINT `tenants_accounts_creator` FOREIGN KEY (`created_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `tenants_accounts_updater` FOREIGN KEY (`updated_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `tenants_accounts_deleter` FOREIGN KEY (`deleted_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION);
-- copy rows from old table "tenants" to new temporary table "new_tenants"
INSERT INTO `new_tenants` (`id`, `created_at`, `updated_at`, `deleted_at`, `public_id`, `name`, `first_name`, `last_name`, `street`, `house_number`, `additional_address_info`, `postal_code`, `city`, `country`, `vat_id`, `terms_of_service_accepted`, `privacy_policy_accepted`, `two_factor_auth_enforced`, `x25519_identity_encrypted`, `maintenance_mode_enabled_at`, `initialized_at`, `created_by`, `updated_by`, `deleted_by`) SELECT `id`, `created_at`, `updated_at`, `deleted_at`, `public_id`, `name`, `first_name`, `last_name`, `street`, `house_number`, `additional_address_info`, `postal_code`, `city`, `country`, `vat_id`, `terms_of_service_accepted`, `privacy_policy_accepted`, `two_factor_auth_enforced`, `x25519_identity_encrypted`, `maintenance_mode_enabled_at`, `initialized_at`, `created_by`, `updated_by`, `deleted_by` FROM `tenants`;
-- drop "tenants" table after copying rows
DROP TABLE `tenants`;
-- rename temporary table "new_tenants" to "tenants"
ALTER TABLE `new_tenants` RENAME TO `tenants`;
-- create index "tenants_public_id_key" to table: "tenants"
CREATE UNIQUE INDEX `tenants_public_id_key` ON `tenants` (`public_id`);
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
