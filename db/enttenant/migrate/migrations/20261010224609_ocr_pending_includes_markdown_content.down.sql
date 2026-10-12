-- reverse: create index "file_ocr_pending_with_markdown_content" to table: "files"
DROP INDEX `file_ocr_pending_with_markdown_content`;
-- reverse: drop index "file_ocr_pending" from table: "files"
CREATE INDEX `file_ocr_pending` ON `files` (`ocr_last_tried_at`, `id`) WHERE `ocr_success_at` is null and `ocr_retry_count` < 3 and `is_directory` = false;
