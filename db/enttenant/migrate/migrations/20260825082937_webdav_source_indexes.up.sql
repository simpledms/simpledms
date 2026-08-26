-- create index "file_inbox_source_created_fast" to table: "files"
CREATE INDEX `file_inbox_source_created_fast` ON `files` (`space_id`, `is_in_inbox`, `deleted_at`, `is_directory`, `source`, `created_at` DESC);
-- create index "file_inbox_source_created_oldest_fast" to table: "files"
CREATE INDEX `file_inbox_source_created_oldest_fast` ON `files` (`space_id`, `is_in_inbox`, `deleted_at`, `is_directory`, `source`, `created_at`);
-- create index "file_inbox_source_name_fast" to table: "files"
CREATE INDEX `file_inbox_source_name_fast` ON `files` (`space_id`, `is_in_inbox`, `deleted_at`, `is_directory`, `source`, `name`);
-- create index "storedfile_source_temporary_file" to table: "stored_files"
CREATE UNIQUE INDEX `storedfile_source_temporary_file` ON `stored_files` (`source_temporary_file_public_id`) WHERE `source_temporary_file_public_id` is not null;
