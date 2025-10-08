-- Modify "message_campaign" table
ALTER TABLE `message_campaign` ADD COLUMN `active` bool NOT NULL DEFAULT 1 AFTER `auto_send`;
