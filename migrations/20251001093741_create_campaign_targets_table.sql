-- Create "campaign_targets" table
CREATE TABLE `campaign_targets` (
  `campaign_id` bigint unsigned NOT NULL,
  `target_type` varchar(20) NOT NULL,
  `target_id` bigint unsigned NULL,
  `created_at` datetime NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX `idx_campaign_id` (`campaign_id`),
  INDEX `idx_target_type_id` (`target_type`, `target_id`)
) CHARSET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
