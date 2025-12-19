-- create "attributes" table
CREATE TABLE `attributes` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL DEFAULT (''), `is_name_giving` bool NOT NULL DEFAULT (false), `is_protected` bool NOT NULL DEFAULT (false), `is_disabled` bool NOT NULL DEFAULT (false), `is_required` bool NOT NULL DEFAULT (false), `type` text NOT NULL, `space_id` integer NOT NULL, `tag_id` integer NULL, `property_id` integer NULL, `document_type_id` integer NOT NULL, CONSTRAINT `attributes_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `attributes_tags_tag` FOREIGN KEY (`tag_id`) REFERENCES `tags` (`id`) ON DELETE NO ACTION, CONSTRAINT `attributes_properties_property` FOREIGN KEY (`property_id`) REFERENCES `properties` (`id`) ON DELETE NO ACTION, CONSTRAINT `attributes_document_types_attributes` FOREIGN KEY (`document_type_id`) REFERENCES `document_types` (`id`) ON DELETE NO ACTION);
-- create index "attribute_space_id_document_type_id_name" to table: "attributes"
CREATE UNIQUE INDEX `attribute_space_id_document_type_id_name` ON `attributes` (`space_id`, `document_type_id`, `name`) WHERE `name` != '';
-- create index "attribute_space_id_document_type_id_tag_id" to table: "attributes"
CREATE UNIQUE INDEX `attribute_space_id_document_type_id_tag_id` ON `attributes` (`space_id`, `document_type_id`, `tag_id`);
-- create index "attribute_space_id_document_type_id_property_id" to table: "attributes"
CREATE UNIQUE INDEX `attribute_space_id_document_type_id_property_id` ON `attributes` (`space_id`, `document_type_id`, `property_id`);
-- create "document_types" table
CREATE TABLE `document_types` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `icon` text NULL, `is_protected` bool NOT NULL DEFAULT (false), `is_disabled` bool NOT NULL DEFAULT (false), `space_id` integer NOT NULL, CONSTRAINT `document_types_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION);
-- create "files" table
CREATE TABLE `files` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `deleted_at` datetime NULL, `public_id` text NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `is_directory` bool NOT NULL, `notes` text NULL, `modified_at` datetime NULL, `indexed_at` datetime NOT NULL, `indexing_completed_at` datetime NULL, `is_in_inbox` bool NOT NULL DEFAULT (false), `is_root_dir` bool NOT NULL DEFAULT (false), `deleted_by` integer NULL, `created_by` integer NULL, `updated_by` integer NULL, `space_id` integer NOT NULL, `parent_id` integer NULL, `document_type_id` integer NULL, CONSTRAINT `files_users_deleter` FOREIGN KEY (`deleted_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `files_users_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `files_users_updater` FOREIGN KEY (`updated_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `files_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `files_files_parent` FOREIGN KEY (`parent_id`) REFERENCES `files` (`id`) ON DELETE NO ACTION, CONSTRAINT `files_document_types_document_type` FOREIGN KEY (`document_type_id`) REFERENCES `document_types` (`id`) ON DELETE NO ACTION);
-- create index "files_public_id_key" to table: "files"
CREATE UNIQUE INDEX `files_public_id_key` ON `files` (`public_id`);
-- create index "file_space_id_is_root_dir" to table: "files"
CREATE UNIQUE INDEX `file_space_id_is_root_dir` ON `files` (`space_id`, `is_root_dir`) WHERE `is_root_dir` = true;
-- create "file_property_assignments" table
CREATE TABLE `file_property_assignments` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `text_value` text NULL, `number_value` integer NULL, `date_value` datetime NULL, `bool_value` bool NULL, `space_id` integer NOT NULL, `file_id` integer NOT NULL, `property_id` integer NOT NULL, CONSTRAINT `file_property_assignments_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `file_property_assignments_files_file` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON DELETE NO ACTION, CONSTRAINT `file_property_assignments_properties_property` FOREIGN KEY (`property_id`) REFERENCES `properties` (`id`) ON DELETE NO ACTION);
-- create index "filepropertyassignment_file_id_property_id" to table: "file_property_assignments"
CREATE UNIQUE INDEX `filepropertyassignment_file_id_property_id` ON `file_property_assignments` (`file_id`, `property_id`);
-- create "properties" table
CREATE TABLE `properties` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `type` text NOT NULL, `unit` text NOT NULL DEFAULT (''), `space_id` integer NOT NULL, CONSTRAINT `properties_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION);
-- create index "property_space_id_name" to table: "properties"
CREATE UNIQUE INDEX `property_space_id_name` ON `properties` (`space_id`, `name`);
-- create "spaces" table
CREATE TABLE `spaces` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `public_id` text NOT NULL, `deleted_at` datetime NULL, `name` text NOT NULL, `icon` text NULL, `description` text NULL, `is_folder_mode` bool NOT NULL DEFAULT (false), `deleted_by` integer NULL, CONSTRAINT `spaces_users_deleter` FOREIGN KEY (`deleted_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION);
-- create index "spaces_public_id_key" to table: "spaces"
CREATE UNIQUE INDEX `spaces_public_id_key` ON `spaces` (`public_id`);
-- create index "space_name" to table: "spaces"
CREATE UNIQUE INDEX `space_name` ON `spaces` (`name`) WHERE `deleted_at` is null;
-- create "space_user_assignments" table
CREATE TABLE `space_user_assignments` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `role` text NOT NULL, `is_default` bool NOT NULL DEFAULT (false), `space_id` integer NOT NULL, `created_by` integer NULL, `updated_by` integer NULL, `user_id` integer NOT NULL, CONSTRAINT `space_user_assignments_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `space_user_assignments_users_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `space_user_assignments_users_updater` FOREIGN KEY (`updated_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `space_user_assignments_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE NO ACTION);
-- create index "spaceuserassignment_space_id_user_id" to table: "space_user_assignments"
CREATE UNIQUE INDEX `spaceuserassignment_space_id_user_id` ON `space_user_assignments` (`space_id`, `user_id`);
-- create index "spaceuserassignment_user_id_is_default" to table: "space_user_assignments"
CREATE UNIQUE INDEX `spaceuserassignment_user_id_is_default` ON `space_user_assignments` (`user_id`, `is_default`) WHERE `is_default` = true;
-- create "stored_files" table
CREATE TABLE `stored_files` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `public_id` text NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `filename` text NOT NULL, `size` integer NULL, `size_in_storage` integer NOT NULL, `sha256` text NULL, `mime_type` text NULL, `storage_type` text NOT NULL, `bucket_name` text NULL, `storage_path` text NOT NULL, `storage_filename` text NOT NULL, `temporary_storage_path` text NOT NULL, `temporary_storage_filename` text NOT NULL, `copied_to_final_destination_at` datetime NULL, `deleted_temporary_file_at` datetime NULL, `created_by` integer NULL, `updated_by` integer NULL, CONSTRAINT `stored_files_users_creator` FOREIGN KEY (`created_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION, CONSTRAINT `stored_files_users_updater` FOREIGN KEY (`updated_by`) REFERENCES `users` (`id`) ON DELETE NO ACTION);
-- create index "stored_files_public_id_key" to table: "stored_files"
CREATE UNIQUE INDEX `stored_files_public_id_key` ON `stored_files` (`public_id`);
-- create "tags" table
CREATE TABLE `tags` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `color` text NULL, `icon` text NULL, `type` text NOT NULL, `space_id` integer NOT NULL, `group_id` integer NULL, CONSTRAINT `tags_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `tags_tags_group` FOREIGN KEY (`group_id`) REFERENCES `tags` (`id`) ON DELETE NO ACTION);
-- create index "tag_space_id_name" to table: "tags"
CREATE UNIQUE INDEX `tag_space_id_name` ON `tags` (`space_id`, `name`) WHERE `group_id` is null;
-- create index "tag_space_id_name_group_id" to table: "tags"
CREATE UNIQUE INDEX `tag_space_id_name_group_id` ON `tags` (`space_id`, `name`, `group_id`);
-- create "tag_assignments" table
CREATE TABLE `tag_assignments` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `space_id` integer NOT NULL, `tag_id` integer NOT NULL, `file_id` integer NOT NULL, CONSTRAINT `tag_assignments_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `tag_assignments_tags_tag` FOREIGN KEY (`tag_id`) REFERENCES `tags` (`id`) ON DELETE NO ACTION, CONSTRAINT `tag_assignments_files_file` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON DELETE NO ACTION);
-- create index "tagassignment_file_id_tag_id" to table: "tag_assignments"
CREATE UNIQUE INDEX `tagassignment_file_id_tag_id` ON `tag_assignments` (`file_id`, `tag_id`);
-- create "users" table
CREATE TABLE `users` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `created_by` integer NULL, `updated_at` datetime NOT NULL, `updated_by` integer NULL, `deleted_by` integer NULL, `deleted_at` datetime NULL, `public_id` text NOT NULL, `account_id` integer NOT NULL, `role` text NOT NULL, `email` text NOT NULL, `first_name` text NOT NULL, `last_name` text NULL, `avatar` text NULL, `description` text NULL);
-- create index "users_public_id_key" to table: "users"
CREATE UNIQUE INDEX `users_public_id_key` ON `users` (`public_id`);
-- create index "users_account_id_key" to table: "users"
CREATE UNIQUE INDEX `users_account_id_key` ON `users` (`account_id`);
-- create index "users_email_key" to table: "users"
CREATE UNIQUE INDEX `users_email_key` ON `users` (`email`);
-- create "file_versions" table
CREATE TABLE `file_versions` (`file_id` integer NOT NULL, `stored_file_id` integer NOT NULL, PRIMARY KEY (`file_id`, `stored_file_id`), CONSTRAINT `file_versions_file_id` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON DELETE CASCADE, CONSTRAINT `file_versions_stored_file_id` FOREIGN KEY (`stored_file_id`) REFERENCES `stored_files` (`id`) ON DELETE CASCADE);
-- create "tag_sub_tags" table
CREATE TABLE `tag_sub_tags` (`tag_id` integer NOT NULL, `super_tag_id` integer NOT NULL, PRIMARY KEY (`tag_id`, `super_tag_id`), CONSTRAINT `tag_sub_tags_tag_id` FOREIGN KEY (`tag_id`) REFERENCES `tags` (`id`) ON DELETE CASCADE, CONSTRAINT `tag_sub_tags_super_tag_id` FOREIGN KEY (`super_tag_id`) REFERENCES `tags` (`id`) ON DELETE CASCADE);
