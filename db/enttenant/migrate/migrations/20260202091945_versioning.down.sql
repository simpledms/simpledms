-- reverse: create index "fileversion_file_id_version_number" to table: "file_versions"
DROP INDEX `fileversion_file_id_version_number`;
-- reverse: create index "fileversion_file_id_stored_file_id" to table: "file_versions"
DROP INDEX `fileversion_file_id_stored_file_id`;
-- reverse: create "new_file_versions" table
DROP TABLE `new_file_versions`;
