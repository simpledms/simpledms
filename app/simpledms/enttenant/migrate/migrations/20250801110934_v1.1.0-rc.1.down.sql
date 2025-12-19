-- reverse: create index "file_space_id_is_root_dir" to table: "files"
DROP INDEX `file_space_id_is_root_dir`;
-- reverse: create index "file_space_id_name_parent_id" to table: "files"
DROP INDEX `file_space_id_name_parent_id`;
-- reverse: create index "files_public_id_key" to table: "files"
DROP INDEX `files_public_id_key`;
-- reverse: create "new_files" table
DROP TABLE `new_files`;
