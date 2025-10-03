-- Modify "player_message" table
ALTER TABLE `player_message` DROP INDEX `idx_player_campaign`,
    ADD INDEX `idx_player_message_campaign_id` (`campaign_id`),
    ADD INDEX `idx_player_message_player_id` (`player_id`);
