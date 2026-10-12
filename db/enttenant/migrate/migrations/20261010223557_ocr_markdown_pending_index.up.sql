-- create index "file_ocr_markdown_pending" to table: "files"
CREATE INDEX `file_ocr_markdown_pending` ON `files` (`ocr_markdown_last_tried_at`, `id`) WHERE `ocr_markdown_success_at` is null and coalesce(`ocr_markdown_retry_count`, 0) < 3 and `is_directory` = false;
