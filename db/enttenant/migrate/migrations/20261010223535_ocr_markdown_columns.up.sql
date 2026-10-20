-- add column "ocr_markdown" to table: "files"
ALTER TABLE `files` ADD COLUMN `ocr_markdown` text NULL;
-- add column "ocr_markdown_success_at" to table: "files"
ALTER TABLE `files` ADD COLUMN `ocr_markdown_success_at` datetime NULL;
-- add column "ocr_markdown_retry_count" to table: "files"
ALTER TABLE `files` ADD COLUMN `ocr_markdown_retry_count` integer NULL;
-- add column "ocr_markdown_last_tried_at" to table: "files"
ALTER TABLE `files` ADD COLUMN `ocr_markdown_last_tried_at` datetime NULL;
