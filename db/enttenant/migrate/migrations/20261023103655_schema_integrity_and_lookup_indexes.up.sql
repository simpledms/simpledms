-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_attributes" table
CREATE TABLE `new_attributes` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL DEFAULT (''), `is_name_giving` bool NOT NULL DEFAULT (false), `is_protected` bool NOT NULL DEFAULT (false), `is_disabled` bool NOT NULL DEFAULT (false), `is_required` bool NOT NULL DEFAULT (false), `type` text NOT NULL, `space_id` integer NOT NULL, `tag_id` integer NULL, `property_id` integer NULL, `document_type_id` integer NOT NULL, CONSTRAINT `attributes_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `attributes_tags_tag` FOREIGN KEY (`tag_id`) REFERENCES `tags` (`id`) ON DELETE NO ACTION, CONSTRAINT `attributes_properties_property` FOREIGN KEY (`property_id`) REFERENCES `properties` (`id`) ON DELETE NO ACTION, CONSTRAINT `attributes_document_types_attributes` FOREIGN KEY (`document_type_id`) REFERENCES `document_types` (`id`) ON DELETE NO ACTION, CONSTRAINT `attribute_type_target` CHECK ((`type` = 'Tag' and `tag_id` is not null and `property_id` is null) or (`type` = 'Field' and `property_id` is not null and `tag_id` is null)));
-- copy rows from old table "attributes" to new temporary table "new_attributes"
INSERT INTO `new_attributes` (`id`, `name`, `is_name_giving`, `is_protected`, `is_disabled`, `is_required`, `type`, `space_id`, `tag_id`, `property_id`, `document_type_id`) SELECT `id`, `name`, `is_name_giving`, `is_protected`, `is_disabled`, `is_required`, `type`, `space_id`, `tag_id`, `property_id`, `document_type_id` FROM `attributes`;
-- drop "attributes" table after copying rows
DROP TABLE `attributes`;
-- rename temporary table "new_attributes" to "attributes"
ALTER TABLE `new_attributes` RENAME TO `attributes`;
-- create index "attribute_space_id" to table: "attributes"
CREATE INDEX `attribute_space_id` ON `attributes` (`space_id`);
-- create index "attribute_space_id_document_type_id_name" to table: "attributes"
CREATE UNIQUE INDEX `attribute_space_id_document_type_id_name` ON `attributes` (`space_id`, `document_type_id`, `name`) WHERE `name` != '';
-- create index "attribute_space_id_document_type_id_tag_id" to table: "attributes"
CREATE UNIQUE INDEX `attribute_space_id_document_type_id_tag_id` ON `attributes` (`space_id`, `document_type_id`, `tag_id`);
-- create index "attribute_space_id_document_type_id_property_id" to table: "attributes"
CREATE UNIQUE INDEX `attribute_space_id_document_type_id_property_id` ON `attributes` (`space_id`, `document_type_id`, `property_id`);
-- create "new_file_property_assignments" table
CREATE TABLE `new_file_property_assignments` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `text_value` text NULL, `number_value` integer NULL, `date_value` datetime NULL, `bool_value` bool NULL, `space_id` integer NOT NULL, `file_id` integer NOT NULL, `property_id` integer NOT NULL, CONSTRAINT `file_property_assignments_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `file_property_assignments_files_file` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON DELETE NO ACTION, CONSTRAINT `file_property_assignments_properties_property` FOREIGN KEY (`property_id`) REFERENCES `properties` (`id`) ON DELETE NO ACTION, CONSTRAINT `file_property_assignment_single_value` CHECK (((`text_value` is not null) + (`number_value` is not null) + (`date_value` is not null) + (`bool_value` is not null)) <= 1));
-- copy rows from old table "file_property_assignments" to new temporary table "new_file_property_assignments"
INSERT INTO `new_file_property_assignments` (`id`, `text_value`, `number_value`, `date_value`, `bool_value`, `space_id`, `file_id`, `property_id`) SELECT `id`, `text_value`, `number_value`, `date_value`, `bool_value`, `space_id`, `file_id`, `property_id` FROM `file_property_assignments`;
-- drop "file_property_assignments" table after copying rows
DROP TABLE `file_property_assignments`;
-- rename temporary table "new_file_property_assignments" to "file_property_assignments"
ALTER TABLE `new_file_property_assignments` RENAME TO `file_property_assignments`;
-- create index "filepropertyassignment_space_id" to table: "file_property_assignments"
CREATE INDEX `filepropertyassignment_space_id` ON `file_property_assignments` (`space_id`);
-- create index "filepropertyassignment_file_id_property_id" to table: "file_property_assignments"
CREATE UNIQUE INDEX `filepropertyassignment_file_id_property_id` ON `file_property_assignments` (`file_id`, `property_id`);
-- drop index "users_email_key" from table: "users"
DROP INDEX `users_email_key`;
-- create index "user_email" to table: "users"
CREATE UNIQUE INDEX `user_email` ON `users` (`email`) WHERE `deleted_at` is null;
-- drop index "file_browse_name" from table: "files"
DROP INDEX `file_browse_name`;
-- drop index "file_browse_created" from table: "files"
DROP INDEX `file_browse_created`;
-- drop index "file_browse_created_oldest" from table: "files"
DROP INDEX `file_browse_created_oldest`;
-- drop index "file_inbox_created" from table: "files"
DROP INDEX `file_inbox_created`;
-- drop index "file_inbox_name" from table: "files"
DROP INDEX `file_inbox_name`;
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
