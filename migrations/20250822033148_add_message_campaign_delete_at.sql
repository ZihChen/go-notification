-- Modify "message_campaign" table
ALTER TABLE `message_campaign` ADD COLUMN `deleted_at` datetime(3) NULL, ADD INDEX `idx_message_campaign_deleted_at` (`deleted_at`);
