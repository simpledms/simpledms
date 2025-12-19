-- create index "file_name_parent_id" to table: "files"
CREATE UNIQUE INDEX `file_name_parent_id` ON `files` (`name`, `parent_id`) WHERE `deleted_at` is null and `is_in_inbox` = false;
