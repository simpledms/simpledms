-- create index "mail_pending" to table: "mails"
CREATE INDEX `mail_pending` ON `mails` (`last_tried_at`, `id`) WHERE `sent_at` is null and `retry_count` < 3;
-- create index "temporaryfile_delete_pending" to table: "temporary_files"
CREATE INDEX `temporaryfile_delete_pending` ON `temporary_files` (`expires_at`, `id`) WHERE `converted_to_stored_file_at` is null and `deleted_at` is null;
