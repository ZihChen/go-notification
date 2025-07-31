-- Create "level" table
CREATE TABLE `level` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `merchant_id` bigint unsigned NOT NULL,
  `name` varchar(255) NOT NULL,
  `global_player_level_id` varchar(100) NOT NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime(3) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_level_deleted_at` (`deleted_at`),
  UNIQUE INDEX `idx_level_global_player_level_id` (`global_player_level_id`),
  INDEX `idx_level_merchant_id` (`merchant_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
