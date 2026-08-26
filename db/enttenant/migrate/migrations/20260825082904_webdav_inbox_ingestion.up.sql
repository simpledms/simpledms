-- add column "source" to table: "files"
ALTER TABLE `files` ADD COLUMN `source` text NOT NULL DEFAULT 'UnknownLegacy';
-- add column "upload_last_progress_at" to table: "stored_files"
ALTER TABLE `stored_files` ADD COLUMN `upload_last_progress_at` datetime NULL;
-- add column "storage_crc32c" to table: "stored_files"
ALTER TABLE `stored_files` ADD COLUMN `storage_crc32c` text NULL;
-- add column "source_temporary_file_public_id" to table: "stored_files"
ALTER TABLE `stored_files` ADD COLUMN `source_temporary_file_public_id` text NULL;
-- add column "source_conversion_claim_token" to table: "stored_files"
ALTER TABLE `stored_files` ADD COLUMN `source_conversion_claim_token` text NULL;
-- create "web_dav_resources" table
CREATE TABLE `web_dav_resources` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `credential_public_id` text NOT NULL, `dav_path` text NOT NULL, `state` text NOT NULL DEFAULT ('Uploading'), `last_progress_at` datetime NOT NULL, `finalized_at` datetime NULL, `created_by` integer NULL, `updated_by` integer NULL, `space_id` integer NOT NULL, `file_id` integer NULL, `stored_file_id` integer NULL, CONSTRAINT `web_dav_resources_users_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `web_dav_resources_users_updater` FOREIGN KEY (`updated_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `web_dav_resources_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `web_dav_resources_files_file` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON DELETE NO ACTION, CONSTRAINT `web_dav_resources_stored_files_stored_file` FOREIGN KEY (`stored_file_id`) REFERENCES `stored_files` (`id`) ON DELETE NO ACTION);
-- create index "webdavresource_space_id" to table: "web_dav_resources"
CREATE INDEX `webdavresource_space_id` ON `web_dav_resources` (`space_id`);
-- create index "webdavresource_active_path" to table: "web_dav_resources"
CREATE UNIQUE INDEX `webdavresource_active_path` ON `web_dav_resources` (`credential_public_id`, `space_id`, `dav_path`) WHERE `state` in ('Uploading', 'Active');
-- create index "webdavresource_state_last_progress_at_id" to table: "web_dav_resources"
CREATE INDEX `webdavresource_state_last_progress_at_id` ON `web_dav_resources` (`state`, `last_progress_at`, `id`);
-- create index "webdavresource_file_id" to table: "web_dav_resources"
CREATE INDEX `webdavresource_file_id` ON `web_dav_resources` (`file_id`);
-- create index "webdavresource_stored_file_id" to table: "web_dav_resources"
CREATE INDEX `webdavresource_stored_file_id` ON `web_dav_resources` (`stored_file_id`);
