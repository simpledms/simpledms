-- reverse: create index "webdavcredential_tenant_id_account_id_revoked_at" to table: "web_dav_credentials"
DROP INDEX `webdavcredential_tenant_id_account_id_revoked_at`;
-- reverse: create index "webdavcredential_account_id_tenant_id_space_public_id_revoked_at" to table: "web_dav_credentials"
DROP INDEX `webdavcredential_account_id_tenant_id_space_public_id_revoked_at`;
-- reverse: create index "web_dav_credentials_username_key" to table: "web_dav_credentials"
DROP INDEX `web_dav_credentials_username_key`;
-- reverse: create index "web_dav_credentials_public_id_key" to table: "web_dav_credentials"
DROP INDEX `web_dav_credentials_public_id_key`;
-- reverse: create "web_dav_credentials" table
DROP TABLE `web_dav_credentials`;
-- reverse: add column "persistence_last_progress_at" to table: "temporary_files"
ALTER TABLE `temporary_files` DROP COLUMN `persistence_last_progress_at`;
-- reverse: add column "persistence_tenant_id" to table: "temporary_files"
ALTER TABLE `temporary_files` DROP COLUMN `persistence_tenant_id`;
-- reverse: add column "persistence_claim_token" to table: "temporary_files"
ALTER TABLE `temporary_files` DROP COLUMN `persistence_claim_token`;
-- reverse: add column "storage_crc32c" to table: "temporary_files"
ALTER TABLE `temporary_files` DROP COLUMN `storage_crc32c`;
-- reverse: add column "content_sha256" to table: "temporary_files"
ALTER TABLE `temporary_files` DROP COLUMN `content_sha256`;
-- reverse: add column "source" to table: "temporary_files"
ALTER TABLE `temporary_files` DROP COLUMN `source`;
-- reverse: add column "upload_last_progress_at" to table: "temporary_files"
ALTER TABLE `temporary_files` DROP COLUMN `upload_last_progress_at`;
