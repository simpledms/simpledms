-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Drop index "user_email" from table: "users"
DROP INDEX `user_email`;
-- Create index "users_email_key" to table: "users"
CREATE UNIQUE INDEX `users_email_key` ON `users` (`email`);
-- Create index "file_browse_name" to table: "files"
CREATE INDEX `file_browse_name` ON `files` (`space_id`, `parent_id`, `is_directory` DESC, `name`) WHERE `deleted_at` is null and `is_in_inbox` = false;
-- Create index "file_browse_created" to table: "files"
CREATE INDEX `file_browse_created` ON `files` (`space_id`, `parent_id`, `is_directory` DESC, `created_at` DESC, `name`) WHERE `deleted_at` is null and `is_in_inbox` = false;
-- Create index "file_browse_created_oldest" to table: "files"
CREATE INDEX `file_browse_created_oldest` ON `files` (`space_id`, `parent_id`, `is_directory` DESC, `created_at`, `name`) WHERE `deleted_at` is null and `is_in_inbox` = false;
-- Create index "file_inbox_created" to table: "files"
CREATE INDEX `file_inbox_created` ON `files` (`space_id`, `is_directory`, `created_at` DESC) WHERE `deleted_at` is null and `is_in_inbox` = true;
-- Create index "file_inbox_name" to table: "files"
CREATE INDEX `file_inbox_name` ON `files` (`space_id`, `is_directory`, `name`) WHERE `deleted_at` is null and `is_in_inbox` = true;
-- Create "new_attributes" table
CREATE TABLE `new_attributes` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL DEFAULT '', `is_name_giving` bool NOT NULL DEFAULT false, `is_protected` bool NOT NULL DEFAULT false, `is_disabled` bool NOT NULL DEFAULT false, `is_required` bool NOT NULL DEFAULT false, `type` text NOT NULL, `space_id` integer NOT NULL, `tag_id` integer NULL, `property_id` integer NULL, `document_type_id` integer NOT NULL, CONSTRAINT `attributes_document_types_attributes` FOREIGN KEY (`document_type_id`) REFERENCES `document_types` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT `attributes_properties_property` FOREIGN KEY (`property_id`) REFERENCES `properties` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT `attributes_tags_tag` FOREIGN KEY (`tag_id`) REFERENCES `tags` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT `attributes_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Copy rows from old table "attributes" to new temporary table "new_attributes"
INSERT INTO `new_attributes` (`id`, `name`, `is_name_giving`, `is_protected`, `is_disabled`, `is_required`, `type`, `space_id`, `tag_id`, `property_id`, `document_type_id`) SELECT `id`, `name`, `is_name_giving`, `is_protected`, `is_disabled`, `is_required`, `type`, `space_id`, `tag_id`, `property_id`, `document_type_id` FROM `attributes`;
-- Drop "attributes" table after copying rows
DROP TABLE `attributes`;
-- Rename temporary table "new_attributes" to "attributes"
ALTER TABLE `new_attributes` RENAME TO `attributes`;
-- Create index "attribute_space_id_document_type_id_name" to table: "attributes"
CREATE UNIQUE INDEX `attribute_space_id_document_type_id_name` ON `attributes` (`space_id`, `document_type_id`, `name`) WHERE `name` != '';
-- Create index "attribute_space_id_document_type_id_tag_id" to table: "attributes"
CREATE UNIQUE INDEX `attribute_space_id_document_type_id_tag_id` ON `attributes` (`space_id`, `document_type_id`, `tag_id`);
-- Create index "attribute_space_id_document_type_id_property_id" to table: "attributes"
CREATE UNIQUE INDEX `attribute_space_id_document_type_id_property_id` ON `attributes` (`space_id`, `document_type_id`, `property_id`);
-- Create index "attribute_space_id" to table: "attributes"
CREATE INDEX `attribute_space_id` ON `attributes` (`space_id`);
-- Create "new_file_property_assignments" table
CREATE TABLE `new_file_property_assignments` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `text_value` text NULL, `number_value` integer NULL, `date_value` datetime NULL, `bool_value` bool NULL, `space_id` integer NOT NULL, `file_id` integer NOT NULL, `property_id` integer NOT NULL, CONSTRAINT `file_property_assignments_properties_property` FOREIGN KEY (`property_id`) REFERENCES `properties` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT `file_property_assignments_files_file` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION, CONSTRAINT `file_property_assignments_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Copy rows from old table "file_property_assignments" to new temporary table "new_file_property_assignments"
INSERT INTO `new_file_property_assignments` (`id`, `text_value`, `number_value`, `date_value`, `bool_value`, `space_id`, `file_id`, `property_id`) SELECT `id`, `text_value`, `number_value`, `date_value`, `bool_value`, `space_id`, `file_id`, `property_id` FROM `file_property_assignments`;
-- Drop "file_property_assignments" table after copying rows
DROP TABLE `file_property_assignments`;
-- Rename temporary table "new_file_property_assignments" to "file_property_assignments"
ALTER TABLE `new_file_property_assignments` RENAME TO `file_property_assignments`;
-- Create index "filepropertyassignment_file_id_property_id" to table: "file_property_assignments"
CREATE UNIQUE INDEX `filepropertyassignment_file_id_property_id` ON `file_property_assignments` (`file_id`, `property_id`);
-- Create index "filepropertyassignment_space_id" to table: "file_property_assignments"
CREATE INDEX `filepropertyassignment_space_id` ON `file_property_assignments` (`space_id`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
