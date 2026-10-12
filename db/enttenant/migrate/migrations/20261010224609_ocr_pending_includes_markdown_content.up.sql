-- drop index "file_ocr_pending" from table: "files"
DROP INDEX `file_ocr_pending`;
-- create index "file_ocr_pending_with_markdown_content" to table: "files"
CREATE INDEX `file_ocr_pending_with_markdown_content` ON `files` (`ocr_last_tried_at`, `id`) WHERE (`ocr_success_at` is null or `is_ocr_content_from_markdown` = true) and `ocr_retry_count` < 3 and `is_directory` = false;
