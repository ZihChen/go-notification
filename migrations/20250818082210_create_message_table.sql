-- Modify "message_campaign" table
ALTER TABLE `message_campaign` MODIFY COLUMN `content` mediumtext NULL, RENAME COLUMN `focus` TO `target`, DROP COLUMN `total_target_count`;
-- Modify "player_message" table
ALTER TABLE `player_message` DROP INDEX `idx_player_message_global_player_id`;
-- Modify "player_message" table
ALTER TABLE `player_message` DROP COLUMN `title`, DROP COLUMN `content`,
    ADD COLUMN `player_id` bigint unsigned NOT NULL AFTER `global_player_id`,
    ADD COLUMN `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    DROP INDEX `idx_player_message_campaign_id`,
    ADD INDEX `idx_player_message_global_player_id` (`global_player_id`),
    ADD UNIQUE INDEX `idx_player_campaign` (`player_id`, `campaign_id`);
