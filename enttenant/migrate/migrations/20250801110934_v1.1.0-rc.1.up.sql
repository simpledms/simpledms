-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_files" table
CREATE TABLE `new_files` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `deleted_at` datetime NULL, `public_id` text NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `is_directory` bool NOT NULL, `notes` text NULL, `modified_at` datetime NULL, `indexed_at` datetime NOT NULL, `indexing_completed_at` datetime NULL, `is_in_inbox` bool NOT NULL DEFAULT (false), `is_root_dir` bool NOT NULL DEFAULT (false), `ocr_content` text NOT NULL DEFAULT (''), `ocr_success_at` datetime NULL, `ocr_retry_count` integer NOT NULL DEFAULT (0), `ocr_last_tried_at` datetime NOT NULL, `deleted_by` integer NULL, `created_by` integer NULL, `updated_by` integer NULL, `space_id` integer NOT NULL, `parent_id` integer NULL, `document_type_id` integer NULL, CONSTRAINT `files_users_deleter` FOREIGN KEY (`deleted_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `files_users_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `files_users_updater` FOREIGN KEY (`updated_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `files_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `files_files_parent` FOREIGN KEY (`parent_id`) REFERENCES `files` (`id`) ON DELETE NO ACTION, CONSTRAINT `files_document_types_document_type` FOREIGN KEY (`document_type_id`) REFERENCES `document_types` (`id`) ON DELETE NO ACTION);
-- copy rows from old table "files" to new temporary table "new_files"
INSERT INTO `new_files` (`id`, `deleted_at`, `public_id`, `created_at`, `updated_at`, `name`, `is_directory`, `notes`, `modified_at`, `indexed_at`, `indexing_completed_at`, `is_in_inbox`, `is_root_dir`, `deleted_by`, `created_by`, `updated_by`, `space_id`, `parent_id`, `document_type_id`, `ocr_last_tried_at`) SELECT `id`, `deleted_at`, `public_id`, `created_at`, `updated_at`, `name`, `is_directory`, `notes`, `modified_at`, `indexed_at`, `indexing_completed_at`, `is_in_inbox`, `is_root_dir`, `deleted_by`, `created_by`, `updated_by`, `space_id`, `parent_id`, `document_type_id`, '0000-01-01 00:00:00' FROM `files`;
-- drop "files" table after copying rows
DROP TABLE `files`;
-- rename temporary table "new_files" to "files"
ALTER TABLE `new_files` RENAME TO `files`;
-- create index "files_public_id_key" to table: "files"
CREATE UNIQUE INDEX `files_public_id_key` ON `files` (`public_id`);
-- create index "file_space_id_name_parent_id" to table: "files"
CREATE UNIQUE INDEX `file_space_id_name_parent_id` ON `files` (`space_id`, `name`, `parent_id`) WHERE `deleted_at` is null and `is_in_inbox` = false;
-- create index "file_space_id_is_root_dir" to table: "files"
CREATE UNIQUE INDEX `file_space_id_is_root_dir` ON `files` (`space_id`, `is_root_dir`) WHERE `is_root_dir` = true;
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
