-- reverse: add column "ocr_markdown_last_tried_at" to table: "files"
ALTER TABLE `files` DROP COLUMN `ocr_markdown_last_tried_at`;
-- reverse: add column "ocr_markdown_retry_count" to table: "files"
ALTER TABLE `files` DROP COLUMN `ocr_markdown_retry_count`;
-- reverse: add column "ocr_markdown_success_at" to table: "files"
ALTER TABLE `files` DROP COLUMN `ocr_markdown_success_at`;
-- reverse: add column "ocr_markdown" to table: "files"
ALTER TABLE `files` DROP COLUMN `ocr_markdown`;
