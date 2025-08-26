-- Modify "message_campaign" table
ALTER TABLE `message_campaign` ADD COLUMN `status` tinyint unsigned NOT NULL DEFAULT 1 AFTER `content`;
