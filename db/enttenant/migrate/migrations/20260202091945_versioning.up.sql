-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_file_versions" table
CREATE TABLE `new_file_versions` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `version_number` integer NOT NULL DEFAULT (1), `note` text NULL, `file_id` integer NOT NULL, `stored_file_id` integer NOT NULL, CONSTRAINT `file_versions_files_file` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON DELETE NO ACTION, CONSTRAINT `file_versions_stored_files_stored_file` FOREIGN KEY (`stored_file_id`) REFERENCES `stored_files` (`id`) ON DELETE NO ACTION);
-- copy rows from old table "file_versions" to new temporary table "new_file_versions"
INSERT INTO `new_file_versions` (`file_id`, `stored_file_id`) SELECT `file_id`, `stored_file_id` FROM `file_versions`;
-- drop "file_versions" table after copying rows
DROP TABLE `file_versions`;
-- rename temporary table "new_file_versions" to "file_versions"
ALTER TABLE `new_file_versions` RENAME TO `file_versions`;
-- create index "fileversion_file_id_stored_file_id" to table: "file_versions"
CREATE UNIQUE INDEX `fileversion_file_id_stored_file_id` ON `file_versions` (`file_id`, `stored_file_id`);
-- create index "fileversion_file_id_version_number" to table: "file_versions"
CREATE UNIQUE INDEX `fileversion_file_id_version_number` ON `file_versions` (`file_id`, `version_number`);
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
