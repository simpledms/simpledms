-- create index "file_ocr_pending" to table: "files"
CREATE INDEX `file_ocr_pending` ON `files` (`ocr_last_tried_at`, `id`) WHERE `ocr_success_at` is null and `ocr_retry_count` < 3 and `is_directory` = false;
-- create index "storedfile_content_hash_pending" to table: "stored_files"
CREATE INDEX `storedfile_content_hash_pending` ON `stored_files` (`id`) WHERE `content_sha256` is null and `upload_succeeded_at` is not null and `copied_to_final_destination_at` is not null;
-- create index "storedfile_copy_pending" to table: "stored_files"
CREATE INDEX `storedfile_copy_pending` ON `stored_files` (`copied_to_final_destination_at`, `id`) WHERE `copied_to_final_destination_at` is null and `deleted_temporary_file_at` is null;
-- create index "storedfile_temp_delete_pending" to table: "stored_files"
CREATE INDEX `storedfile_temp_delete_pending` ON `stored_files` (`copied_to_final_destination_at`, `id`) WHERE `copied_to_final_destination_at` is not null and `deleted_temporary_file_at` is null;
