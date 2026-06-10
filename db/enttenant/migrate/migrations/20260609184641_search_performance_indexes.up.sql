-- create index "file_browse_name_fast" to table: "files"
CREATE INDEX `file_browse_name_fast` ON `files` (`space_id`, `parent_id`, `is_in_inbox`, `deleted_at`, `is_directory` DESC, `name`);
-- create index "file_browse_created_fast" to table: "files"
CREATE INDEX `file_browse_created_fast` ON `files` (`space_id`, `parent_id`, `is_in_inbox`, `deleted_at`, `is_directory` DESC, `created_at` DESC, `name`);
-- create index "file_browse_created_oldest_fast" to table: "files"
CREATE INDEX `file_browse_created_oldest_fast` ON `files` (`space_id`, `parent_id`, `is_in_inbox`, `deleted_at`, `is_directory` DESC, `created_at`, `name`);
-- create index "file_inbox_created_fast" to table: "files"
CREATE INDEX `file_inbox_created_fast` ON `files` (`space_id`, `is_in_inbox`, `is_directory`, `deleted_at`, `created_at` DESC);
-- create index "file_inbox_name_fast" to table: "files"
CREATE INDEX `file_inbox_name_fast` ON `files` (`space_id`, `is_in_inbox`, `is_directory`, `deleted_at`, `name`);
