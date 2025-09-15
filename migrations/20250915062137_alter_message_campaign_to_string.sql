ALTER TABLE `message_campaign`
    MODIFY COLUMN `category` varchar(255) NOT NULL,
    MODIFY COLUMN `item` varchar(255) NOT NULL,
    MODIFY COLUMN `status` varchar(255) NOT NULL DEFAULT "draft",
    MODIFY COLUMN `target` varchar(255) NOT NULL;

ALTER TABLE `message_campaign`
    ADD INDEX `idx_message_campaign_category` (`category`),
    ADD INDEX `idx_message_campaign_created_at` (`created_at`),
    ADD INDEX `idx_message_campaign_created_by` (`created_by`),
    ADD INDEX `idx_message_campaign_status` (`status`);
