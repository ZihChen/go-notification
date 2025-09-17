-- Modify "message_campaign" table
ALTER TABLE `message_campaign` ADD COLUMN `legacy_id` bigint unsigned NULL COMMENT "舊系統notification ID，用於資料遷移" AFTER `global_id`, ADD INDEX `idx_message_campaign_legacy_id` (`legacy_id`);
