-- reverse: add column "upload_succeeded_at" to table: "temporary_files"
ALTER TABLE `temporary_files` DROP COLUMN `upload_succeeded_at`;
-- reverse: add column "upload_failed_at" to table: "temporary_files"
ALTER TABLE `temporary_files` DROP COLUMN `upload_failed_at`;
-- reverse: add column "upload_started_at" to table: "temporary_files"
ALTER TABLE `temporary_files` DROP COLUMN `upload_started_at`;
