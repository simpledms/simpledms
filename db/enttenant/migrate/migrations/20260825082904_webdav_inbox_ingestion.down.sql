-- reverse: create index "webdavresource_stored_file_id" to table: "web_dav_resources"
DROP INDEX `webdavresource_stored_file_id`;
-- reverse: create index "webdavresource_file_id" to table: "web_dav_resources"
DROP INDEX `webdavresource_file_id`;
-- reverse: create index "webdavresource_state_last_progress_at_id" to table: "web_dav_resources"
DROP INDEX `webdavresource_state_last_progress_at_id`;
-- reverse: create index "webdavresource_active_path" to table: "web_dav_resources"
DROP INDEX `webdavresource_active_path`;
-- reverse: create index "webdavresource_space_id" to table: "web_dav_resources"
DROP INDEX `webdavresource_space_id`;
-- reverse: create "web_dav_resources" table
DROP TABLE `web_dav_resources`;
-- reverse: add column "source_conversion_claim_token" to table: "stored_files"
ALTER TABLE `stored_files` DROP COLUMN `source_conversion_claim_token`;
-- reverse: add column "source_temporary_file_public_id" to table: "stored_files"
ALTER TABLE `stored_files` DROP COLUMN `source_temporary_file_public_id`;
-- reverse: add column "storage_crc32c" to table: "stored_files"
ALTER TABLE `stored_files` DROP COLUMN `storage_crc32c`;
-- reverse: add column "upload_last_progress_at" to table: "stored_files"
ALTER TABLE `stored_files` DROP COLUMN `upload_last_progress_at`;
-- reverse: add column "source" to table: "files"
ALTER TABLE `files` DROP COLUMN `source`;
