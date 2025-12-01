-- Modify "message_campaign" table
ALTER TABLE `message_campaign` ADD COLUMN `target_detail` varchar(512) NULL AFTER `target`;
