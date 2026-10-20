-- add column "is_ocr_content_from_markdown" to table: "files"
ALTER TABLE `files` ADD COLUMN `is_ocr_content_from_markdown` bool NULL;
