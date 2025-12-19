-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_stored_files" table
CREATE TABLE `new_stored_files` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `filename` text NOT NULL, `size` integer NULL, `size_in_storage` integer NOT NULL, `sha256` text NULL, `mime_type` text NULL, `storage_type` text NOT NULL, `bucket_name` text NULL, `storage_path` text NOT NULL, `storage_filename` text NOT NULL, `temporary_storage_path` text NOT NULL, `temporary_storage_filename` text NOT NULL, `copied_to_final_destination_at` datetime NULL, `deleted_temporary_file_at` datetime NULL, `created_by` integer NULL, `updated_by` integer NULL, CONSTRAINT `stored_files_users_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `stored_files_users_updater` FOREIGN KEY (`updated_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION);
-- copy rows from old table "stored_files" to new temporary table "new_stored_files"
INSERT INTO `new_stored_files` (`id`, `created_at`, `updated_at`, `filename`, `size`, `size_in_storage`, `sha256`, `mime_type`, `storage_type`, `bucket_name`, `storage_path`, `storage_filename`, `temporary_storage_path`, `temporary_storage_filename`, `copied_to_final_destination_at`, `deleted_temporary_file_at`, `created_by`, `updated_by`) SELECT `id`, `created_at`, `updated_at`, `filename`, `size`, `size_in_storage`, `sha256`, `mime_type`, `storage_type`, `bucket_name`, `storage_path`, `storage_filename`, `temporary_storage_path`, `temporary_storage_filename`, `copied_to_final_destination_at`, `deleted_temporary_file_at`, `created_by`, `updated_by` FROM `stored_files`;
-- drop "stored_files" table after copying rows
DROP TABLE `stored_files`;
-- rename temporary table "new_stored_files" to "stored_files"
ALTER TABLE `new_stored_files` RENAME TO `stored_files`;
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
