-- add column "file_list_preferences" to table: "accounts"
ALTER TABLE `accounts` ADD COLUMN `file_list_preferences` json NOT NULL DEFAULT '{}';
