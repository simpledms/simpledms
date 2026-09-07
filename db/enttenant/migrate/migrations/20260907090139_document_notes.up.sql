-- create "document_notes" table
CREATE TABLE `document_notes` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `public_id` text NOT NULL, `body` text NOT NULL, `authored_at` datetime NULL, `edited_at` datetime NULL, `deleted_at` datetime NULL, `space_id` integer NOT NULL, `file_id` integer NOT NULL, `author_id` integer NULL, `editor_id` integer NULL, `replaced_by_id` integer NULL, CONSTRAINT `document_notes_spaces_space` FOREIGN KEY (`space_id`) REFERENCES `spaces` (`id`) ON DELETE NO ACTION, CONSTRAINT `document_notes_files_file` FOREIGN KEY (`file_id`) REFERENCES `files` (`id`) ON DELETE NO ACTION, CONSTRAINT `document_notes_users_author` FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE SET NULL, CONSTRAINT `document_notes_users_editor` FOREIGN KEY (`editor_id`) REFERENCES `users` (`id`) ON DELETE SET NULL, CONSTRAINT `document_notes_document_notes_replacement` FOREIGN KEY (`replaced_by_id`) REFERENCES `document_notes` (`id`) ON DELETE SET NULL);
-- create index "document_notes_public_id_key" to table: "document_notes"
CREATE UNIQUE INDEX `document_notes_public_id_key` ON `document_notes` (`public_id`);
-- create index "document_notes_replaced_by_id_key" to table: "document_notes"
CREATE UNIQUE INDEX `document_notes_replaced_by_id_key` ON `document_notes` (`replaced_by_id`);
-- create index "documentnote_space_id" to table: "document_notes"
CREATE INDEX `documentnote_space_id` ON `document_notes` (`space_id`);
-- create index "documentnote_file_id" to table: "document_notes"
CREATE INDEX `documentnote_file_id` ON `document_notes` (`file_id`);
