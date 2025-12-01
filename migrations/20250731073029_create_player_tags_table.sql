-- Create "player_tags" table
CREATE TABLE `player_tags` (
  `player_id` bigint unsigned NULL,
  `tag_id` bigint unsigned NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE INDEX `idx_player_tag` (`player_id`, `tag_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- Create "tags" table
CREATE TABLE `tags` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `merchant_id` bigint unsigned NOT NULL,
  `name` varchar(255) NOT NULL,
  `global_tag_id` varchar(100) NOT NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_tags_deleted_at` (`deleted_at`),
  UNIQUE INDEX `idx_tags_global_tag_id` (`global_tag_id`),
  INDEX `idx_tags_merchant_id` (`merchant_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
