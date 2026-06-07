-- create index "file_browse_name" to table: "files"
CREATE INDEX `file_browse_name` ON `files` (`space_id`, `parent_id`, `is_directory` DESC, `name`) WHERE `deleted_at` is null and `is_in_inbox` = false;
-- create index "file_browse_created" to table: "files"
CREATE INDEX `file_browse_created` ON `files` (`space_id`, `parent_id`, `is_directory` DESC, `created_at` DESC, `name`) WHERE `deleted_at` is null and `is_in_inbox` = false;
-- create index "file_browse_created_oldest" to table: "files"
CREATE INDEX `file_browse_created_oldest` ON `files` (`space_id`, `parent_id`, `is_directory` DESC, `created_at`, `name`) WHERE `deleted_at` is null and `is_in_inbox` = false;
-- create index "file_inbox_created" to table: "files"
CREATE INDEX `file_inbox_created` ON `files` (`space_id`, `is_directory`, `created_at` DESC) WHERE `deleted_at` is null and `is_in_inbox` = true;
-- create index "file_inbox_name" to table: "files"
CREATE INDEX `file_inbox_name` ON `files` (`space_id`, `is_directory`, `name`) WHERE `deleted_at` is null and `is_in_inbox` = true;
