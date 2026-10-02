-- add column "accepts_inbox_transfers" to table: "spaces"
ALTER TABLE `spaces` ADD COLUMN `accepts_inbox_transfers` bool NOT NULL DEFAULT false;
