-- add column "upload_started_at" to table: "stored_files"
ALTER TABLE `stored_files` ADD COLUMN `upload_started_at` datetime NULL;
-- add column "upload_failed_at" to table: "stored_files"
ALTER TABLE `stored_files` ADD COLUMN `upload_failed_at` datetime NULL;
-- add column "upload_succeeded_at" to table: "stored_files"
ALTER TABLE `stored_files` ADD COLUMN `upload_succeeded_at` datetime NULL;
