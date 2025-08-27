-- Modify "message_campaign" table
ALTER TABLE `message_campaign` ADD COLUMN `merchant_id` bigint unsigned NOT NULL DEFAULT 0 AFTER `item`, ADD COLUMN `global_id` varchar(100) NOT NULL DEFAULT "" AFTER `merchant_id`, ADD UNIQUE INDEX `idx_message_campaign_global_id` (`global_id`), ADD INDEX `idx_message_campaign_merchant_id` (`merchant_id`);
