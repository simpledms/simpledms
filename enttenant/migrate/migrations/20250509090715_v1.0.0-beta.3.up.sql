-- drop index "file_name_parent_id" from table: "files"
DROP INDEX `file_name_parent_id`;
-- create index "file_space_id_name_parent_id" to table: "files"
CREATE UNIQUE INDEX `file_space_id_name_parent_id` ON `files` (`space_id`, `name`, `parent_id`) WHERE `deleted_at` is null and `is_in_inbox` = false;
