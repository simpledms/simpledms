-- add column "upload_last_progress_at" to table: "temporary_files"
ALTER TABLE `temporary_files` ADD COLUMN `upload_last_progress_at` datetime NULL;
-- add column "source" to table: "temporary_files"
ALTER TABLE `temporary_files` ADD COLUMN `source` text NOT NULL DEFAULT 'UnknownLegacy';
-- add column "content_sha256" to table: "temporary_files"
ALTER TABLE `temporary_files` ADD COLUMN `content_sha256` text NULL;
-- add column "storage_crc32c" to table: "temporary_files"
ALTER TABLE `temporary_files` ADD COLUMN `storage_crc32c` text NULL;
-- add column "persistence_claim_token" to table: "temporary_files"
ALTER TABLE `temporary_files` ADD COLUMN `persistence_claim_token` text NULL;
-- add column "persistence_tenant_id" to table: "temporary_files"
ALTER TABLE `temporary_files` ADD COLUMN `persistence_tenant_id` integer NULL;
-- add column "persistence_last_progress_at" to table: "temporary_files"
ALTER TABLE `temporary_files` ADD COLUMN `persistence_last_progress_at` datetime NULL;
-- create "web_dav_credentials" table
CREATE TABLE `web_dav_credentials` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `public_id` text NOT NULL, `space_public_id` text NOT NULL, `label` text NOT NULL, `username` text NOT NULL, `secret_salt` text NOT NULL, `secret_hash` text NOT NULL, `last_used_at` datetime NULL, `revoked_at` datetime NULL, `created_by` integer NULL, `updated_by` integer NULL, `account_id` integer NOT NULL, `tenant_id` integer NOT NULL, `revoked_by_account_id` integer NULL, CONSTRAINT `web_dav_credentials_accounts_creator` FOREIGN KEY (`created_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `web_dav_credentials_accounts_updater` FOREIGN KEY (`updated_by`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `web_dav_credentials_accounts_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION, CONSTRAINT `web_dav_credentials_tenants_tenant` FOREIGN KEY (`tenant_id`) REFERENCES `tenants` (`id`) ON DELETE NO ACTION, CONSTRAINT `web_dav_credentials_accounts_revoked_by_account` FOREIGN KEY (`revoked_by_account_id`) REFERENCES `accounts` (`id`) ON DELETE NO ACTION);
-- create index "web_dav_credentials_public_id_key" to table: "web_dav_credentials"
CREATE UNIQUE INDEX `web_dav_credentials_public_id_key` ON `web_dav_credentials` (`public_id`);
-- create index "web_dav_credentials_username_key" to table: "web_dav_credentials"
CREATE UNIQUE INDEX `web_dav_credentials_username_key` ON `web_dav_credentials` (`username`);
-- create index "webdavcredential_account_id_tenant_id_space_public_id_revoked_at" to table: "web_dav_credentials"
CREATE INDEX `webdavcredential_account_id_tenant_id_space_public_id_revoked_at` ON `web_dav_credentials` (`account_id`, `tenant_id`, `space_public_id`, `revoked_at`);
-- create index "webdavcredential_tenant_id_account_id_revoked_at" to table: "web_dav_credentials"
CREATE INDEX `webdavcredential_tenant_id_account_id_revoked_at` ON `web_dav_credentials` (`tenant_id`, `account_id`, `revoked_at`);
